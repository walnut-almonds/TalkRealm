package repository

import (
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

type FeedPostRepository interface {
	Create(p *model.FeedPost) error
	GetByID(id uint) (*model.FeedPost, error)
	Update(p *model.FeedPost) error
	AttachFiles(postID uint, fileIDs []uint) error
	TimelineCursor(authorIDs []uint, before uint, limit int) ([]*model.FeedPost, error)
	ByAuthorCursor(authorID, before uint, limit int) ([]*model.FeedPost, error)
	DeleteCascade(postID uint) error
	RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error)
	AuthorAffinity(viewerID uint) (map[uint]int64, error)
}

type feedPostRepository struct{ db *gorm.DB }

func NewFeedPostRepository(db *gorm.DB) FeedPostRepository { return &feedPostRepository{db: db} }

func (r *feedPostRepository) Create(p *model.FeedPost) error { return r.db.Create(p).Error }

func (r *feedPostRepository) GetByID(id uint) (*model.FeedPost, error) {
	var p model.FeedPost
	err := r.db.Preload("Author").Preload("Attachments.File").First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *feedPostRepository) Update(p *model.FeedPost) error { return r.db.Save(p).Error }

func (r *feedPostRepository) AttachFiles(postID uint, fileIDs []uint) error {
	if len(fileIDs) == 0 {
		return nil
	}
	rows := make([]model.FeedPostAttachment, len(fileIDs))
	for i, fid := range fileIDs {
		rows[i] = model.FeedPostAttachment{PostID: postID, FileID: fid}
	}
	return r.db.Create(&rows).Error
}

func (r *feedPostRepository) TimelineCursor(authorIDs []uint, before uint, limit int) ([]*model.FeedPost, error) {
	var posts []*model.FeedPost
	if len(authorIDs) == 0 {
		return posts, nil
	}
	q := r.db.Preload("Author").Preload("Attachments.File").
		Where("author_id IN ?", authorIDs).
		Order("id DESC").Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	return posts, q.Find(&posts).Error // timeline stays newest-first, no reverse
}

func (r *feedPostRepository) ByAuthorCursor(authorID, before uint, limit int) ([]*model.FeedPost, error) {
	var posts []*model.FeedPost
	q := r.db.Preload("Author").Preload("Attachments.File").
		Where("author_id = ?", authorID).
		Order("id DESC").Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	return posts, q.Find(&posts).Error
}

func (r *feedPostRepository) RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error) {
	var posts []*model.FeedPost
	err := r.db.Preload("Author").Preload("Attachments.File").
		Where("author_id <> ? AND created_at >= ?", excludeAuthorID, since).
		Order("id DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

func (r *feedPostRepository) AuthorAffinity(viewerID uint) (map[uint]int64, error) {
	out := make(map[uint]int64)
	type row struct {
		AuthorID uint
		Cnt      int64
	}
	// viewer's likes, grouped by the liked post's author
	var likeRows []row
	if err := r.db.Table("feed_post_likes AS fpl").
		Select("fp.author_id AS author_id, COUNT(*) AS cnt").
		Joins("JOIN feed_posts AS fp ON fp.id = fpl.post_id").
		Where("fpl.user_id = ?", viewerID).
		Group("fp.author_id").Scan(&likeRows).Error; err != nil {
		return nil, err
	}
	for _, x := range likeRows {
		out[x.AuthorID] += x.Cnt
	}
	// viewer's comments, grouped by the commented post's author
	var commentRows []row
	if err := r.db.Table("feed_comments AS fc").
		Select("fp.author_id AS author_id, COUNT(*) AS cnt").
		Joins("JOIN feed_posts AS fp ON fp.id = fc.post_id").
		Where("fc.author_id = ?", viewerID).
		Group("fp.author_id").Scan(&commentRows).Error; err != nil {
		return nil, err
	}
	for _, x := range commentRows {
		out[x.AuthorID] += x.Cnt
	}
	return out, nil
}

func (r *feedPostRepository) DeleteCascade(postID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", postID).Delete(&model.FeedComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.FeedPostLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.FeedPostAttachment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.FeedPost{}, postID).Error
	})
}
