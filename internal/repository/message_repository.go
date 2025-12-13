package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// MessageRepository 訊息資料庫操作介面
type MessageRepository interface {
	Create(message *model.Message) error
	GetByID(id uint) (*model.Message, error)
	Update(message *model.Message) error
	Delete(id uint) error
	GetByChannelID(channelID uint, offset, limit int) ([]*model.Message, error)
	GetByUserID(userID uint, offset, limit int) ([]*model.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 建立訊息 repository
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

// Create 建立新訊息
func (r *messageRepository) Create(message *model.Message) error {
	return r.db.Create(message).Error
}

// GetByID 透過 ID 取得訊息
func (r *messageRepository) GetByID(id uint) (*model.Message, error) {
	var message model.Message

	err := r.db.Preload("User").Preload("Channel").First(&message, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("message not found")
		}

		return nil, err
	}

	return &message, nil
}

// Update 更新訊息
func (r *messageRepository) Update(message *model.Message) error {
	return r.db.Save(message).Error
}

// Delete 刪除訊息
func (r *messageRepository) Delete(id uint) error {
	return r.db.Delete(&model.Message{}, id).Error
}

// GetByChannelID 取得頻道的訊息（分頁）
func (r *messageRepository) GetByChannelID(
	channelID uint,
	offset, limit int,
) ([]*model.Message, error) {
	var messages []*model.Message

	err := r.db.
		Preload("User").
		Where("channel_id = ?", channelID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error

	return messages, err
}

// GetByUserID 取得使用者的訊息（分頁）
func (r *messageRepository) GetByUserID(userID uint, offset, limit int) ([]*model.Message, error) {
	var messages []*model.Message

	err := r.db.
		Preload("Channel").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error

	return messages, err
}
