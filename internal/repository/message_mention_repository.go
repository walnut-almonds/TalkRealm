package repository

import (
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// MessageMentionRepository 訊息提及資料庫操作介面
type MessageMentionRepository interface {
	// BulkCreate 批次建立多筆 mention 記錄
	BulkCreate(mentions []*model.MessageMention) error
	// GetByMessageID 取得某訊息中所有被提及的用戶（含 mention_type）
	GetByMessageID(messageID uint) ([]*model.MessageMention, error)
	// GetMentionedUserIDs 取得訊息中所有被提及用戶的 ID
	GetMentionedUserIDs(messageID uint) ([]uint, error)
}

type messageMentionRepository struct {
	db *gorm.DB
}

// NewMessageMentionRepository 建立 MessageMention repository
func NewMessageMentionRepository(db *gorm.DB) MessageMentionRepository {
	return &messageMentionRepository{db: db}
}

// BulkCreate 批次建立 mention 記錄（忽略重複）
func (r *messageMentionRepository) BulkCreate(mentions []*model.MessageMention) error {
	if len(mentions) == 0 {
		return nil
	}

	now := time.Now()

	for _, m := range mentions {
		m.CreatedAt = now
	}

	return r.db.CreateInBatches(mentions, 100).Error
}

// GetByMessageID 取得某訊息的所有 mention 記錄
func (r *messageMentionRepository) GetByMessageID(messageID uint) ([]*model.MessageMention, error) {
	var mentions []*model.MessageMention

	if err := r.db.Preload("User").
		Where("message_id = ?", messageID).
		Find(&mentions).Error; err != nil {
		return nil, err
	}

	return mentions, nil
}

// GetMentionedUserIDs 取得訊息中所有被提及的 user IDs（去重）
func (r *messageMentionRepository) GetMentionedUserIDs(messageID uint) ([]uint, error) {
	var ids []uint

	if err := r.db.Model(&model.MessageMention{}).
		Select("DISTINCT user_id").
		Where("message_id = ?", messageID).
		Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}

	return ids, nil
}
