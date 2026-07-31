package service

import (
	"errors"
	"sort"
	"time"

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
	Timeline(userID, before uint, limit int) (*TimelineResponse, error)
	DiscoverTimeline(viewerID uint, offset, limit int) (*TimelineResponse, error)
	ProfilePosts(targetID, viewerID, before uint, limit int) (*TimelineResponse, error)
	UpdatePost(postID, userID uint, content string) (*FeedPostResponse, error)
	DeletePost(postID, userID uint) error
	LikePost(postID, userID uint) (int64, error)
	UnlikePost(postID, userID uint) (int64, error)
	LikeComment(commentID, userID uint) (int64, error)
	UnlikeComment(commentID, userID uint) (int64, error)
	SetCommentLikeRepo(r repository.FeedCommentLikeRepository)
	SetWebSocketManager(m WebSocketManager)
	ListComments(postID, viewerID, before uint, limit int) (*CommentListResponse, error)
	AddComment(postID, authorID uint, content string) (*model.FeedComment, error)
	UpdateComment(commentID, userID uint, content string) (*model.FeedComment, error)
	DeleteComment(commentID, userID uint) error
}

type feedService struct {
	followRepo      repository.FollowRepository
	postRepo        repository.FeedPostRepository
	likeRepo        repository.FeedPostLikeRepository
	commentRepo     repository.FeedCommentRepository
	friendRepo      repository.FriendshipRepository
	commentLikeRepo repository.FeedCommentLikeRepository
	wsManager       WebSocketManager
}

// SetCommentLikeRepo 注入留言按讚 repository（避免更動 5 參數的 NewFeedService 簽名）。
func (s *feedService) SetCommentLikeRepo(r repository.FeedCommentLikeRepository) {
	s.commentLikeRepo = r
}

func (s *feedService) SetWebSocketManager(m WebSocketManager) { s.wsManager = m }

// broadcastToFollowers pushes an event to every follower of authorID (and to the
// author too when includeAuthor). Fan-out of a signal only — not materialization.
// ponytail: synchronous fan-out; move to a goroutine/queue only if follower counts get large.
func (s *feedService) broadcastToFollowers(
	authorID uint,
	includeAuthor bool,
	event string,
	data any,
) {
	if s.wsManager == nil {
		return
	}

	ids, err := s.followRepo.FollowerIDs(authorID)
	if err != nil {
		return
	}

	for _, uid := range ids {
		s.wsManager.BroadcastToUser(uid, event, data)
	}

	if includeAuthor {
		s.wsManager.BroadcastToUser(authorID, event, data)
	}
}

