package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// FeedCommentRepository 貼文留言資料庫操作介面
type FeedCommentRepository interface {
	Create(c *model.FeedComment) error
	GetByID(id uint) (*model.FeedComment, error)
	Update(c *model.FeedComment) error
	Delete(id uint) error
	ByPostCursor(postID, before uint, limit int) ([]*model.FeedComment, error)
	CountByPostIDs(ids []uint) (map[uint]int64, error)
}

type feedCommentRepository struct{ db *gorm.DB }

// NewFeedCommentRepository 建立貼文留言 repository
func NewFeedCommentRepository(db *gorm.DB) FeedCommentRepository {
	return &feedCommentRepository{db: db}
}

func (r *feedCommentRepository) Create(c *model.FeedComment) error { return r.db.Create(c).Error }

func (r *feedCommentRepository) GetByID(id uint) (*model.FeedComment, error) {
	var c model.FeedComment
	if err := r.db.Preload("Author").First(&c, id).Error; err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *feedCommentRepository) Update(c *model.FeedComment) error { return r.db.Save(c).Error }

func (r *feedCommentRepository) Delete(id uint) error {
	return r.db.Delete(&model.FeedComment{}, id).Error
}

// ByPostCursor 以 id DESC 抓取後反轉為時間順序（與 GetByChannelIDCursor 一致）
func (r *feedCommentRepository) ByPostCursor(
	postID, before uint,
	limit int,
) ([]*model.FeedComment, error) {
	var cs []*model.FeedComment
	q := r.db.Preload("Author").Where("post_id = ?", postID).Order("id DESC").Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Find(&cs).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(cs)-1; i < j; i, j = i+1, j-1 {
		cs[i], cs[j] = cs[j], cs[i]
	}

	return cs, nil
}

func (r *feedCommentRepository) CountByPostIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		PostID uint
		Cnt    int64
	}
	var rows []row
	err := r.db.Model(&model.FeedComment{}).
		Select("post_id, COUNT(*) AS cnt").
		Where("post_id IN ?", ids).Group("post_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.PostID] = x.Cnt
	}

	return out, nil
}
