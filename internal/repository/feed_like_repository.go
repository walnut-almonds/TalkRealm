package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FeedPostLikeRepository 貼文按讚資料庫操作介面
type FeedPostLikeRepository interface {
	Create(like *model.FeedPostLike) error
	Delete(postID, userID uint) error
	CountByPostID(postID uint) (int64, error)
	CountByPostIDs(ids []uint) (map[uint]int64, error)
	LikedPostIDs(userID uint, ids []uint) (map[uint]bool, error)
}

type feedPostLikeRepository struct {
	db *gorm.DB
}

// NewFeedPostLikeRepository 建立貼文按讚 repository
func NewFeedPostLikeRepository(db *gorm.DB) FeedPostLikeRepository {
	return &feedPostLikeRepository{db: db}
}

// Create 冪等建立按讚（重複讚不報錯）
func (r *feedPostLikeRepository) Create(like *model.FeedPostLike) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *feedPostLikeRepository) Delete(postID, userID uint) error {
	return r.db.
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.FeedPostLike{}).Error
}

func (r *feedPostLikeRepository) CountByPostID(postID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.FeedPostLike{}).
		Where("post_id = ?", postID).Count(&n).Error

	return n, err
}

type feedLikeCountRow struct {
	PostID uint
	Cnt    int64
}

func (r *feedPostLikeRepository) CountByPostIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []feedLikeCountRow
	err := r.db.Model(&model.FeedPostLike{}).
		Select("post_id, COUNT(*) AS cnt").
		Where("post_id IN ?", ids).
		Group("post_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = row.Cnt
	}

	return out, nil
}

func (r *feedPostLikeRepository) LikedPostIDs(userID uint, ids []uint) (map[uint]bool, error) {
	out := make(map[uint]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var liked []uint
	err := r.db.Model(&model.FeedPostLike{}).
		Where("user_id = ? AND post_id IN ?", userID, ids).
		Pluck("post_id", &liked).Error
	if err != nil {
		return nil, err
	}
	for _, id := range liked {
		out[id] = true
	}

	return out, nil
}
