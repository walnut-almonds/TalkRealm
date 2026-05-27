package service

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrDMSelfMessage    = errors.New("cannot open dm with yourself")
	ErrDMNotParticipant = errors.New("not a participant of this dm channel")
)

// DMService 私訊頻道管理服務介面
type DMService interface {
	// GetOrCreateChannel 取得或建立與目標使用者的 DM 頻道
	GetOrCreateChannel(requesterID, targetUserID uint) (*model.Channel, error)
	// ListDMChannels 列出使用者參與的所有 DM 頻道
	ListDMChannels(userID uint) ([]*model.Channel, error)
}

type dmService struct {
	channelRepo repository.ChannelRepository
	userRepo    repository.UserRepository
}

// NewDMService 建立私訊服務
func NewDMService(
	channelRepo repository.ChannelRepository,
	userRepo repository.UserRepository,
) DMService {
	return &dmService{
		channelRepo: channelRepo,
		userRepo:    userRepo,
	}
}

// GetOrCreateChannel 取得或建立與目標使用者的 DM 頻道
func (s *dmService) GetOrCreateChannel(requesterID, targetUserID uint) (*model.Channel, error) {
	if requesterID == targetUserID {
		return nil, ErrDMSelfMessage
	}

	// 確認目標使用者存在
	if _, err := s.userRepo.GetByID(targetUserID); err != nil {
		return nil, errors.New("target user not found")
	}

	return s.channelRepo.GetOrCreateDMChannel(requesterID, targetUserID)
}

// ListDMChannels 列出使用者的所有 DM 頻道
func (s *dmService) ListDMChannels(userID uint) ([]*model.Channel, error) {
	return s.channelRepo.ListDMChannels(userID)
}
