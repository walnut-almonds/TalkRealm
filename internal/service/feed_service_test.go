package service_test

import (
	"testing"

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
