package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// FriendshipRepository 好友關係資料庫操作介面
type FriendshipRepository interface {
	// Create 建立一筆好友申請（status=pending）
	Create(requesterID, addresseeID uint) (*model.Friendship, error)
	// GetBetween 取得兩位使用者之間的好友記錄（任意方向）
	GetBetween(userA, userB uint) (*model.Friendship, error)
	// UpdateStatus 更新好友關係狀態
	UpdateStatus(requesterID, addresseeID uint, status string) error
	// Delete 刪除好友關係（任意方向）
	Delete(userA, userB uint) error
	// ListFriends 列出使用者所有已接受的好友（帶入對方 User 資料）
	ListFriends(userID uint) ([]*model.Friendship, error)
	// ListIncomingRequests 列出使用者收到的待處理申請
	ListIncomingRequests(userID uint) ([]*model.Friendship, error)
	// ListOutgoingRequests 列出使用者送出的待處理申請
	ListOutgoingRequests(userID uint) ([]*model.Friendship, error)
	// FriendIDs 列出所有已接受好友的 user id（輕量，learn 好友榜等場景用）
	FriendIDs(userID uint) ([]uint, error)
}

type friendshipRepository struct {
	db *gorm.DB
}

// NewFriendshipRepository 建立好友關係 repository
func NewFriendshipRepository(db *gorm.DB) FriendshipRepository {
	return &friendshipRepository{db: db}
}

func (r *friendshipRepository) Create(requesterID, addresseeID uint) (*model.Friendship, error) {
	f := &model.Friendship{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      "pending",
	}
	if err := r.db.Create(f).Error; err != nil {
		return nil, err
	}

	return r.getWithUsers(f.ID)
}

func (r *friendshipRepository) GetBetween(userA, userB uint) (*model.Friendship, error) {
	var f model.Friendship

	err := r.db.
		Preload("Requester").
		Preload("Addressee").
		Where(
			"(requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)",
			userA, userB, userB, userA,
		).
		First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil //nolint:nilnil
		}

		return nil, err
	}

	return &f, nil
}

func (r *friendshipRepository) UpdateStatus(requesterID, addresseeID uint, status string) error {
	return r.db.
		Model(&model.Friendship{}).
		Where("requester_id = ? AND addressee_id = ?", requesterID, addresseeID).
		Update("status", status).Error
}

func (r *friendshipRepository) Delete(userA, userB uint) error {
	return r.db.
		Where(
			"(requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)",
			userA, userB, userB, userA,
		).
		Delete(&model.Friendship{}).Error
}

func (r *friendshipRepository) ListFriends(userID uint) ([]*model.Friendship, error) {
	var friendships []*model.Friendship

	err := r.db.
		Preload("Requester").
		Preload("Addressee").
		Where(
			"(requester_id = ? OR addressee_id = ?) AND status = 'accepted'",
			userID, userID,
		).
		Find(&friendships).Error

	return friendships, err
}

func (r *friendshipRepository) FriendIDs(userID uint) ([]uint, error) {
	var ids []uint

	err := r.db.Model(&model.Friendship{}).
		Where("(requester_id = ? OR addressee_id = ?) AND status = 'accepted'", userID, userID).
		Select("CASE WHEN requester_id = ? THEN addressee_id ELSE requester_id END", userID).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *friendshipRepository) ListIncomingRequests(userID uint) ([]*model.Friendship, error) {
	var friendships []*model.Friendship

	err := r.db.
		Preload("Requester").
		Where("addressee_id = ? AND status = 'pending'", userID).
		Find(&friendships).Error

	return friendships, err
}

func (r *friendshipRepository) ListOutgoingRequests(userID uint) ([]*model.Friendship, error) {
	var friendships []*model.Friendship

	err := r.db.
		Preload("Addressee").
		Where("requester_id = ? AND status = 'pending'", userID).
		Find(&friendships).Error

	return friendships, err
}

// getWithUsers 輔助：載入 Preload 後取回完整記錄
func (r *friendshipRepository) getWithUsers(id uint) (*model.Friendship, error) {
	var f model.Friendship

	err := r.db.Preload("Requester").Preload("Addressee").First(&f, id).Error
	if err != nil {
		return nil, err
	}

	return &f, nil
}
