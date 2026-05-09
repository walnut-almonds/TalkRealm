package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait 允許等待的寫入時間
	writeWait = 10 * time.Second

	// pongWait 允許從對等方讀取下一個 pong 消息的時間
	pongWait = 60 * time.Second

	// pingPeriod 在此期間向對等方發送 ping，必須小於 pongWait
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize 對等方允許的最大消息大小
	maxMessageSize = 512 * 1024 // 512KB

	// heartbeatInterval 應用層心跳間隔（毫秒）
	heartbeatInterval = 30_000
)

// IncomingMessage 從 client 接收的訊息
type IncomingMessage struct {
	Op   string          `json:"op"`
	Data json.RawMessage `json:"d,omitempty"`
}

// OutgoingMessage 發送給 client 的訊息
type OutgoingMessage struct {
	Op        string `json:"op"`
	Data      any    `json:"d,omitempty"`
	Timestamp int64  `json:"t"`
}

// Message 為向後相容保留的型別別名
type Message = OutgoingMessage

// Client 代表單個 WebSocket 客戶端連接
type Client struct {
	// WebSocket 連接
	conn *websocket.Conn

	// 管理器引用
	manager *Manager

	// 使用者 ID（identify 後設定）
	userID uint

	// 使用者名稱（identify 後設定）
	username string

	// 是否已通過 identify 認證
	identified bool

	// 訂閱的頻道 ID 集合
	channels map[uint]bool

	// 訂閱的 guild ID 集合
	guilds map[uint]bool

	// 緩衝通道，用於發送消息
	send chan []byte
}

// NewClient 創建新的客戶端（連線建立時不需傳入使用者資訊，待 identify 後設定）
func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		conn:     conn,
		manager:  manager,
		channels: make(map[uint]bool),
		guilds:   make(map[uint]bool),
		send:     make(chan []byte, 256),
	}
}

// readPump 從 WebSocket 連接讀取消息
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c

		c.conn.Close() //nolint:errcheck,gosec
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck,gosec
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck,gosec
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("websocket error: %v", err)
			}

			break
		}

		var msg IncomingMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("error parsing incoming message: %v", err)
			continue
		}

		c.handleMessage(&msg)
	}
}

