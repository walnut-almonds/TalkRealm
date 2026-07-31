package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func TestFeedService_Follow_RejectsSelf(t *testing.T) {
	svc := service.NewFeedService(
		&testutil.MockFollowRepository{}, nil, nil, nil, nil,
	)
	err := svc.Follow(5, 5)
	require.Error(t, err)
}

func TestFeedService_Suggestions_ExcludesAlreadyFollowed(t *testing.T) {
	follow := &testutil.MockFollowRepository{
		FolloweeIDsFn: func(_ uint) ([]uint, error) { return []uint{2}, nil }, // already follows 2
	}
	friends := &testutil.MockFriendshipRepository{
		ListFriendsFn: func(_ uint) ([]*model.Friendship, error) {
			// friends: users 2 and 3, preloaded on the opposite side of self (5)
			return []*model.Friendship{
				{RequesterID: 5, AddresseeID: 2, Addressee: model.User{ID: 2, Username: "b"}},
				{RequesterID: 3, AddresseeID: 5, Requester: model.User{ID: 3, Username: "c"}},
			}, nil
		},
	}
	svc := service.NewFeedService(follow, nil, nil, nil, friends)
	sugg, err := svc.Suggestions(5)
	require.NoError(t, err)
	// user 2 already followed -> only user 3 suggested
	require.Len(t, sugg, 1)
	assert.Equal(t, uint(3), sugg[0].ID)
}

func TestFeedService_Timeline_IncludesSelfAndEnriches(t *testing.T) {
	var gotAuthorIDs []uint

	post := &model.FeedPost{ID: 9, AuthorID: 2}
	follow := &testutil.MockFollowRepository{
		FolloweeIDsFn: func(_ uint) ([]uint, error) { return []uint{2, 3}, nil },
	}
	posts := &testutil.MockFeedPostRepository{
		TimelineCursorFn: func(ids []uint, _ uint, _ int) ([]*model.FeedPost, error) {
			gotAuthorIDs = ids
			return []*model.FeedPost{post}, nil
		},
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CountByPostIDsFn: func(_ []uint) (map[uint]int64, error) { return map[uint]int64{9: 4}, nil },
		LikedPostIDsFn:   func(_ uint, _ []uint) (map[uint]bool, error) { return map[uint]bool{9: true}, nil },
	}
	comments := &testutil.MockFeedCommentRepository{
		CountByPostIDsFn: func(_ []uint) (map[uint]int64, error) { return map[uint]int64{9: 2}, nil },
	}
	svc := service.NewFeedService(follow, posts, likes, comments, nil)

	resp, err := svc.Timeline(5, 0, 20)
	require.NoError(t, err)
	assert.Contains(t, gotAuthorIDs, uint(5)) // self included
	require.Len(t, resp.Posts, 1)
	assert.Equal(t, int64(4), resp.Posts[0].LikeCount)
	assert.Equal(t, int64(2), resp.Posts[0].CommentCount)
	assert.True(t, resp.Posts[0].LikedByMe)
}

func TestFeedService_DeletePost_OwnerCascades(t *testing.T) {
	cascaded := false
	posts := &testutil.MockFeedPostRepository{
		GetByIDFn:       func(_ uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7, AuthorID: 5}, nil },
		DeleteCascadeFn: func(_ uint) error { cascaded = true; return nil },
	}
	svc := service.NewFeedService(nil, posts, nil, nil, nil)
	require.NoError(t, svc.DeletePost(7, 5))
	assert.True(t, cascaded)
}

func TestFeedService_DeletePost_NonOwnerRejected(t *testing.T) {
	posts := &testutil.MockFeedPostRepository{
		GetByIDFn: func(_ uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7, AuthorID: 99}, nil },
	}
	svc := service.NewFeedService(nil, posts, nil, nil, nil)
	require.ErrorIs(t, svc.DeletePost(7, 5), service.ErrNotFeedOwner)
}

func TestFeedService_LikePost_ReturnsCount(t *testing.T) {
	posts := &testutil.MockFeedPostRepository{
		GetByIDFn: func(_ uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7}, nil },
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CreateFn:        func(_ *model.FeedPostLike) error { return nil },
		CountByPostIDFn: func(_ uint) (int64, error) { return 3, nil },
	}
	svc := service.NewFeedService(nil, posts, likes, nil, nil)
	n, err := svc.LikePost(7, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestFeedService_LikeComment_ReturnsCount(t *testing.T) {
	comments := &testutil.MockFeedCommentRepository{
		GetByIDFn: func(_ uint) (*model.FeedComment, error) { return &model.FeedComment{ID: 7}, nil },
	}
	likes := &testutil.MockFeedCommentLikeRepository{
		CreateFn:           func(_ *model.FeedCommentLike) error { return nil },
		CountByCommentIDFn: func(_ uint) (int64, error) { return 3, nil },
	}
	svc := service.NewFeedService(nil, nil, nil, comments, nil)
	svc.SetCommentLikeRepo(likes)
	n, err := svc.LikeComment(7, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestFeedService_DiscoverTimeline_RanksAndPaginates(t *testing.T) {
	now := time.Now()
	// two candidates: post 8 by author 3 (high affinity), post 9 by author 2 (no affinity)
	cands := []*model.FeedPost{
		{ID: 9, AuthorID: 2, CreatedAt: now.Add(-time.Hour)},
		{ID: 8, AuthorID: 3, CreatedAt: now.Add(-time.Hour)},
	}
	posts := &testutil.MockFeedPostRepository{
		RecentCandidatesFn: func(excl uint, since time.Time, limit int) ([]*model.FeedPost, error) {
			assert.Equal(t, uint(5), excl) // excludes viewer's own posts
			return cands, nil
		},
		AuthorAffinityFn: func(viewerID uint) (map[uint]int64, error) { return map[uint]int64{3: 8}, nil },
		GetByIDFn: func(id uint) (*model.FeedPost, error) {
			for _, p := range cands {
				if p.ID == id {
					return p, nil
				}
			}

			return nil, service.ErrFeedPostNotFound
		},
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{}, nil },
		LikedPostIDsFn:   func(uid uint, ids []uint) (map[uint]bool, error) { return map[uint]bool{}, nil },
	}
	comments := &testutil.MockFeedCommentRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{}, nil },
	}
	svc := service.NewFeedService(nil, posts, likes, comments, nil)

	resp, err := svc.DiscoverTimeline(5, 0, 20)
	require.NoError(t, err)
	require.Len(t, resp.Posts, 2)
	// author 3 has affinity 8 → post 8 outranks post 9 (same time, no engagement)
	assert.Equal(t, uint(8), resp.Posts[0].ID)
	assert.False(t, resp.HasMore)

	// offset pagination
	page2, err := svc.DiscoverTimeline(5, 1, 1)
	require.NoError(t, err)
	require.Len(t, page2.Posts, 1)
	assert.Equal(t, uint(9), page2.Posts[0].ID)
}
