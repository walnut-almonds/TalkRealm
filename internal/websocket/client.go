package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 允許等待的寫入時間
	writeWait = 10 * time.Second

	// 允許從對等方讀取下一個 pong 消息的時間
	pongWait = 60 * time.Second

	// 在此期間向對等方發送 ping。必須小於 pongWait
	pingPeriod = (pongWait * 9) / 10

	// 對等方允許的最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

// Client 代表單個 WebSocket 客戶端連接
type Client struct {
	// WebSocket 連接
	conn *websocket.Conn

	// 管理器引用
	manager *Manager

	// 使用者 ID
	userID uint

	// 使用者名稱
	username string

	// 訂閱的頻道 ID 列表
	channels map[uint]bool

	// 緩衝通道，用於發送消息
	send chan []byte
}

// Message 代表 WebSocket 消息
type Message struct {
	Type      string `json:"type"`
	ChannelID uint   `json:"channel_id,omitempty"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// NewClient 創建新的客戶端
func NewClient(conn *websocket.Conn, manager *Manager, userID uint, username string) *Client {
	return &Client{
		conn:     conn,
		manager:  manager,
		userID:   userID,
		username: username,
		channels: make(map[uint]bool),
		send:     make(chan []byte, 256),
	}
}

// readPump 從 WebSocket 連接讀取消息並發送到管理器
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
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

		// 解析消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			continue
		}

		// 處理客戶端消息（例如訂閱/取消訂閱頻道）
		c.handleMessage(&msg)
	}
}

// writePump 將消息從管理器發送到 WebSocket 連接
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 管理器關閉了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 將排隊的消息添加到當前 WebSocket 消息中
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 處理從客戶端接收的消息
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "subscribe":
		// 訂閱頻道
		if msg.ChannelID > 0 {
			c.channels[msg.ChannelID] = true
			log.Printf("User %s subscribed to channel %d", c.username, msg.ChannelID)
		}

	case "unsubscribe":
		// 取消訂閱頻道
		if msg.ChannelID > 0 {
			delete(c.channels, msg.ChannelID)
			log.Printf("User %s unsubscribed from channel %d", c.username, msg.ChannelID)
		}

	case "ping":
		// 回應 pong
		response := Message{
			Type:      "pong",
			Timestamp: time.Now().Unix(),
		}
		if data, err := json.Marshal(response); err == nil {
			c.send <- data
		}

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// SendMessage 發送消息給客戶端
func (c *Client) SendMessage(data []byte) {
	select {
	case c.send <- data:
	default:
		// 如果發送緩衝區已滿，關閉客戶端
		close(c.send)
		c.manager.unregister <- c
	}
}

// IsSubscribed 檢查客戶端是否訂閱了指定頻道
func (c *Client) IsSubscribed(channelID uint) bool {
	return c.channels[channelID]
}
