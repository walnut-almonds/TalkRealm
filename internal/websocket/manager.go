package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
)

// GuildMemberLookup 提供查詢使用者所屬 guild IDs 的介面（避免 websocket package 直接相依 repository）
type GuildMemberLookup interface {
	GetUserGuildIDs(userID uint) ([]uint, error)
}

// MessageSender 訊息建立介面（避免循環依賴）
// 由 MessageService 實作，注入 Manager 後供 send_message op 使用
type MessageSender interface {
	CreateMessageWS(userID, channelID uint, content, contentType, nonce string, fileIDs []uint) (any, error)
}

// serverID 用於 Redis user server mapping（單體架構下固定為 "1"）
const serverID = "1"

// Manager 管理所有 WebSocket 連接
type Manager struct {
	// 所有已連線的客戶端
	clients map[*Client]bool

	// channelSubscriptions 頻道訂閱索引：channelID -> 訂閱該頻道的 clients（O(1) 查找）
	channelSubscriptions map[uint]map[*Client]bool

	// guildSubscriptions guild 訂閱索引：guildID -> 訂閱該 guild 的 clients（O(1) 廣播 guild 事件）
	guildSubscriptions map[uint]map[*Client]bool

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

	// redisClient 用於 user server mapping 及 guild online set
	redisClient *goredis.Client

	// guildLookup 用於查詢使用者所屬 guild IDs
	guildLookup GuildMemberLookup

	// msgSender 用於 send_message op（注入 MessageService，避免循環依賴）
	msgSender MessageSender

	// voiceParticipants 追蹤目前在各語音頻道的成員（channelID → { userID → username }）
	voiceParticipants map[uint]map[uint]string
}

