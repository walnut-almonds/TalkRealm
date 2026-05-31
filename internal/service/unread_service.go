package service

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/repository"
)

// ChannelUnreadCount 單一頻道的未讀統計（供 handler 使用）
type ChannelUnreadCount = repository.ChannelUnreadCount

// UnreadService 未讀狀態服務介面
type UnreadService interface {
	// AckChannel 標記使用者在某頻道的已讀位置
	AckChannel(userID, channelID, lastMessageID uint) error
	// GetChannelUnread 取得使用者在某頻道的未讀計數
	GetChannelUnread(userID, channelID uint) (*ChannelUnreadCount, error)
	// GetAllUnread 批次取得使用者所有頻道的未讀計數
	GetAllUnread(userID uint) ([]*ChannelUnreadCount, error)
}

var (
	ErrUnreadAccessDenied     = errors.New("no access to channel")
	ErrUnreadInvalidAckTarget = errors.New("ack message does not belong to channel")
)

type unreadService struct {
	readStateRepo   repository.ChannelReadStateRepository
	channelRepo     repository.ChannelRepository
	guildMemberRepo repository.GuildMemberRepository
	messageRepo     repository.MessageRepository
}

// NewUnreadService 建立 UnreadService
func NewUnreadService(
	readStateRepo repository.ChannelReadStateRepository,
	channelRepo repository.ChannelRepository,
	guildMemberRepo repository.GuildMemberRepository,
	messageRepo repository.MessageRepository,
) UnreadService {
	return &unreadService{
		readStateRepo:   readStateRepo,
		channelRepo:     channelRepo,
		guildMemberRepo: guildMemberRepo,
		messageRepo:     messageRepo,
	}
}

// AckChannel 更新已讀位置
func (s *unreadService) AckChannel(userID, channelID, lastMessageID uint) error {
	if err := s.ensureChannelAccess(userID, channelID); err != nil {
		return err
	}

	msg, err := s.messageRepo.GetByID(lastMessageID)
	if err != nil || msg == nil || msg.ChannelID != channelID {
		return ErrUnreadInvalidAckTarget
	}

	return s.readStateRepo.Upsert(userID, channelID, lastMessageID)
}

// GetChannelUnread 取得單一頻道未讀計數
func (s *unreadService) GetChannelUnread(userID, channelID uint) (*ChannelUnreadCount, error) {
	if err := s.ensureChannelAccess(userID, channelID); err != nil {
		return nil, err
	}

	return s.readStateRepo.GetChannelUnread(userID, channelID)
}

// GetAllUnread 批次取得全部頻道未讀計數
func (s *unreadService) GetAllUnread(userID uint) ([]*ChannelUnreadCount, error) {
	return s.readStateRepo.GetAllUnread(userID)
}

func (s *unreadService) ensureChannelAccess(userID, channelID uint) error {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil || channel == nil {
		return ErrUnreadAccessDenied
	}

	if channel.Type == "dm" {
		ok, err := s.channelRepo.IsDMParticipant(channelID, userID)
		if err != nil || !ok {
			return ErrUnreadAccessDenied
		}

		return nil
	}

	if channel.GuildID == nil {
		return ErrUnreadAccessDenied
	}

	member, err := s.guildMemberRepo.GetMember(*channel.GuildID, userID)
	if err != nil || member == nil {
		return ErrUnreadAccessDenied
	}

	return nil
}
