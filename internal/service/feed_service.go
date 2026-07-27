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

// FeedPostResponse 貼文加上按讚/留言統計的 enrichment DTO
type FeedPostResponse struct {
	*model.FeedPost
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	LikedByMe    bool  `json:"liked_by_me"`
}

// TimelineResponse 時間軸/個人頁貼文列表回應
type TimelineResponse struct {
	Posts   []*FeedPostResponse `json:"posts"`
	HasMore bool                `json:"has_more"`
}

// FeedService 個人動態牆服務（follow 部分，貼文/留言方法於 Task 6-7 加入）
type FeedService interface {
	Follow(followerID, followeeID uint) error
	Unfollow(followerID, followeeID uint) error
	Suggestions(userID uint) ([]*model.User, error)
	ListFollowing(userID uint) (*FollowListResponse, error)
	ListFollowers(userID uint) (*FollowListResponse, error)
	CreatePost(authorID uint, content string, fileIDs []uint) (*FeedPostResponse, error)
	Timeline(userID uint, before uint, limit int) (*TimelineResponse, error)
	ProfilePosts(targetID, viewerID uint, before uint, limit int) (*TimelineResponse, error)
	// comment/like methods added in Task 7
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

func (s *feedService) CreatePost(authorID uint, content string, fileIDs []uint) (*FeedPostResponse, error) {
	if content == "" && len(fileIDs) == 0 {
		return nil, errors.New("empty post")
	}

	p := &model.FeedPost{AuthorID: authorID, Content: content}
	if err := s.postRepo.Create(p); err != nil {
		return nil, err
	}

	if err := s.postRepo.AttachFiles(p.ID, fileIDs); err != nil {
		return nil, err
	}

	full, err := s.postRepo.GetByID(p.ID)
	if err != nil {
		return nil, err
	}

	return s.enrich([]*model.FeedPost{full}, authorID)[0], nil
}

func (s *feedService) Timeline(userID uint, before uint, limit int) (*TimelineResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ids, err := s.followRepo.FolloweeIDs(userID)
	if err != nil {
		return nil, err
	}

	ids = append(ids, userID) // include self

	posts, err := s.postRepo.TimelineCursor(ids, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	return &TimelineResponse{Posts: s.enrich(posts, userID), HasMore: hasMore}, nil
}

func (s *feedService) ProfilePosts(targetID, viewerID uint, before uint, limit int) (*TimelineResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	posts, err := s.postRepo.ByAuthorCursor(targetID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	return &TimelineResponse{Posts: s.enrich(posts, viewerID), HasMore: hasMore}, nil
}

// enrich 批次補上按讚/留言統計與 viewer 是否已讚，避免 N+1。
// likeRepo/commentRepo 可為 nil（部分呼叫端不需要）。
func (s *feedService) enrich(posts []*model.FeedPost, viewerID uint) []*FeedPostResponse {
	ids := make([]uint, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	var likeCounts, commentCounts map[uint]int64

	var liked map[uint]bool

	if s.likeRepo != nil {
		likeCounts, _ = s.likeRepo.CountByPostIDs(ids)
		liked, _ = s.likeRepo.LikedPostIDs(viewerID, ids)
	}

	if s.commentRepo != nil {
		commentCounts, _ = s.commentRepo.CountByPostIDs(ids)
	}

	out := make([]*FeedPostResponse, len(posts))
	for i, p := range posts {
		out[i] = &FeedPostResponse{
			FeedPost:     p,
			LikeCount:    likeCounts[p.ID],
			CommentCount: commentCounts[p.ID],
			LikedByMe:    liked[p.ID],
		}
	}

	return out
}