// writePump 將消息從管理器發送到 WebSocket 連接
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close() //nolint:errcheck,gosec
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))    //nolint:errcheck,gosec
				c.conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck,gosec

				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck,gosec

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// 將排隊中的訊息逐一以獨立 frame 發送，避免多個 JSON 合併在同一 frame 導致解析失敗
			n := len(c.send)
			for i := 0; i < n; i++ {
				c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck,gosec

				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					return
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck,gosec

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 處理從客戶端接收的訊息
func (c *Client) handleMessage(msg *IncomingMessage) {
	// identify 和 heartbeat 在認證前均可處理
	switch msg.Op {
	case "identify":
		c.handleIdentify(msg.Data)
		return
	case "heartbeat":
		c.sendJSON(OutgoingMessage{
			Op:        "heartbeat_ack",
			Data:      map[string]int64{"timestamp": time.Now().UnixMilli()},
			Timestamp: time.Now().UnixMilli(),
		})
		// 已認證的 client 收到心跳時，刷新 Redis TTL，維持 online 狀態
		if c.identified {
			go c.manager.redisRefreshHeartbeat(c.userID)
		}

		return
	}

	// 未 identify 的 client 不允許其他操作
	if !c.identified {
		c.sendJSON(OutgoingMessage{
			Op:        "error",
			Data:      map[string]string{"message": "not identified"},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	switch msg.Op {
	case "subscribe":
		var payload struct {
			ChannelID uint `json:"channel_id"`
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.ChannelID == 0 {
			log.Printf("invalid subscribe payload from user %s", c.username)
			return
		}

		c.manager.SubscribeToChannel(c, payload.ChannelID)

	case "unsubscribe":
		var payload struct {
			ChannelID uint `json:"channel_id"`
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.ChannelID == 0 {
			return
		}

		c.manager.UnsubscribeFromChannel(c, payload.ChannelID)

	case "send_message":
		c.handleSendMessage(msg.Data)

	case "typing_start":
		var payload struct {
			ChannelID uint `json:"channel_id"`
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil || payload.ChannelID == 0 {
			return
		}

		c.manager.BroadcastToChannelExcept(c, payload.ChannelID, "typing_start", map[string]any{
			"channel_id": payload.ChannelID,
			"user_id":    c.userID,
			"username":   c.username,
		})

	case "voice_state_update":
		c.handleVoiceStateUpdate(msg.Data)

	default:
		log.Printf("unknown op '%s' from user %s", msg.Op, c.username)
	}
}

// handleSendMessage 處理 send_message op：建立訊息並廣播
func (c *Client) handleSendMessage(raw json.RawMessage) {
	var payload struct {
		ChannelID   uint   `json:"channel_id"`
		Content     string `json:"content"`
		ContentType string `json:"type"`
		Nonce       string `json:"nonce"`
		FileIDs     []uint `json:"file_ids"`
	}

	if err := json.Unmarshal(raw, &payload); err != nil || payload.ChannelID == 0 ||
		(payload.Content == "" && len(payload.FileIDs) == 0) {
		c.sendJSON(OutgoingMessage{
			Op: "error",
			Data: map[string]string{
				"message": "invalid send_message payload: channel_id and content or file_ids required",
			},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	if !c.manager.CheckRateLimit(c.userID, 10) {
		c.sendJSON(OutgoingMessage{
			Op:        "error",
			Data:      map[string]string{"message": "rate limit exceeded; please slow down"},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	if c.manager.msgSender == nil {
		c.sendJSON(OutgoingMessage{
			Op:        "error",
			Data:      map[string]string{"message": "messaging not available"},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	contentType := payload.ContentType
	if contentType == "" {
		contentType = "text"
	}

	if _, err := c.manager.msgSender.CreateMessageWS(
		c.userID, payload.ChannelID, payload.Content, contentType, payload.Nonce, payload.FileIDs,
	); err != nil {
		c.sendJSON(OutgoingMessage{
			Op:        "error",
			Data:      map[string]string{"message": err.Error()},
			Timestamp: time.Now().UnixMilli(),
		})
	}
}

// handleVoiceStateUpdate 處理 voice_state_update op：廣播使用者加入/離開語音頻道
func (c *Client) handleVoiceStateUpdate(raw json.RawMessage) {
	var payload struct {
		ChannelID uint   `json:"channel_id"`
		Action    string `json:"action"` // "join" | "leave"
	}

	if err := json.Unmarshal(raw, &payload); err != nil || payload.ChannelID == 0 {
		c.sendJSON(OutgoingMessage{
			Op: "error",
			Data: map[string]string{
				"message": "invalid voice_state_update payload: channel_id required",
			},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	if payload.Action != "join" && payload.Action != "leave" {
		payload.Action = "join"
	}

	c.manager.BroadcastToChannel(payload.ChannelID, "voice_state_update", map[string]any{
		"channel_id": payload.ChannelID,
		"user_id":    c.userID,
		"username":   c.username,
		"action":     payload.Action,
	})
}

// handleIdentify 處理 identify op：驗證 JWT 並回應 ready
func (c *Client) handleIdentify(raw json.RawMessage) {
	if c.identified {
		return // 防止重複 identify
	}

	var payload struct {
		Token    string `json:"token"`
		Channels []uint `json:"channels,omitempty"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Token == "" {
		c.sendJSON(OutgoingMessage{
			Op:        "error",
			Data:      map[string]string{"message": "invalid identify payload"},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	claims, err := c.manager.jwtManager.ValidateToken(payload.Token)
	if err != nil {
		c.sendJSON(OutgoingMessage{
			Op:        "error",
			Data:      map[string]string{"message": "invalid or expired token"},
			Timestamp: time.Now().UnixMilli(),
		})

		return
	}

	// 設定 client 身份
	c.userID = claims.UserID
	c.username = claims.Username
	c.identified = true

	log.Printf("Client identified: User %s (ID: %d)", c.username, c.userID)

	// 訂閱初始頻道列表
	for _, channelID := range payload.Channels {
		c.manager.SubscribeToChannel(c, channelID)
	}

	// 訂閱所屬的所有 guild（非同步 DB 查詢，不阻塞 readPump）
	go c.manager.SubscribeClientToUserGuilds(c)

	// 回應 ready
	c.sendJSON(OutgoingMessage{
		Op: "ready",
		Data: map[string]any{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"email":    claims.Email,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// 廣播上線狀態
	c.manager.broadcastPresenceUpdate(c.userID, c.username, "online")

	// Redis：記錄 user server mapping 及 guild online set
	go c.manager.redisOnIdentify(c.userID)
}

// sendHello 向剛連線的 client 發送 hello 訊息
func (c *Client) sendHello() {
	c.sendJSON(OutgoingMessage{
		Op: "hello",
		Data: map[string]any{
			"heartbeat_interval": heartbeatInterval,
		},
		Timestamp: time.Now().UnixMilli(),
	})
}

// sendJSON 序列化並發送 OutgoingMessage
func (c *Client) sendJSON(msg OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling outgoing message: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Printf("send buffer full for client %s, dropping message", c.username)
	}
}

// SendMessage 發送原始位元組給客戶端
func (c *Client) SendMessage(data []byte) {
	select {
	case c.send <- data:
	default:
		close(c.send)

		c.manager.unregister <- c
	}
}

// IsSubscribed 檢查客戶端是否訂閱了指定頻道
func (c *Client) IsSubscribed(channelID uint) bool {
	return c.channels[channelID]
}
