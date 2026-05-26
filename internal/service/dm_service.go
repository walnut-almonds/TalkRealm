package service

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrDMChannelNotFound = errors.New("dm channel not found")
	ErrDMNotParticipant  = errors.New("not a participant of this dm channel")
	ErrDMEmptyContent    = errors.New("message content cannot be empty")
	ErrDMSelfMessage     = errors.New("cannot send dm to yourself")
	ErrDMDuplicateNonce  = errors.New("duplicate nonce: dm message already exists")
)

// DMWebSocketManager DM 服務所需的 WebSocket 廣播介面
type DMWebSocketManager interface {
	BroadcastToUser(userID uint, msgType string, data any)
}

// DMService 私訊服務介面
type DMService interface {
	// GetOrCreateChannel 取得或建立與目標使用者的 DM 頻道（不能和自己開 DM）
	GetOrCreateChannel(requesterID, targetUserID uint) (*model.DirectMessageChannel, error)
	// ListDMChannels 列出使用者參與的所有 DM 頻道
	ListDMChannels(userID uint) ([]*model.DirectMessageChannel, error)
	// SendDM 發送私訊並透過 WS 推播給雙方；回傳 (any, error) 以符合 websocket.DMSender 介面
	SendDM(senderID, dmChannelID uint, content, nonce string) (any, error)
	// ListMessages 取得 DM 頻道訊息（cursor-based 分頁）
	ListMessages(userID, dmChannelID, before uint, limit int) ([]*model.DirectMessage, error)
	// SetWebSocketManager 注入 WebSocket 管理器
	SetWebSocketManager(m DMWebSocketManager)
}

type dmService struct {
	dmRepo    repository.DMRepository
	userRepo  repository.UserRepository
	wsManager DMWebSocketManager
}

// NewDMService 建立私訊服務
func NewDMService(dmRepo repository.DMRepository, userRepo repository.UserRepository) DMService {
	return &dmService{
		dmRepo:   dmRepo,
		userRepo: userRepo,
	}
}

// SetWebSocketManager 注入 WebSocket 管理器
func (s *dmService) SetWebSocketManager(m DMWebSocketManager) {
	s.wsManager = m
}

// GetOrCreateChannel 取得或建立與目標使用者的 DM 頻道
func (s *dmService) GetOrCreateChannel(requesterID, targetUserID uint) (*model.DirectMessageChannel, error) {
	if requesterID == targetUserID {
		return nil, ErrDMSelfMessage
	}

	// 確認目標使用者存在
	if _, err := s.userRepo.GetByID(targetUserID); err != nil {
		return nil, errors.New("target user not found")
	}

	return s.dmRepo.GetOrCreateChannel(requesterID, targetUserID)
}

// ListDMChannels 列出使用者的所有 DM 頻道
func (s *dmService) ListDMChannels(userID uint) ([]*model.DirectMessageChannel, error) {
	return s.dmRepo.ListChannelsForUser(userID)
}

// SendDM 發送私訊
func (s *dmService) SendDM(senderID, dmChannelID uint, content, nonce string) (any, error) {
	if content == "" {
		return nil, ErrDMEmptyContent
	}

	// 確認使用者是頻道參與者
	ok, err := s.dmRepo.IsParticipant(dmChannelID, senderID)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrDMNotParticipant
	}

	// 冪等去重：nonce 不為空時先查重
	if nonce != "" {
		existing, err := s.dmRepo.GetMessageByNonce(senderID, nonce)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if existing != nil {
			return existing, nil
		}
	}

	msg := &model.DirectMessage{
		DMChannelID: dmChannelID,
		SenderID:    senderID,
		Content:     content,
	}

	if nonce != "" {
		msg.Nonce = nonce
	}

	if err := s.dmRepo.CreateMessage(msg); err != nil {
		return nil, err
	}

	// 取得 DM 頻道以得知接收方
	ch, err := s.dmRepo.GetChannelByID(dmChannelID)
	if err != nil {
		return nil, err
	}

	// WS 推播給雙方（發送方也要收到，用於多裝置同步）
	if s.wsManager != nil {
		data := map[string]any{
			"id":            msg.ID,
			"dm_channel_id": msg.DMChannelID,
			"sender_id":     msg.SenderID,
			"sender":        msg.Sender,
			"content":       msg.Content,
			"is_edited":     msg.IsEdited,
			"nonce":         msg.Nonce,
			"created_at":    msg.CreatedAt,
		}
		s.wsManager.BroadcastToUser(ch.User1ID, "dm_message_create", data)
		s.wsManager.BroadcastToUser(ch.User2ID, "dm_message_create", data)
	}

	return msg, nil
}

// ListMessages 取得 DM 頻道訊息
func (s *dmService) ListMessages(userID, dmChannelID, before uint, limit int) ([]*model.DirectMessage, error) {
	ok, err := s.dmRepo.IsParticipant(dmChannelID, userID)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrDMNotParticipant
	}

	return s.dmRepo.ListMessages(dmChannelID, before, limit)
}
