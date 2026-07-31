package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FeedCommentLikeRepository 留言按讚資料庫操作介面
type FeedCommentLikeRepository interface {
	Create(like *model.FeedCommentLike) error
	Delete(commentID, userID uint) error
	CountByCommentID(commentID uint) (int64, error)
	CountByCommentIDs(ids []uint) (map[uint]int64, error)
	LikedCommentIDs(userID uint, ids []uint) (map[uint]bool, error)
}

type feedCommentLikeRepository struct {
	db *gorm.DB
}

// NewFeedCommentLikeRepository 建立留言按讚 repository
func NewFeedCommentLikeRepository(db *gorm.DB) FeedCommentLikeRepository {
	return &feedCommentLikeRepository{db: db}
}

// Create 冪等建立按讚（重複讚不報錯）
func (r *feedCommentLikeRepository) Create(like *model.FeedCommentLike) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *feedCommentLikeRepository) Delete(commentID, userID uint) error {
	return r.db.
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&model.FeedCommentLike{}).Error
}

func (r *feedCommentLikeRepository) CountByCommentID(commentID uint) (int64, error) {
	var n int64

	err := r.db.Model(&model.FeedCommentLike{}).
		Where("comment_id = ?", commentID).Count(&n).Error

	return n, err
}

type feedCommentLikeCountRow struct {
	CommentID uint
	Cnt       int64
}

func (r *feedCommentLikeRepository) CountByCommentIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var rows []feedCommentLikeCountRow

	err := r.db.Model(&model.FeedCommentLike{}).
		Select("comment_id, COUNT(*) AS cnt").
		Where("comment_id IN ?", ids).
		Group("comment_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.CommentID] = row.Cnt
	}

	return out, nil
}

func (r *feedCommentLikeRepository) LikedCommentIDs(
	userID uint,
	ids []uint,
) (map[uint]bool, error) {
	out := make(map[uint]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var liked []uint

	err := r.db.Model(&model.FeedCommentLike{}).
		Where("user_id = ? AND comment_id IN ?", userID, ids).
		Pluck("comment_id", &liked).Error
	if err != nil {
		return nil, err
	}

	for _, id := range liked {
		out[id] = true
	}

	return out, nil
}
