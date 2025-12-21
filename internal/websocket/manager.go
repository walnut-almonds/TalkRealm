package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

// Manager 管理所有 WebSocket 連接
type Manager struct {
	// 註冊的客戶端
	clients map[*Client]bool

	// 從客戶端接收的廣播消息
	broadcast chan []byte

	// 從客戶端註冊請求
	register chan *Client

	// 從客戶端取消註冊請求
	unregister chan *Client

	// 互斥鎖保護客戶端映射
	mu sync.RWMutex
}

// NewManager 創建新的 WebSocket 管理器
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
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
			log.Printf("Client registered: User %s (ID: %d). Total clients: %d",
				client.username, client.userID, len(m.clients))

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.send)
				log.Printf("Client unregistered: User %s (ID: %d). Total clients: %d",
					client.username, client.userID, len(m.clients))
			}
			m.mu.Unlock()

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

// RegisterClient 註冊新客戶端
func (m *Manager) RegisterClient(client *Client) {
	m.register <- client
	// 啟動客戶端的讀寫 goroutines
	go client.writePump()
	go client.readPump()
}

// BroadcastToChannel 向訂閱了指定頻道的所有客戶端廣播消息
func (m *Manager) BroadcastToChannel(channelID uint, msgType string, data interface{}) {
	message := Message{
		Type:      msgType,
		ChannelID: channelID,
		Data:      data,
		Timestamp: 0, // 將由前端設置或使用當前時間
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for client := range m.clients {
		// 只發送給訂閱了該頻道的客戶端
		if client.IsSubscribed(channelID) {
			select {
			case client.send <- messageBytes:
				count++
			default:
				// 客戶端的發送通道已滿，跳過
				log.Printf("Failed to send message to client %s (buffer full)", client.username)
			}
		}
	}

	log.Printf("Broadcasted %s message to channel %d: %d clients", msgType, channelID, count)
}

// BroadcastToAll 向所有連接的客戶端廣播消息
func (m *Manager) BroadcastToAll(msgType string, data interface{}) {
	message := Message{
		Type:      msgType,
		Data:      data,
		Timestamp: 0,
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
	message := Message{
		Type:      msgType,
		Data:      data,
		Timestamp: 0,
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

// GetConnectedClients 獲取當前連接的客戶端數量
func (m *Manager) GetConnectedClients() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetChannelSubscribers 獲取訂閱了指定頻道的客戶端數量
func (m *Manager) GetChannelSubscribers(channelID uint) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for client := range m.clients {
		if client.IsSubscribed(channelID) {
			count++
		}
	}
	return count
}
