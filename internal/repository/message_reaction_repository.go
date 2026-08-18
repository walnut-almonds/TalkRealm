package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// MessageReactionRepository 訊息表情回應的資料庫操作介面
type MessageReactionRepository interface {
	// Toggle 有則刪、無則增，回傳 true 代表這次是新增。
	Toggle(messageID, userID uint, emoji string) (bool, error)
	// DeleteByMessageIDs 刪訊息時一併清掉其表情回應。
	DeleteByMessageIDs(ids []uint) error
}

type messageReactionRepository struct {
	db *gorm.DB
}

// NewMessageReactionRepository 建立表情回應 repository
func NewMessageReactionRepository(db *gorm.DB) MessageReactionRepository {
	return &messageReactionRepository{db: db}
}

// Toggle 在單一交易內完成「查→ 有就刪／沒有就加」，避免同一使用者連點兩下時
// 兩個請求都讀到「不存在」而雙雙嘗試新增（其中一個會撞上唯一索引）。
func (r *messageReactionRepository) Toggle(
	messageID, userID uint,
	emoji string,
) (bool, error) {
	added := false

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing model.MessageReaction

		err := tx.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
			First(&existing).Error

		switch {
		case err == nil:
			added = false

			return tx.Delete(&existing).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			added = true

			return tx.Create(&model.MessageReaction{
				MessageID: messageID,
				UserID:    userID,
				Emoji:     emoji,
			}).Error
		default:
			return err
		}
	})

	return added, err
}

func (r *messageReactionRepository) DeleteByMessageIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.Where("message_id IN ?", ids).Delete(&model.MessageReaction{}).Error
}
