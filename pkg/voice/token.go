// Package voice 封裝 LiveKit token 生成邏輯
package voice

import (
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/walnut-almonds/talkrealm/pkg/config"
)

// Manager 負責 LiveKit token 生成
type Manager struct {
	cfg *config.LiveKitConfig
}

// NewManager 建立新的 Voice Manager
func NewManager(cfg *config.LiveKitConfig) *Manager {
	return &Manager{cfg: cfg}
}

// IsConfigured 判斷 LiveKit 是否已設定（api_key 與 api_secret 皆非空）
func (m *Manager) IsConfigured() bool {
	return m.cfg.APIKey != "" && m.cfg.APISecret != ""
}

// RoomTokenResponse 包含 LiveKit Room token 及連線 URL
type RoomTokenResponse struct {
	Token    string `json:"token"`
	URL      string `json:"url"`
	RoomName string `json:"room_name"`
	Identity string `json:"identity"`
}

// GenerateRoomToken 為指定 channel 與使用者生成 LiveKit Room Join token
// roomName 以 "channel:{channelID}" 命名，確保各頻道對應獨立 room
func (m *Manager) GenerateRoomToken(
	channelID, userID uint,
	username string,
) (*RoomTokenResponse, error) {
	if !m.IsConfigured() {
		return nil, fmt.Errorf("livekit not configured: api_key and api_secret are required")
	}

	roomName := fmt.Sprintf("channel:%d", channelID)
	identity := fmt.Sprintf("%d", userID)

	ttl := time.Duration(m.cfg.TokenTTL) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}

	at := auth.NewAccessToken(m.cfg.APIKey, m.cfg.APISecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(username).
		SetValidFor(ttl)

	token, err := at.ToJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate livekit token: %w", err)
	}

	publicURL := m.cfg.PublicURL
	if publicURL == "" {
		publicURL = m.cfg.URL
	}

	return &RoomTokenResponse{
		Token:    token,
		URL:      publicURL,
		RoomName: roomName,
		Identity: identity,
	}, nil
}