// NewManager 創建新的 WebSocket 管理器
func NewManager(jwtManager *auth.JWTManager) *Manager {
	return &Manager{
		clients:              make(map[*Client]bool),
		channelSubscriptions: make(map[uint]map[*Client]bool),
		guildSubscriptions:   make(map[uint]map[*Client]bool),
		broadcast:            make(chan []byte, 256),
		register:             make(chan *Client),
		unregister:           make(chan *Client),
		jwtManager:           jwtManager,
		voiceParticipants:    make(map[uint]map[uint]string),
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
				for guildID := range client.guilds {
					if subscribers, ok := m.guildSubscriptions[guildID]; ok {
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

			// Redis 清理（在鎖外執行）
			if wasIdentified {
				m.redisOnDisconnect(userID)
			}

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

// SubscribeToGuild 將 client 加入 guild 訂閱索引
func (m *Manager) SubscribeToGuild(client *Client, guildID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	client.guilds[guildID] = true
	if m.guildSubscriptions[guildID] == nil {
		m.guildSubscriptions[guildID] = make(map[*Client]bool)
	}
	m.guildSubscriptions[guildID][client] = true
}

// UnsubscribeFromGuild 將 client 從 guild 訂閱索引中移除
func (m *Manager) UnsubscribeFromGuild(client *Client, guildID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(client.guilds, guildID)
	if subscribers, ok := m.guildSubscriptions[guildID]; ok {
		delete(subscribers, client)
	}
}

// BroadcastToGuild 向訂閱了指定 guild 的所有客戶端廣播消息
func (m *Manager) BroadcastToGuild(guildID uint, msgType string, data interface{}) {
	message := OutgoingMessage{
		Op:        msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling guild message: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	subscribers, ok := m.guildSubscriptions[guildID]
	if !ok {
		return
	}

	count := 0
	for client := range subscribers {
		select {
		case client.send <- messageBytes:
			count++
		default:
			log.Printf("Failed to send guild message to client %s (buffer full)", client.username)
		}
	}
	log.Printf("Broadcasted %s to guild %d: %d clients", msgType, guildID, count)
}

// SubscribeClientToUserGuilds 將 client 訂閱至其所有 guild（由 guildLookup 查詢）
func (m *Manager) SubscribeClientToUserGuilds(client *Client) {
	if m.guildLookup == nil {
		return
	}
	guildIDs, err := m.guildLookup.GetUserGuildIDs(client.userID)
	if err != nil {
		log.Printf("ws: failed to get guild IDs for user %d: %v", client.userID, err)
		return
	}
	for _, gid := range guildIDs {
		m.SubscribeToGuild(client, gid)
	}
	log.Printf("User %s subscribed to %d guilds", client.username, len(guildIDs))
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

// SetRedis 注入 Redis client（可選；未設定時跳過 Redis 操作）
func (m *Manager) SetRedis(rdb *goredis.Client) {
	m.redisClient = rdb
}

// SetGuildLookup 注入 guild 成員查詢介面
func (m *Manager) SetGuildLookup(l GuildMemberLookup) {
	m.guildLookup = l
}

// SetMessageSender 注入訊息建立器（供 send_message op 使用）
func (m *Manager) SetMessageSender(s MessageSender) {
	m.msgSender = s
}

// CheckRateLimit 檢查使用者是否超出速率限制（每秒 maxMsg 則）
// 回傳 true 表示允許，false 表示超限
func (m *Manager) CheckRateLimit(userID uint, maxMsg int) bool {
	if m.redisClient == nil {
		return true // 無 Redis 時放行
	}
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:%d:ws_msg", userID)
	pipe := m.redisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return true // Redis 錯誤時放行
	}
	return incr.Val() <= int64(maxMsg)
}

// redisOnIdentify 使用者上線時寫入 Redis：user server mapping + guild online set
func (m *Manager) redisOnIdentify(userID uint) {
	if m.redisClient == nil {
		return
	}
	ctx := context.Background()

	// SET user:{userID}:server {serverID} EX 86400
	key := fmt.Sprintf("user:%d:server", userID)
	if err := m.redisClient.Set(ctx, key, serverID, 86400e9).Err(); err != nil {
		log.Printf("redis: set user server mapping failed: %v", err)
	}

	// SADD guild:{guildID}:online {userID}
	if m.guildLookup != nil {
		guildIDs, err := m.guildLookup.GetUserGuildIDs(userID)
		if err != nil {
			log.Printf("redis: fetch guild IDs for user %d failed: %v", userID, err)
			return
		}
		pipe := m.redisClient.Pipeline()
		for _, gid := range guildIDs {
			guildKey := fmt.Sprintf("guild:%d:online", gid)
			pipe.SAdd(ctx, guildKey, userID)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("redis: sadd guild online failed: %v", err)
		}
	}
}

// redisOnDisconnect 使用者下線時清理 Redis：刪除 user server mapping + 從 guild online set 移除
func (m *Manager) redisOnDisconnect(userID uint) {
	if m.redisClient == nil {
		return
	}
	ctx := context.Background()

	// DEL user:{userID}:server
	key := fmt.Sprintf("user:%d:server", userID)
	if err := m.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("redis: del user server mapping failed: %v", err)
	}

	// SREM guild:{guildID}:online {userID}
	if m.guildLookup != nil {
		guildIDs, err := m.guildLookup.GetUserGuildIDs(userID)
		if err != nil {
			log.Printf("redis: fetch guild IDs for user %d failed: %v", userID, err)
			return
		}
		pipe := m.redisClient.Pipeline()
		for _, gid := range guildIDs {
			guildKey := fmt.Sprintf("guild:%d:online", gid)
			pipe.SRem(ctx, guildKey, userID)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("redis: srem guild online failed: %v", err)
		}
	}
}

// redisRefreshHeartbeat 收到 heartbeat 時刷新 user server mapping 的 TTL
func (m *Manager) redisRefreshHeartbeat(userID uint) {
	if m.redisClient == nil {
		return
	}
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:server", userID)
	if err := m.redisClient.Expire(ctx, key, 86400*time.Second).Err(); err != nil {
		log.Printf("redis: refresh heartbeat TTL for user %d failed: %v", userID, err)
	}
}

// IsUserOnline 檢查使用者是否在線（以 Redis key 是否存在為準）
func (m *Manager) IsUserOnline(userID uint) bool {
	if m.redisClient == nil {
		// fallback：掃描本地 clients
		m.mu.RLock()
		defer m.mu.RUnlock()
		for c := range m.clients {
			if c.identified && c.userID == userID {
				return true
			}
		}
		return false
	}
	ctx := context.Background()
	key := fmt.Sprintf("user:%d:server", userID)
	exists, err := m.redisClient.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("redis: check user %d online failed: %v", userID, err)
		return false
	}
	return exists > 0
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

// UpsertVoiceParticipant 記錄使用者加入語音頻道
func (m *Manager) UpsertVoiceParticipant(channelID, userID uint, username string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.voiceParticipants[channelID] == nil {
		m.voiceParticipants[channelID] = make(map[uint]string)
	}

	m.voiceParticipants[channelID][userID] = username
}

// RemoveVoiceParticipant 記錄使用者離開語音頻道
func (m *Manager) RemoveVoiceParticipant(channelID, userID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.voiceParticipants[channelID] != nil {
		delete(m.voiceParticipants[channelID], userID)
	}
}

// GetVoiceParticipants 回傳目前在指定語音頻道的成員（userID → username）
func (m *Manager) GetVoiceParticipants(channelID uint) map[uint]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uint]string)
	for uid, name := range m.voiceParticipants[channelID] {
		result[uid] = name
	}

	return result
}
