package service

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrMessageNotFound     = errors.New("message not found")
	ErrNotChannelMemberMsg = errors.New("not a member of this channel's guild")
	ErrNotMessageOwner     = errors.New("not the owner of this message")
	ErrEmptyMessageContent = errors.New("message content cannot be empty")
	ErrInvalidMessageType  = errors.New("invalid message type")
)

// WebSocketManager 定義 WebSocket 管理器的介面（避免循環依賴）
type WebSocketManager interface {
	BroadcastToChannel(channelID uint, msgType string, data interface{})
}

// MessageService 訊息服務介面
type MessageService interface {
	CreateMessage(userID uint, req *CreateMessageRequest) (*model.Message, error)
	GetMessage(messageID, userID uint) (*model.Message, error)
	ListChannelMessages(channelID, userID uint, page, pageSize int) (*MessageListResponse, error)
	UpdateMessage(messageID, userID uint, req *UpdateMessageRequest) (*model.Message, error)
	DeleteMessage(messageID, userID uint) error
	SetWebSocketManager(manager WebSocketManager)
}

type messageService struct {
	messageRepo     repository.MessageRepository
	channelRepo     repository.ChannelRepository
	guildMemberRepo repository.GuildMemberRepository
	wsManager       WebSocketManager
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
		wsManager:       nil, // 稍後設定
	}
}

// SetWebSocketManager 設定 WebSocket 管理器
func (s *messageService) SetWebSocketManager(manager WebSocketManager) {
	s.wsManager = manager
}

// CreateMessageRequest 建立訊息請求
type CreateMessageRequest struct {
	ChannelID uint   `json:"channel_id" binding:"required"`
	Content   string `json:"content"    binding:"required"`
	Type      string `json:"type"` // text, image, file (預設: text)
}

// UpdateMessageRequest 更新訊息請求
type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// MessageListResponse 訊息列表回應
type MessageListResponse struct {
	Messages   []*model.Message `json:"messages"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// 重新取得訊息（包含關聯資料）
	fullMessage, err := s.messageRepo.GetByID(message.ID)
	if err != nil {
		return nil, err
	}

	// 如果有 WebSocket 管理器，即時推送新訊息
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(req.ChannelID, "new_message", fullMessage)
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

// ListChannelMessages 列出頻道的訊息
func (s *messageService) ListChannelMessages(
	channelID, userID uint,
	page, pageSize int,
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

	// 設定預設分頁參數
	if page < 1 {
		page = 1
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// 計算偏移量
	offset := (page - 1) * pageSize

	// 取得訊息列表
	messages, err := s.messageRepo.GetByChannelID(channelID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	// 計算總頁數（這裡簡化處理，實際應該查詢總數）
	// TODO: 新增 CountByChannelID 方法到 repository
	totalPages := 1
	if len(messages) == pageSize {
		totalPages = page + 1 // 如果有完整一頁，假設還有下一頁
	}

	return &MessageListResponse{
		Messages:   messages,
		Total:      len(messages),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
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
	message.UpdatedAt = time.Now()

	if err := s.messageRepo.Update(message); err != nil {
		return nil, err
	}

	// 重新取得訊息（包含關聯資料）
	return s.messageRepo.GetByID(message.ID)
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
	return s.messageRepo.Delete(messageID)
}