// NewFeedService 建立 feed 服務。參數順序固定，後續 task 依賴此 5 參數順序。
func NewFeedService(
	followRepo repository.FollowRepository,
	postRepo repository.FeedPostRepository,
	likeRepo repository.FeedPostLikeRepository,
	commentRepo repository.FeedCommentRepository,
	friendRepo repository.FriendshipRepository,
) FeedService {
	return &feedService{
		followRepo:  followRepo,
		postRepo:    postRepo,
		likeRepo:    likeRepo,
		commentRepo: commentRepo,
		friendRepo:  friendRepo,
	}
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

func (s *feedService) CreatePost(
	authorID uint,
	content string,
	fileIDs []uint,
) (*FeedPostResponse, error) {
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

	resp := s.enrich([]*model.FeedPost{full}, authorID)[0]

	s.broadcastToFollowers(authorID, false, "feed_new_post", map[string]any{"author_id": authorID})

	return resp, nil
}

func (s *feedService) Timeline(userID, before uint, limit int) (*TimelineResponse, error) {
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

func (s *feedService) DiscoverTimeline(
	viewerID uint,
	offset, limit int,
) (*TimelineResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if offset < 0 {
		offset = 0
	}

	since := time.Now().AddDate(0, 0, -DiscoverWindowDays)

	candidates, err := s.postRepo.RecentCandidates(viewerID, since, DiscoverPoolSize)
	if err != nil {
		return nil, err
	}

	affinity, err := s.postRepo.AuthorAffinity(viewerID)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, len(candidates))
	for i, p := range candidates {
		ids[i] = p.ID
	}

	likeCounts, _ := s.likeRepo.CountByPostIDs(ids)

	commentCounts, _ := s.commentRepo.CountByPostIDs(ids)

	now := time.Now()

	type scored struct {
		p *model.FeedPost
		s float64
	}

	arr := make([]scored, len(candidates))
	for i, p := range candidates {
		arr[i] = scored{
			p,
			scorePost(
				p.ID,
				likeCounts[p.ID],
				commentCounts[p.ID],
				affinity[p.AuthorID],
				p.CreatedAt,
				now,
			),
		}
	}

	sort.SliceStable(arr, func(i, j int) bool { return arr[i].s > arr[j].s })

	hasMore := offset+limit < len(arr)
	if offset > len(arr) {
		offset = len(arr)
	}

	end := offset + limit
	if end > len(arr) {
		end = len(arr)
	}

	page := make([]*model.FeedPost, 0, end-offset)
	for _, x := range arr[offset:end] {
		page = append(page, x.p)
	}

	return &TimelineResponse{Posts: s.enrich(page, viewerID), HasMore: hasMore}, nil
}

func (s *feedService) ProfilePosts(
	targetID, viewerID, before uint,
	limit int,
) (*TimelineResponse, error) {
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

// FeedCommentResponse 留言加上按讚統計與 viewer 是否已讚的 enrichment DTO
type FeedCommentResponse struct {
	*model.FeedComment
	LikeCount int64 `json:"like_count"`
	LikedByMe bool  `json:"liked_by_me"`
}

// CommentListResponse 貼文留言列表回應
type CommentListResponse struct {
	Comments []*FeedCommentResponse `json:"comments"`
	HasMore  bool                   `json:"has_more"`
}

// getOwnedPost 取貼文並驗證 userID 為作者，非作者回 ErrNotFeedOwner。
func (s *feedService) getOwnedPost(postID, userID uint) (*model.FeedPost, error) {
	p, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, ErrFeedPostNotFound
	}

	if p.AuthorID != userID {
		return nil, ErrNotFeedOwner
	}

	return p, nil
}

func (s *feedService) UpdatePost(postID, userID uint, content string) (*FeedPostResponse, error) {
	p, err := s.getOwnedPost(postID, userID)
	if err != nil {
		return nil, err
	}

	p.Content = content

	p.IsEdited = true
	if err := s.postRepo.Update(p); err != nil {
		return nil, err
	}

	full, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, err
	}

	return s.enrich([]*model.FeedPost{full}, userID)[0], nil
}

func (s *feedService) DeletePost(postID, userID uint) error {
	if _, err := s.getOwnedPost(postID, userID); err != nil {
		return err
	}

	return s.postRepo.DeleteCascade(postID)
}

func (s *feedService) LikePost(postID, userID uint) (int64, error) {
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return 0, ErrFeedPostNotFound
	}

	if err := s.likeRepo.Create(&model.FeedPostLike{PostID: postID, UserID: userID}); err != nil {
		return 0, err
	}

	n, err := s.likeRepo.CountByPostID(postID)
	if err != nil {
		return 0, err
	}

	s.broadcastToFollowers(
		post.AuthorID,
		true,
		"feed_post_like",
		map[string]any{"post_id": postID, "like_count": n},
	)

	return n, nil
}

func (s *feedService) UnlikePost(postID, userID uint) (int64, error) {
	post, postErr := s.postRepo.GetByID(postID)

	if err := s.likeRepo.Delete(postID, userID); err != nil {
		return 0, err
	}

	n, err := s.likeRepo.CountByPostID(postID)
	if err != nil {
		return 0, err
	}

	if postErr == nil {
		s.broadcastToFollowers(
			post.AuthorID,
			true,
			"feed_post_like",
			map[string]any{"post_id": postID, "like_count": n},
		)
	}

	return n, nil
}

