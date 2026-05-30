package service

import (
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

type unreadService struct {
	readStateRepo repository.ChannelReadStateRepository
}

// NewUnreadService 建立 UnreadService
func NewUnreadService(readStateRepo repository.ChannelReadStateRepository) UnreadService {
	return &unreadService{readStateRepo: readStateRepo}
}

// AckChannel 更新已讀位置
func (s *unreadService) AckChannel(userID, channelID, lastMessageID uint) error {
	return s.readStateRepo.Upsert(userID, channelID, lastMessageID)
}

// GetChannelUnread 取得單一頻道未讀計數
func (s *unreadService) GetChannelUnread(userID, channelID uint) (*ChannelUnreadCount, error) {
	return s.readStateRepo.GetChannelUnread(userID, channelID)
}

// GetAllUnread 批次取得全部頻道未讀計數
func (s *unreadService) GetAllUnread(userID uint) ([]*ChannelUnreadCount, error) {
	return s.readStateRepo.GetAllUnread(userID)
}
