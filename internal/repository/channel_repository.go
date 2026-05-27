package repository

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// ChannelRepository 頻道資料庫操作介面
type ChannelRepository interface {
	Create(channel *model.Channel) error
	GetByID(id uint) (*model.Channel, error)
	Update(channel *model.Channel) error
	Delete(id uint) error
	GetByGuildID(guildID uint) ([]*model.Channel, error)
	GetByType(guildID uint, channelType string) ([]*model.Channel, error)

	// DM channel operations
	GetOrCreateDMChannel(user1ID, user2ID uint) (*model.Channel, error)
	ListDMChannels(userID uint) ([]*model.Channel, error)
	IsDMParticipant(channelID, userID uint) (bool, error)
	GetDMParticipants(channelID uint) ([]*model.ChannelParticipant, error)
}

type channelRepository struct {
	db *gorm.DB
}

// NewChannelRepository 建立頻道 repository
func NewChannelRepository(db *gorm.DB) ChannelRepository {
	return &channelRepository{db: db}
}

// Create 建立新頻道
func (r *channelRepository) Create(channel *model.Channel) error {
	return r.db.Create(channel).Error
}

// GetByID 透過 ID 取得頻道
func (r *channelRepository) GetByID(id uint) (*model.Channel, error) {
	var channel model.Channel

	err := r.db.First(&channel, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("channel not found")
		}

		return nil, err
	}

	return &channel, nil
}

// Update 更新頻道資訊
func (r *channelRepository) Update(channel *model.Channel) error {
	return r.db.Save(channel).Error
}

// Delete 刪除頻道
func (r *channelRepository) Delete(id uint) error {
	return r.db.Delete(&model.Channel{}, id).Error
}

// GetByGuildID 取得社群的所有頻道
func (r *channelRepository) GetByGuildID(guildID uint) ([]*model.Channel, error) {
	var channels []*model.Channel

	err := r.db.Where("guild_id = ?", guildID).Order("position ASC").Find(&channels).Error

	return channels, err
}

// GetByType 取得特定類型的頻道
func (r *channelRepository) GetByType(guildID uint, channelType string) ([]*model.Channel, error) {
	var channels []*model.Channel

	err := r.db.
		Where("guild_id = ? AND type = ?", guildID, channelType).
		Order("position ASC").
		Find(&channels).Error

	return channels, err
}

// GetOrCreateDMChannel 取得或建立兩位使用者之間的 DM 頻道
func (r *channelRepository) GetOrCreateDMChannel(user1ID, user2ID uint) (*model.Channel, error) {
	// 查詢同時包含兩位使用者的 DM 頻道
	var channel model.Channel

	err := r.db.
		Joins("JOIN channel_participants cp1 ON cp1.channel_id = channels.id AND cp1.user_id = ?", user1ID).
		Joins("JOIN channel_participants cp2 ON cp2.channel_id = channels.id AND cp2.user_id = ?", user2ID).
		Where("channels.type = 'dm'").
		Preload("Participants.User").
		First(&channel).Error
	if err == nil {
		return &channel, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 不存在時建立新 DM 頻道
	newChannel := &model.Channel{
		Type:      "dm",
		Name:      "",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := r.db.Create(newChannel).Error; err != nil {
		return nil, err
	}

	// 建立兩個 ChannelParticipant 記錄
	participants := []model.ChannelParticipant{
		{ChannelID: newChannel.ID, UserID: user1ID},
		{ChannelID: newChannel.ID, UserID: user2ID},
	}

	if err := r.db.Create(&participants).Error; err != nil {
		// 回滾頻道建立
		_ = r.db.Delete(newChannel).Error
		return nil, err
	}

	// 重新載入（含 Participants.User）
	if err := r.db.Preload("Participants.User").First(newChannel, newChannel.ID).Error; err != nil {
		return nil, err
	}

	return newChannel, nil
}

// ListDMChannels 列出使用者參與的所有 DM 頻道
func (r *channelRepository) ListDMChannels(userID uint) ([]*model.Channel, error) {
	var channels []*model.Channel

	err := r.db.
		Joins(
			"JOIN channel_participants cp ON cp.channel_id = channels.id AND cp.user_id = ?",
			userID,
		).
		Where("channels.type = 'dm'").
		Preload("Participants.User").
		Order("channels.updated_at DESC").
		Find(&channels).Error

	return channels, err
}

// IsDMParticipant 檢查使用者是否為指定 DM 頻道的參與者
func (r *channelRepository) IsDMParticipant(channelID, userID uint) (bool, error) {
	var count int64

	err := r.db.Model(&model.ChannelParticipant{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Count(&count).Error

	return count > 0, err
}

// GetDMParticipants 取得 DM 頻道的所有參與者
func (r *channelRepository) GetDMParticipants(channelID uint) ([]*model.ChannelParticipant, error) {
	var participants []*model.ChannelParticipant

	err := r.db.
		Where("channel_id = ?", channelID).
		Find(&participants).Error

	return participants, err
}
