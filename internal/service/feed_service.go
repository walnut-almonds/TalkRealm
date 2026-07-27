package service

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	ErrFeedPostNotFound = errors.New("feed post not found")
	ErrNotFeedOwner     = errors.New("not the owner")
)

// FollowListResponse 追蹤者/被追蹤者列表回應
type FollowListResponse struct {
	Users []*model.User `json:"users"`
	Count int64         `json:"count"`
}

// FeedService 個人動態牆服務（follow 部分，貼文/留言方法於 Task 6-7 加入）
type FeedService interface {
	Follow(followerID, followeeID uint) error
	Unfollow(followerID, followeeID uint) error
	Suggestions(userID uint) ([]*model.User, error)
	ListFollowing(userID uint) (*FollowListResponse, error)
	ListFollowers(userID uint) (*FollowListResponse, error)
	// post/comment methods added in Tasks 6-7
}

type feedService struct {
	followRepo  repository.FollowRepository
	postRepo    repository.FeedPostRepository
	likeRepo    repository.FeedPostLikeRepository
	commentRepo repository.FeedCommentRepository
	friendRepo  repository.FriendshipRepository
}

// NewFeedService 建立 feed 服務。參數順序固定，後續 task 依賴此 5 參數順序。
func NewFeedService(
	followRepo repository.FollowRepository,
	postRepo repository.FeedPostRepository,
	likeRepo repository.FeedPostLikeRepository,
	commentRepo repository.FeedCommentRepository,
	friendRepo repository.FriendshipRepository,
) FeedService {
	return &feedService{followRepo, postRepo, likeRepo, commentRepo, friendRepo}
}

func (s *feedService) Follow(followerID, followeeID uint) error {
	if followerID == followeeID {
		return ErrCannotFollowSelf
	}

	return s.followRepo.Follow(followerID, followeeID)
}

func (s *feedService) Unfollow(followerID, followeeID uint) error {
	return s.followRepo.Unfollow(followerID, followeeID)
}

func (s *feedService) Suggestions(userID uint) ([]*model.User, error) {
	friends, err := s.friendRepo.ListFriends(userID)
	if err != nil {
		return nil, err
	}

	followeeIDs, err := s.followRepo.FolloweeIDs(userID)
	if err != nil {
		return nil, err
	}

	followed := make(map[uint]bool, len(followeeIDs))
	for _, id := range followeeIDs {
		followed[id] = true
	}

	var out []*model.User
	for _, f := range friends {
		other := friendCounterpart(f, userID)
		if other == nil || other.ID == userID || followed[other.ID] {
			continue
		}
		out = append(out, other)
	}

	return out, nil
}

func (s *feedService) ListFollowing(userID uint) (*FollowListResponse, error) {
	rows, err := s.followRepo.ListFollowing(userID)
	if err != nil {
		return nil, err
	}

	cnt, _ := s.followRepo.CountFollowing(userID)
	users := make([]*model.User, 0, len(rows))
	for _, r := range rows {
		u := r.Followee
		users = append(users, &u)
	}

	return &FollowListResponse{Users: users, Count: cnt}, nil
}

func (s *feedService) ListFollowers(userID uint) (*FollowListResponse, error) {
	rows, err := s.followRepo.ListFollowers(userID)
	if err != nil {
		return nil, err
	}

	cnt, _ := s.followRepo.CountFollowers(userID)
	users := make([]*model.User, 0, len(rows))
	for _, r := range rows {
		u := r.Follower
		users = append(users, &u)
	}

	return &FollowListResponse{Users: users, Count: cnt}, nil
}

// friendCounterpart 回傳好友關係中「非 self」那一側的預載 User。
// Friendship 的兩側為 RequesterID/Requester 與 AddresseeID/Addressee，
// ListFriends 已 Preload 兩側 User。
func friendCounterpart(f *model.Friendship, self uint) *model.User {
	if f.RequesterID == self {
		return &f.Addressee
	}

	return &f.Requester
}
