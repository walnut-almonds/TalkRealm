package service

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrMessageNotFound     = errors.New("message not found")
	ErrNotChannelMemberMsg = errors.New("not a member of this channel's guild")
	ErrNotMessageOwner     = errors.New("not the owner of this message")
	ErrEmptyMessageContent = errors.New("message content cannot be empty")
	ErrInvalidMessageType  = errors.New("invalid message type")
	ErrDuplicateNonce      = errors.New("duplicate nonce: message already exists")
)

// WebSocketManager 定義 WebSocket 管理器的介面（避免循環依賴）
type WebSocketManager interface {
	BroadcastToChannel(channelID uint, msgType string, data any)
}

// MessageService 訊息服務介面
type MessageService interface {
	CreateMessage(userID uint, req *CreateMessageRequest) (*model.Message, error)
	GetMessage(messageID, userID uint) (*model.Message, error)
	ListChannelMessages(
		channelID, userID uint,
		limit int,
		before uint,
	) (*MessageListResponse, error)
	UpdateMessage(messageID, userID uint, req *UpdateMessageRequest) (*model.Message, error)
	DeleteMessage(messageID, userID uint) error
	SetWebSocketManager(manager WebSocketManager)
	SetFileService(fs FileService)
	// CreateMessageWS 提供給 WebSocket send_message op 的薄包裝
	CreateMessageWS(userID, channelID uint, content, contentType, nonce string) (any, error)
}

type messageService struct {
	messageRepo     repository.MessageRepository
	channelRepo     repository.ChannelRepository
	guildMemberRepo repository.GuildMemberRepository
	wsManager       WebSocketManager
	fileService     FileService // 可選，用於建立附件關聯
}

// NewMessageService 建立訊息服務實例
func NewMessageService(
	messageRepo repository.MessageRepository,
	channelRepo repository.ChannelRepository,
	guildMemberRepo repository.GuildMemberRepository,
) MessageService {
	return &messageService{
		messageRepo:     messageRepo,
		channelRepo:     channelRepo,
		guildMemberRepo: guildMemberRepo,
		wsManager:       nil,
		fileService:     nil, // 稍後透過 SetFileService 設定
	}
}

// SetFileService 設定 FileService（用於建立附件關聯）
func (s *messageService) SetFileService(fs FileService) {
	s.fileService = fs
}

// SetWebSocketManager 設定 WebSocket 管理器
func (s *messageService) SetWebSocketManager(manager WebSocketManager) {
	s.wsManager = manager
}

// CreateMessageWS 給 WebSocket send_message op 使用的薄包裝
func (s *messageService) CreateMessageWS(
	userID, channelID uint,
	content, contentType, nonce string,
) (any, error) {
	return s.CreateMessage(userID, &CreateMessageRequest{
		ChannelID: channelID,
		Content:   content,
		Type:      contentType,
		Nonce:     nonce,
	})
}

// CreateMessageRequest 建立訊息請求
type CreateMessageRequest struct {
	ChannelID uint   `json:"channel_id"`
	Content   string `json:"content"`
	Type      string `json:"type"`     // text, image, file (預設: text)
	Nonce     string `json:"nonce"`    // client 產生的冪等 key（可選，建議 UUID v4）
	FileIDs   []uint `json:"file_ids"` // 附加的已確認檔案 ID（可選）
}

