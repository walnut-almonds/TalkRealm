package service

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrFriendRequestExists   = errors.New("friend request already sent")
	ErrAlreadyFriends        = errors.New("already friends")
	ErrFriendRequestNotFound = errors.New("friend request not found")
	ErrCannotFriendSelf      = errors.New("cannot send friend request to yourself")
)

// FriendNotifier 提供向指定使用者推播 WS 事件的能力（由 websocket.Manager 實作）
type FriendNotifier interface {
	BroadcastToUser(userID uint, msgType string, data any)
}

// FriendService 好友系統服務介面
type FriendService interface {
	// SendRequest 透過 username 送出好友申請
	SendRequest(requesterID uint, targetUsername string) (*model.Friendship, error)
	// Accept 接受一筆待處理的好友申請
	Accept(userID, requesterID uint) (*model.Friendship, error)
	// Reject 拒絕一筆待處理的好友申請
	Reject(userID, requesterID uint) error
	// Unfriend 解除好友關係
	Unfriend(userID, friendID uint) error
	// ListFriends 列出已接受的好友
	ListFriends(userID uint) ([]*model.Friendship, error)
	// ListIncomingRequests 列出收到的待處理申請
	ListIncomingRequests(userID uint) ([]*model.Friendship, error)
	// ListOutgoingRequests 列出送出的待處理申請
	ListOutgoingRequests(userID uint) ([]*model.Friendship, error)
}

type friendService struct {
	friendRepo repository.FriendshipRepository
	userRepo   repository.UserRepository
	notifier   FriendNotifier
}

// NewFriendService 建立好友服務
func NewFriendService(
	friendRepo repository.FriendshipRepository,
	userRepo repository.UserRepository,
) *friendService {
	return &friendService{
		friendRepo: friendRepo,
		userRepo:   userRepo,
	}
}

// SetNotifier 注入 WS notifier（避免循環依賴，啟動後再設定）
func (s *friendService) SetNotifier(n FriendNotifier) {
	s.notifier = n
}

func (s *friendService) SendRequest(
	requesterID uint,
	targetUsername string,
) (*model.Friendship, error) {
	target, err := s.userRepo.GetByUsername(targetUsername)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if target.ID == requesterID {
		return nil, ErrCannotFriendSelf
	}

	existing, err := s.friendRepo.GetBetween(requesterID, target.ID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		switch existing.Status {
		case "accepted":
			return nil, ErrAlreadyFriends
		case "pending":
			return nil, ErrFriendRequestExists
		}
	}

	f, err := s.friendRepo.Create(requesterID, target.ID)
	if err != nil {
		return nil, err
	}

	// WS: 通知被申請方
	if s.notifier != nil {
		s.notifier.BroadcastToUser(target.ID, "friend_request", f)
	}

	return f, nil
}

func (s *friendService) Accept(userID, requesterID uint) (*model.Friendship, error) {
	f, err := s.friendRepo.GetBetween(requesterID, userID)
	if err != nil {
		return nil, err
	}

	if f == nil || f.Status != "pending" || f.RequesterID != requesterID {
		return nil, ErrFriendRequestNotFound
	}

	if err := s.friendRepo.UpdateStatus(requesterID, userID, "accepted"); err != nil {
		return nil, err
	}

	f.Status = "accepted"

	// WS: 通知申請方已被接受
	if s.notifier != nil {
		s.notifier.BroadcastToUser(requesterID, "friend_accept", f)
	}

	return f, nil
}

func (s *friendService) Reject(userID, requesterID uint) error {
	f, err := s.friendRepo.GetBetween(requesterID, userID)
	if err != nil {
		return err
	}

	if f == nil || f.Status != "pending" || f.RequesterID != requesterID {
		return ErrFriendRequestNotFound
	}

	if err := s.friendRepo.Delete(requesterID, userID); err != nil {
		return err
	}

	// WS: 通知申請方已被拒絕
	if s.notifier != nil {
		s.notifier.BroadcastToUser(requesterID, "friend_reject", map[string]uint{"user_id": userID})
	}

	return nil
}

func (s *friendService) Unfriend(userID, friendID uint) error {
	if err := s.friendRepo.Delete(userID, friendID); err != nil {
		return err
	}

	// WS: 雙向通知
	if s.notifier != nil {
		s.notifier.BroadcastToUser(friendID, "friend_remove", map[string]uint{"user_id": userID})
	}

	return nil
}

func (s *friendService) ListFriends(userID uint) ([]*model.Friendship, error) {
	return s.friendRepo.ListFriends(userID)
}

func (s *friendService) ListIncomingRequests(userID uint) ([]*model.Friendship, error) {
	return s.friendRepo.ListIncomingRequests(userID)
}

func (s *friendService) ListOutgoingRequests(userID uint) ([]*model.Friendship, error) {
	return s.friendRepo.ListOutgoingRequests(userID)
}
