package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FollowRepository 單向追蹤關係的資料庫操作介面
type FollowRepository interface {
	Follow(followerID, followeeID uint) error
	Unfollow(followerID, followeeID uint) error
	IsFollowing(followerID, followeeID uint) (bool, error)
	FolloweeIDs(followerID uint) ([]uint, error)
	FollowerIDs(followeeID uint) ([]uint, error)
	SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error)
	ListFollowing(userID uint) ([]*model.Follow, error)
	ListFollowers(userID uint) ([]*model.Follow, error)
	CountFollowing(userID uint) (int64, error)
	CountFollowers(userID uint) (int64, error)
}

type followRepository struct{ db *gorm.DB }

// NewFollowRepository 建立追蹤 repository
func NewFollowRepository(db *gorm.DB) FollowRepository { return &followRepository{db: db} }

// Follow 冪等建立追蹤關係（重複追蹤不報錯）
func (r *followRepository) Follow(followerID, followeeID uint) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.Follow{FollowerID: followerID, FolloweeID: followeeID}).Error
}

func (r *followRepository) Unfollow(followerID, followeeID uint) error {
	return r.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&model.Follow{}).Error
}

func (r *followRepository) IsFollowing(followerID, followeeID uint) (bool, error) {
	var n int64

	err := r.db.Model(&model.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Count(&n).Error

	return n > 0, err
}

func (r *followRepository) FolloweeIDs(followerID uint) ([]uint, error) {
	var ids []uint

	err := r.db.Model(&model.Follow{}).
		Where("follower_id = ?", followerID).Pluck("followee_id", &ids).Error

	return ids, err
}

func (r *followRepository) FollowerIDs(followeeID uint) ([]uint, error) {
	var ids []uint

	err := r.db.Model(&model.Follow{}).
		Where("followee_id = ?", followeeID).Pluck("follower_id", &ids).Error

	return ids, err
}

// SecondDegreeAuthorIDs 回傳「被追蹤者所追蹤的人」集合，排除 viewer 直接追蹤者與 viewer 本身。
func (r *followRepository) SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error) {
	var ids []uint

	err := r.db.Raw(`
		SELECT DISTINCT f2.followee_id
		FROM follows f1
		JOIN follows f2 ON f2.follower_id = f1.followee_id
		WHERE f1.follower_id = ?
		  AND f2.followee_id <> ?
		  AND f2.followee_id NOT IN (SELECT followee_id FROM follows WHERE follower_id = ?)
	`, viewerID, viewerID, viewerID).Scan(&ids).Error
	if err != nil {
		return nil, err
	}

	out := make(map[uint]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}

	return out, nil
}

func (r *followRepository) ListFollowing(userID uint) ([]*model.Follow, error) {
	var out []*model.Follow

	err := r.db.Preload("Followee").Where("follower_id = ?", userID).
		Order("id DESC").Find(&out).Error

	return out, err
}

func (r *followRepository) ListFollowers(userID uint) ([]*model.Follow, error) {
	var out []*model.Follow

	err := r.db.Preload("Follower").Where("followee_id = ?", userID).
		Order("id DESC").Find(&out).Error

	return out, err
}

func (r *followRepository) CountFollowing(userID uint) (int64, error) {
	var n int64

	err := r.db.Model(&model.Follow{}).Where("follower_id = ?", userID).Count(&n).Error

	return n, err
}

func (r *followRepository) CountFollowers(userID uint) (int64, error) {
	var n int64

	err := r.db.Model(&model.Follow{}).Where("followee_id = ?", userID).Count(&n).Error

	return n, err
}
