package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/walnut-almonds/talkrealm/pkg/auth"
)

// Manager 管理所有 WebSocket 連接
type Manager struct {
	// 所有已連線的客戶端
	clients map[*Client]bool

	// channelSubscriptions 頻道訂閱索引：channelID -> 訂閱該頻道的 clients（O(1) 查找）
	channelSubscriptions map[uint]map[*Client]bool

	// 從客戶端接收的廣播消息
	broadcast chan []byte

	// 從客戶端註冊請求
	register chan *Client

	// 從客戶端取消註冊請求
	unregister chan *Client

	// 互斥鎖保護 clients 與 channelSubscriptions
	mu sync.RWMutex

	// jwtManager 用於 identify op 的 token 驗證
	jwtManager *auth.JWTManager
}

// NewManager 創建新的 WebSocket 管理器
func NewManager(jwtManager *auth.JWTManager) *Manager {
	return &Manager{
		clients:              make(map[*Client]bool),
		channelSubscriptions: make(map[uint]map[*Client]bool),
		broadcast:            make(chan []byte, 256),
		register:             make(chan *Client),
		unregister:           make(chan *Client),
		jwtManager:           jwtManager,
	}
}

// Run 運行管理器的主循環
func (m *Manager) Run() {
	log.Println("WebSocket Manager started")
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client] = true
			m.mu.Unlock()
			log.Printf("Client connected (pending identify). Total clients: %d", len(m.clients))
			// 發送 hello，告知 client 心跳間隔
			client.sendHello()

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				// 從所有頻道訂閱中移除
				for channelID := range client.channels {
					if subscribers, ok := m.channelSubscriptions[channelID]; ok {
						delete(subscribers, client)
					}
				}
				delete(m.clients, client)
				close(client.send)
				if client.identified {
					log.Printf("Client disconnected: User %s (ID: %d). Total clients: %d",
						client.username, client.userID, len(m.clients))
				} else {
					log.Printf("Unauthenticated client disconnected. Total clients: %d", len(m.clients))
				}
			}
			wasIdentified := client.identified
			userID := client.userID
			username := client.username
			m.mu.Unlock()

			// 廣播下線狀態（在鎖外執行，避免死鎖）
			if wasIdentified {
				m.broadcastPresenceUpdate(userID, username, "offline")
			}

		case message := <-m.broadcast:
			m.mu.RLock()
			for client := range m.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(m.clients, client)
				}
			}
			m.mu.RUnlock()
		}
	}
}

// RegisterClient 註冊新客戶端，啟動讀寫 goroutines
func (m *Manager) RegisterClient(client *Client) {
	m.register <- client
	go client.writePump()
	go client.readPump()
}

// SubscribeToChannel 將 client 加入頻道訂閱索引
func (m *Manager) SubscribeToChannel(client *Client, channelID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	client.channels[channelID] = true
	if m.channelSubscriptions[channelID] == nil {
		m.channelSubscriptions[channelID] = make(map[*Client]bool)
	}
	m.channelSubscriptions[channelID][client] = true
	log.Printf("User %s subscribed to channel %d", client.username, channelID)
}

// UnsubscribeFromChannel 將 client 從頻道訂閱索引中移除
func (m *Manager) UnsubscribeFromChannel(client *Client, channelID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(client.channels, channelID)
	if subscribers, ok := m.channelSubscriptions[channelID]; ok {
		delete(subscribers, client)
	}
	log.Printf("User %s unsubscribed from channel %d", client.username, channelID)
}

// BroadcastToChannel 向訂閱了指定頻道的所有客戶端廣播消息（使用 O(1) 索引查找）
func (m *Manager) BroadcastToChannel(channelID uint, msgType string, data interface{}) {
	message := OutgoingMessage{
		Op:        msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	subscribers, ok := m.channelSubscriptions[channelID]
	if !ok {
		return
	}

	count := 0
	for client := range subscribers {
		select {
		case client.send <- messageBytes:
			count++
		default:
			log.Printf("Failed to send message to client %s (buffer full)", client.username)
		}
	}
	log.Printf("Broadcasted %s to channel %d: %d clients", msgType, channelID, count)
}

// BroadcastToChannelExcept 向頻道訂閱者廣播，但排除指定 client（用於 typing_start）
func (m *Manager) BroadcastToChannelExcept(exclude *Client, channelID uint, msgType string, data interface{}) {
	message := OutgoingMessage{
		Op:        msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	subscribers, ok := m.channelSubscriptions[channelID]
	if !ok {
		return
	}

	count := 0
	for client := range subscribers {
		if client == exclude {
			continue
		}
		select {
		case client.send <- messageBytes:
			count++
		default:
			log.Printf("Failed to send message to client %s (buffer full)", client.username)
		}
	}
	log.Printf("Broadcasted %s to channel %d (excl. sender): %d clients", msgType, channelID, count)
}

// broadcastPresenceUpdate 向所有已認證的 client 廣播使用者在線狀態變更
func (m *Manager) broadcastPresenceUpdate(userID uint, username, status string) {
	message := OutgoingMessage{
		Op: "presence_update",
		Data: map[string]any{
			"user_id":  userID,
			"username": username,
			"status":   status,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if client.identified {
			select {
			case client.send <- messageBytes:
			default:
			}
		}
	}
}

// BroadcastToAll 向所有連接的客戶端廣播消息
func (m *Manager) BroadcastToAll(msgType string, data interface{}) {
	message := OutgoingMessage{
		Op:        msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	m.broadcast <- messageBytes
}

// BroadcastToUser 向指定使用者發送消息
func (m *Manager) BroadcastToUser(userID uint, msgType string, data interface{}) {
	message := OutgoingMessage{
		Op:        msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if client.userID == userID {
			select {
			case client.send <- messageBytes:
				log.Printf("Sent %s message to user %d", msgType, userID)
			default:
				log.Printf("Failed to send message to user %d (buffer full)", userID)
			}
		}
	}
}
