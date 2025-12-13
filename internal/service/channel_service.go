package service

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrChannelNotFound    = errors.New("channel not found")
	ErrNotGuildMemberCh   = errors.New("not a member of this guild")
	ErrInvalidChannelType = errors.New("invalid channel type")
)

// CreateChannelRequest 建立頻道請求
type CreateChannelRequest struct {
	GuildID  uint   `json:"guild_id" binding:"required"`
	Name     string `json:"name"     binding:"required,min=1,max=100"`
	Type     string `json:"type"     binding:"required,oneof=text voice"`
	Topic    string `json:"topic"    binding:"max=1024"`
	Position int    `json:"position"`
}

// UpdateChannelRequest 更新頻道請求
type UpdateChannelRequest struct {
	Name     string `json:"name"     binding:"omitempty,min=1,max=100"`
	Type     string `json:"type"     binding:"omitempty,oneof=text voice"`
	Topic    string `json:"topic"    binding:"max=1024"`
	Position *int   `json:"position"`
}

// ChannelService 頻道服務介面
type ChannelService interface {
	CreateChannel(userID uint, req *CreateChannelRequest) (*model.Channel, error)
	GetChannel(channelID, userID uint) (*model.Channel, error)
	ListGuildChannels(guildID, userID uint) ([]*model.Channel, error)
	UpdateChannel(channelID, userID uint, req *UpdateChannelRequest) (*model.Channel, error)
	DeleteChannel(channelID, userID uint) error
	UpdateChannelPosition(channelID, userID uint, position int) error
}

type channelService struct {
	channelRepo     repository.ChannelRepository
	guildRepo       repository.GuildRepository
	guildMemberRepo repository.GuildMemberRepository
}

// NewChannelService 建立頻道服務
func NewChannelService(
	channelRepo repository.ChannelRepository,
	guildRepo repository.GuildRepository,
	guildMemberRepo repository.GuildMemberRepository,
) ChannelService {
	return &channelService{
		channelRepo:     channelRepo,
		guildRepo:       guildRepo,
		guildMemberRepo: guildMemberRepo,
	}
}

// CreateChannel 建立頻道
func (s *channelService) CreateChannel(
	userID uint,
	req *CreateChannelRequest,
) (*model.Channel, error) {
	// 檢查社群是否存在
	guild, err := s.guildRepo.GetByID(req.GuildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	// 檢查使用者是否為社群擁有者或管理員
	if guild.OwnerID != userID {
		// 檢查是否為管理員
		member, err := s.guildMemberRepo.GetMember(req.GuildID, userID)
		if err != nil || member == nil {
			return nil, ErrNotGuildMemberCh
		}

		//nolint:goconst // 足夠清晰不需要 const
		if member.Role != "admin" && member.Role != "owner" {
			return nil, errors.New("only owner or admin can create channels")
		}
	}

	// 驗證頻道類型
	//nolint:goconst // 足夠清晰不需要 const
	if req.Type != "text" && req.Type != "voice" {
		return nil, ErrInvalidChannelType
	}

	// 如果沒有指定位置，自動設定為最後
	position := req.Position
	if position == 0 {
		channels, err := s.channelRepo.GetByGuildID(req.GuildID)
		if err == nil {
			position = len(channels)
		}
	}

	channel := &model.Channel{
		GuildID:   req.GuildID,
		Name:      req.Name,
		Type:      req.Type,
		Topic:     req.Topic,
		Position:  position,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.channelRepo.Create(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetChannel 取得頻道詳情
func (s *channelService) GetChannel(channelID, userID uint) (*model.Channel, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	// 檢查使用者是否為該社群成員
	member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
	if err != nil || member == nil {
		return nil, ErrNotGuildMemberCh
	}

	return channel, nil
}

// ListGuildChannels 列出社群的所有頻道
func (s *channelService) ListGuildChannels(guildID, userID uint) ([]*model.Channel, error) {
	// 檢查社群是否存在
	_, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	// 檢查使用者是否為社群成員
	member, err := s.guildMemberRepo.GetMember(guildID, userID)
	if err != nil || member == nil {
		return nil, ErrNotGuildMemberCh
	}

	return s.channelRepo.GetByGuildID(guildID)
}

// UpdateChannel 更新頻道資訊
func (s *channelService) UpdateChannel(
	channelID, userID uint,
	req *UpdateChannelRequest,
) (*model.Channel, error) {
	// 取得頻道
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	// 檢查權限（只有擁有者或管理員可以更新）
	guild, err := s.guildRepo.GetByID(channel.GuildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	if guild.OwnerID != userID {
		member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
		if err != nil || member == nil {
			return nil, ErrNotGuildMemberCh
		}

		if member.Role != "admin" && member.Role != "owner" {
			return nil, errors.New("only owner or admin can update channels")
		}
	}

	// 更新欄位
	if req.Name != "" {
		channel.Name = req.Name
	}

	if req.Type != "" {
		if req.Type != "text" && req.Type != "voice" {
			return nil, ErrInvalidChannelType
		}

		channel.Type = req.Type
	}

	if req.Topic != "" {
		channel.Topic = req.Topic
	}

	if req.Position != nil {
		channel.Position = *req.Position
	}

	channel.UpdatedAt = time.Now()

	if err := s.channelRepo.Update(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// DeleteChannel 刪除頻道
func (s *channelService) DeleteChannel(channelID, userID uint) error {
	// 取得頻道
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return ErrChannelNotFound
	}

	// 檢查權限（只有擁有者或管理員可以刪除）
	guild, err := s.guildRepo.GetByID(channel.GuildID)
	if err != nil {
		return ErrGuildNotFound
	}

	if guild.OwnerID != userID {
		member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
		if err != nil || member == nil {
			return ErrNotGuildMemberCh
		}

		if member.Role != "admin" && member.Role != "owner" {
			return errors.New("only owner or admin can delete channels")
		}
	}

	return s.channelRepo.Delete(channelID)
}

// UpdateChannelPosition 更新頻道位置
func (s *channelService) UpdateChannelPosition(channelID, userID uint, position int) error {
	// 取得頻道
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return ErrChannelNotFound
	}

	// 檢查權限
	guild, err := s.guildRepo.GetByID(channel.GuildID)
	if err != nil {
		return ErrGuildNotFound
	}

	if guild.OwnerID != userID {
		member, err := s.guildMemberRepo.GetMember(channel.GuildID, userID)
		if err != nil || member == nil {
			return ErrNotGuildMemberCh
		}

		if member.Role != "admin" && member.Role != "owner" {
			return errors.New("only owner or admin can update channel position")
		}
	}

	channel.Position = position
	channel.UpdatedAt = time.Now()

	return s.channelRepo.Update(channel)
}
