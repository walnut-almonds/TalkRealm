package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MessageLikeRepository 按讚資料庫操作介面
type MessageLikeRepository interface {
	Create(like *model.MessageLike) error
	Delete(messageID, userID uint) error
	CountByMessageID(messageID uint) (int64, error)
	CountByMessageIDs(ids []uint) (map[uint]int64, error)
	LikedMessageIDs(userID uint, ids []uint) (map[uint]bool, error)
	DeleteByMessageIDs(ids []uint) error
}

type messageLikeRepository struct {
	db *gorm.DB
}

// NewMessageLikeRepository 建立按讚 repository
func NewMessageLikeRepository(db *gorm.DB) MessageLikeRepository {
	return &messageLikeRepository{db: db}
}

// Create 冪等建立按讚（重複讚不報錯）
func (r *messageLikeRepository) Create(like *model.MessageLike) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *messageLikeRepository) Delete(messageID, userID uint) error {
	return r.db.
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Delete(&model.MessageLike{}).Error
}

func (r *messageLikeRepository) CountByMessageID(messageID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.MessageLike{}).
		Where("message_id = ?", messageID).Count(&n).Error

	return n, err
}

type likeCountRow struct {
	MessageID uint
	Cnt       int64
}

func (r *messageLikeRepository) CountByMessageIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []likeCountRow
	err := r.db.Model(&model.MessageLike{}).
		Select("message_id, COUNT(*) AS cnt").
		Where("message_id IN ?", ids).
		Group("message_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.MessageID] = row.Cnt
	}

	return out, nil
}

func (r *messageLikeRepository) LikedMessageIDs(userID uint, ids []uint) (map[uint]bool, error) {
	out := make(map[uint]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var liked []uint
	err := r.db.Model(&model.MessageLike{}).
		Where("user_id = ? AND message_id IN ?", userID, ids).
		Pluck("message_id", &liked).Error
	if err != nil {
		return nil, err
	}
	for _, id := range liked {
		out[id] = true
	}

	return out, nil
}

func (r *messageLikeRepository) DeleteByMessageIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.Where("message_id IN ?", ids).Delete(&model.MessageLike{}).Error
}