func (s *feedService) ListComments(
	postID, viewerID, before uint,
	limit int,
) (*CommentListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	cs, err := s.commentRepo.ByPostCursor(postID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := false
	if len(cs) > limit {
		hasMore = true
		cs = cs[len(cs)-limit:]
	}

	return &CommentListResponse{Comments: s.enrichComments(cs, viewerID), HasMore: hasMore}, nil
}

// enrichComments 批次補上留言按讚數與 viewer 是否已讚，避免 N+1。
// commentLikeRepo 可為 nil（部分呼叫端未注入）。
func (s *feedService) enrichComments(
	cs []*model.FeedComment,
	viewerID uint,
) []*FeedCommentResponse {
	ids := make([]uint, len(cs))
	for i, c := range cs {
		ids[i] = c.ID
	}

	var likeCounts map[uint]int64

	var liked map[uint]bool

	if s.commentLikeRepo != nil {
		likeCounts, _ = s.commentLikeRepo.CountByCommentIDs(ids)
		liked, _ = s.commentLikeRepo.LikedCommentIDs(viewerID, ids)
	}

	out := make([]*FeedCommentResponse, len(cs))
	for i, c := range cs {
		out[i] = &FeedCommentResponse{
			FeedComment: c,
			LikeCount:   likeCounts[c.ID],
			LikedByMe:   liked[c.ID],
		}
	}

	return out
}

func (s *feedService) LikeComment(commentID, userID uint) (int64, error) {
	if _, err := s.commentRepo.GetByID(commentID); err != nil {
		return 0, ErrFeedPostNotFound
	}

	if err := s.commentLikeRepo.Create(
		&model.FeedCommentLike{CommentID: commentID, UserID: userID},
	); err != nil {
		return 0, err
	}

	return s.commentLikeRepo.CountByCommentID(commentID)
}

func (s *feedService) UnlikeComment(commentID, userID uint) (int64, error) {
	if err := s.commentLikeRepo.Delete(commentID, userID); err != nil {
		return 0, err
	}

	return s.commentLikeRepo.CountByCommentID(commentID)
}

func (s *feedService) AddComment(
	postID, authorID uint,
	content string,
) (*model.FeedComment, error) {
	if content == "" {
		return nil, errors.New("empty comment")
	}

	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, ErrFeedPostNotFound
	}

	c := &model.FeedComment{PostID: postID, AuthorID: authorID, Content: content}
	if err := s.commentRepo.Create(c); err != nil {
		return nil, err
	}

	full, err := s.commentRepo.GetByID(c.ID)
	if err != nil {
		return nil, err
	}

	cnt := int64(0)
	if counts, cerr := s.commentRepo.CountByPostIDs([]uint{postID}); cerr == nil {
		cnt = counts[postID]
	}

	s.broadcastToFollowers(
		post.AuthorID,
		true,
		"feed_comment_count",
		map[string]any{"post_id": postID, "comment_count": cnt},
	)

	return full, nil
}

func (s *feedService) UpdateComment(
	commentID, userID uint,
	content string,
) (*model.FeedComment, error) {
	c, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	if c.AuthorID != userID {
		return nil, ErrNotFeedOwner
	}

	c.Content = content

	c.IsEdited = true
	if err := s.commentRepo.Update(c); err != nil {
		return nil, err
	}

	return s.commentRepo.GetByID(commentID)
}

func (s *feedService) DeleteComment(commentID, userID uint) error {
	c, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return errors.New("comment not found")
	}

	if c.AuthorID != userID {
		return ErrNotFeedOwner
	}

	return s.commentRepo.Delete(commentID)
}