// UpdateMessageRequest 更新訊息請求
type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// MessageListResponse 訊息列表回應
type MessageListResponse struct {
	Messages []*model.Message `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

// CreateMessage 建立訊息
func (s *messageService) CreateMessage(
	userID uint,
	req *CreateMessageRequest,
) (*model.Message, error) {
	// 驗證訊息內容
	if req.Content == "" {
		return nil, ErrEmptyMessageContent
	}

	// 驗證訊息類型
	msgType := req.Type
	if msgType == "" {
		msgType = "text"
	}

	if msgType != "text" && msgType != "image" && msgType != "file" {
		return nil, ErrInvalidMessageType
	}

	// 冪等去重：若 client 提供 nonce，先查 DB 是否已存在
	if req.Nonce != "" {
		existing, err := s.messageRepo.GetByNonce(userID, req.Nonce)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if existing != nil {
			// 已存在：直接返回原訊息，不重複寫入
			return existing, nil
		}
	}

	// 檢查頻道是否存在
	channel, err := s.channelRepo.GetByID(req.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	// 檢查使用者是否為該社群成員
	member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
	if err != nil || member == nil {
		return nil, ErrNotChannelMemberMsg
	}

	// 建立訊息
	message := &model.Message{
		ChannelID: req.ChannelID,
		UserID:    userID,
		Content:   req.Content,
		Type:      msgType,
		Nonce:     req.Nonce,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// 建立附件關聯
	if s.fileService != nil && len(req.FileIDs) > 0 {
		for _, fid := range req.FileIDs {
			if _, err := s.fileService.AttachToMessage(message.ID, fid); err != nil {
				// 附件關聯失敗不中斷訊息發送，僅記錄
				_ = err
			}
		}
	}

	// 重新取得訊息（包含關聯資料）
	fullMessage, err := s.messageRepo.GetByID(message.ID)
	if err != nil {
		return nil, err
	}

	// 如果有 WebSocket 管理器，即時推送新訊息
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(req.ChannelID, "message_create", fullMessage)
	}

	return fullMessage, nil
}

// GetMessage 取得訊息
func (s *messageService) GetMessage(messageID, userID uint) (*model.Message, error) {
	// 取得訊息
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// 檢查使用者是否為該社群成員
	channel, err := s.channelRepo.GetByID(message.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
	if err != nil || member == nil {
		return nil, ErrNotChannelMemberMsg
	}

	return message, nil
}

// ListChannelMessages 列出頻道的訊息（cursor-based 分頁）
func (s *messageService) ListChannelMessages(
	channelID, userID uint,
	limit int,
	before uint,
) (*MessageListResponse, error) {
	// 檢查頻道是否存在
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	// 檢查使用者是否為該社群成員
	member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
	if err != nil || member == nil {
		return nil, ErrNotChannelMemberMsg
	}

	if limit < 1 || limit > 100 {
		limit = 50
	}

	// 多取一筆用於判斷 has_more
	messages, err := s.messageRepo.GetByChannelIDCursor(channelID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	return &MessageListResponse{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}

// UpdateMessage 更新訊息
func (s *messageService) UpdateMessage(
	messageID, userID uint,
	req *UpdateMessageRequest,
) (*model.Message, error) {
	// 驗證訊息內容
	if req.Content == "" {
		return nil, ErrEmptyMessageContent
	}

	// 取得訊息
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// 檢查是否為訊息擁有者
	if message.UserID != userID {
		return nil, ErrNotMessageOwner
	}

	// 更新訊息
	message.Content = req.Content
	message.IsEdited = true
	message.UpdatedAt = time.Now()

	if err := s.messageRepo.Update(message); err != nil {
		return nil, err
	}

	// 重新取得訊息（包含關聯資料）
	updated, err := s.messageRepo.GetByID(message.ID)
	if err != nil {
		return nil, err
	}

	// 廣播訊息更新事件
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(message.ChannelID, "message_update", updated)
	}

	return updated, nil
}

// DeleteMessage 刪除訊息
func (s *messageService) DeleteMessage(messageID, userID uint) error {
	// 取得訊息
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return ErrMessageNotFound
	}

	// 檢查是否為訊息擁有者或社群管理員
	if message.UserID != userID {
		// 檢查是否為社群管理員
		channel, err := s.channelRepo.GetByID(message.ChannelID)
		if err != nil {
			return errors.New("channel not found")
		}

		member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
		if err != nil || member == nil {
			return ErrNotChannelMemberMsg
		}

		// 只有擁有者或管理員可以刪除他人訊息
		if member.Role != "owner" && member.Role != "admin" {
			return ErrNotMessageOwner
		}
	}

	// 刪除訊息
	if err := s.messageRepo.Delete(messageID); err != nil {
		return err
	}

	// 廣播訊息刪除事件
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(message.ChannelID, "message_delete", map[string]any{
			"message_id": messageID,
			"channel_id": message.ChannelID,
		})
	}

	return nil
}
