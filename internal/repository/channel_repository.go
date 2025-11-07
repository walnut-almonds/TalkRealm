package repository

import (
	"errors"

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
	err := r.db.Preload("Guild").First(&channel, id).Error
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
