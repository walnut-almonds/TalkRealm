package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func TestMessageService_CreateMessage_DeduplicatesMentions(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "text"}
	member := &model.GuildMember{ID: 1, GuildID: 10, UserID: 5}

	done := make(chan []*model.MessageMention, 1)

	fullMsg := &model.Message{
		ID:        1,
		ChannelID: 1,
		UserID:    5,
		Content:   "hello <@2> and @everyone and again <@2>",
		User: model.User{
			ID:       5,
			Username: "author",
		},
	}

	mockMsg := &testutil.MockMessageRepository{
		CreateFn: func(m *model.Message) error {
			m.ID = 1
			return nil
		},
		GetByIDFn: func(id uint) (*model.Message, error) {
			return fullMsg, nil
		},
	}

	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}

	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return member, nil
		},
		GetByGuildIDFn: func(guildID uint) ([]*model.GuildMember, error) {
			return []*model.GuildMember{
				{UserID: 2},
				{UserID: 3},
				{UserID: 4},
				{UserID: 5},
			}, nil
		},
	}

	mockMention := &testutil.MockMessageMentionRepository{
		BulkCreateFn: func(mentions []*model.MessageMention) error {
			done <- mentions
			return nil
		},
	}

	svc := service.NewMessageService(mockMsg, mockCh, mockMember, mockMention)

	_, err := svc.CreateMessage(
		5,
		&service.CreateMessageRequest{ChannelID: 1, Content: fullMsg.Content},
	)
	require.NoError(t, err)

	select {
	case mentions := <-done:
		require.Len(t, mentions, 3)

		byUser := map[uint]string{}
		for _, m := range mentions {
			byUser[m.UserID] = m.MentionType
		}

		assert.Equal(t, "user", byUser[2])
		assert.Equal(t, "everyone", byUser[3])
		assert.Equal(t, "everyone", byUser[4])
	case <-time.After(300 * time.Millisecond):
		t.Fatal("mention processing did not complete in time")
	}
}

func TestMessageService_CreateMessage_Success(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	member := &model.GuildMember{ID: 1, GuildID: 10, UserID: 5}
	msg := &model.Message{ID: 1, Content: "hello", ChannelID: 1, UserID: 5, CreatedAt: time.Now()}

	mockMsg := &testutil.MockMessageRepository{
		CreateFn:  func(m *model.Message) error { m.ID = 1; return nil },
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}

	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	result, err := svc.CreateMessage(
		5,
		&service.CreateMessageRequest{ChannelID: 1, Content: "hello"},
	)
	require.NoError(t, err)
	assert.Equal(t, "hello", result.Content)
}

func TestMessageService_CreateMessage_EmptyContent(t *testing.T) {
	svc := service.NewMessageService(
		&testutil.MockMessageRepository{},
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
		nil,
	)

	_, err := svc.CreateMessage(1, &service.CreateMessageRequest{ChannelID: 1, Content: ""})
	assert.ErrorIs(t, err, service.ErrEmptyMessageContent)
}

func TestMessageService_CreateMessage_InvalidType(t *testing.T) {
	svc := service.NewMessageService(
		&testutil.MockMessageRepository{},
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
		nil,
	)

	_, err := svc.CreateMessage(
		1,
		&service.CreateMessageRequest{ChannelID: 1, Content: "hi", Type: "invalid"},
	)
	assert.ErrorIs(t, err, service.ErrInvalidMessageType)
}

func TestMessageService_CreateMessage_ChannelNotFound(t *testing.T) {
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return nil, errors.New("not found") },
	}
	svc := service.NewMessageService(
		&testutil.MockMessageRepository{},
		mockCh,
		&testutil.MockGuildMemberRepository{},
		nil,
	)

	_, err := svc.CreateMessage(1, &service.CreateMessageRequest{ChannelID: 99, Content: "hi"})
	assert.Error(t, err)
}

func TestMessageService_CreateMessage_NotMember(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return nil, errors.New("not found") },
	}
	svc := service.NewMessageService(&testutil.MockMessageRepository{}, mockCh, mockMember, nil)

	_, err := svc.CreateMessage(99, &service.CreateMessageRequest{ChannelID: 1, Content: "hi"})
	assert.ErrorIs(t, err, service.ErrNotChannelMemberMsg)
}

func TestMessageService_CreateMessage_WithWSManager(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	member := &model.GuildMember{ID: 1, GuildID: 10, UserID: 5}
	savedMsg := &model.Message{ID: 1, Content: "hi", ChannelID: 1, UserID: 5}

	broadcastCalled := false
	mockMsg := &testutil.MockMessageRepository{
		CreateFn:  func(m *model.Message) error { m.ID = 1; return nil },
		GetByIDFn: func(id uint) (*model.Message, error) { return savedMsg, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}

	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)
	svc.SetWebSocketManager(&mockWSManager{
		broadcastFn: func(channelID uint, msgType string, data any) { broadcastCalled = true },
	})

	_, err := svc.CreateMessage(5, &service.CreateMessageRequest{ChannelID: 1, Content: "hi"})
	require.NoError(t, err)
	assert.True(t, broadcastCalled)
}

func TestMessageService_GetMessage_Success(t *testing.T) {
	msg := &model.Message{ID: 1, ChannelID: 1, UserID: 5}
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	member := &model.GuildMember{ID: 1}

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	got, err := svc.GetMessage(1, 5)
	require.NoError(t, err)
	assert.Equal(t, uint(1), got.ID)
}

func TestMessageService_GetMessage_NotFound(t *testing.T) {
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return nil, errors.New("not found") },
	}
	svc := service.NewMessageService(
		mockMsg,
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
		nil,
	)

	_, err := svc.GetMessage(999, 1)
	assert.ErrorIs(t, err, service.ErrMessageNotFound)
}

func TestMessageService_ListChannelMessages_Success(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	member := &model.GuildMember{ID: 1}
	msgs := []*model.Message{{ID: 3}, {ID: 2}, {ID: 1}}

	mockMsg := &testutil.MockMessageRepository{
		GetByChannelIDCursorFn: func(channelID, before uint, limit int) ([]*model.Message, error) {
			return msgs, nil
		},
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	resp, err := svc.ListChannelMessages(1, 1, 50, 0)
	require.NoError(t, err)
	assert.Len(t, resp.Messages, 3)
}

func TestMessageService_UpdateMessage_Success(t *testing.T) {
	msg := &model.Message{ID: 1, UserID: 5, Content: "old"}
	updated := false

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
		UpdateFn:  func(m *model.Message) error { updated = true; return nil },
	}
	svc := service.NewMessageService(
		mockMsg,
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
		nil,
	)

	got, err := svc.UpdateMessage(1, 5, &service.UpdateMessageRequest{Content: "new content"})
	require.NoError(t, err)
	assert.Equal(t, "new content", got.Content)
	assert.True(t, updated)
}

func TestMessageService_UpdateMessage_NotOwner(t *testing.T) {
	msg := &model.Message{ID: 1, UserID: 99}
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
	}
	svc := service.NewMessageService(
		mockMsg,
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
		nil,
	)

	_, err := svc.UpdateMessage(1, 5, &service.UpdateMessageRequest{Content: "new"})
	assert.ErrorIs(t, err, service.ErrNotMessageOwner)
}

func TestMessageService_DeleteMessage_Success(t *testing.T) {
	msg := &model.Message{ID: 1, UserID: 5, ChannelID: 1}
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	member := &model.GuildMember{ID: 1, UserID: 5, Role: "member"}
	deleted := false

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
		DeleteFn:  func(id uint) error { deleted = true; return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	err := svc.DeleteMessage(1, 5)
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestMessageService_DeleteMessage_NotOwnerAndNotAdmin(t *testing.T) {
	msg := &model.Message{ID: 1, UserID: 99, ChannelID: 1}
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10)}
	member := &model.GuildMember{ID: 1, UserID: 5, Role: "member"}

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	err := svc.DeleteMessage(1, 5)
	assert.ErrorIs(t, err, service.ErrNotMessageOwner)
}

func TestMessageService_DeleteMessage_FeedPostCascades(t *testing.T) {
	post := &model.Message{ID: 7, ChannelID: 1, UserID: 5, ParentID: nil}
	cascaded := false
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn:           func(id uint) (*model.Message, error) { return post, nil },
		DeletePostCascadeFn: func(postID uint) error { cascaded = true; return nil },
		DeleteFn:            func(id uint) error { t.Fatalf("plain Delete must not be used for a feed post"); return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	svc := service.NewMessageService(mockMsg, mockCh, &testutil.MockGuildMemberRepository{}, nil)

	require.NoError(t, svc.DeleteMessage(7, 5))
	assert.True(t, cascaded)
}

func TestMessageService_DeleteMessage_CommentClearsOwnLikes(t *testing.T) {
	comment := &model.Message{ID: 8, ChannelID: 1, UserID: 5, ParentID: testutil.PtrUint(7)}

	var clearedIDs []uint

	deleted := false
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return comment, nil },
		DeleteFn:  func(id uint) error { deleted = true; return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	mockLike := &testutil.MockMessageLikeRepository{
		DeleteByMessageIDsFn: func(ids []uint) error { clearedIDs = ids; return nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, &testutil.MockGuildMemberRepository{}, nil)
	svc.SetLikeRepo(mockLike)

	require.NoError(t, svc.DeleteMessage(8, 5))
	assert.True(t, deleted)
	assert.Equal(t, []uint{8}, clearedIDs)
}

// ---------------------------------------------------------------------------
// mockWSManager helper
// ---------------------------------------------------------------------------

type mockWSManager struct {
	broadcastFn     func(channelID uint, msgType string, data any)
	broadcastUserFn func(userID uint, msgType string, data any)
	isOnlineFn      func(userID uint) bool
}

func (m *mockWSManager) BroadcastToChannel(channelID uint, msgType string, data any) {
	if m.broadcastFn != nil {
		m.broadcastFn(channelID, msgType, data)
	}
}

func (m *mockWSManager) BroadcastToUser(userID uint, msgType string, data any) {
	if m.broadcastUserFn != nil {
		m.broadcastUserFn(userID, msgType, data)
	}
}

func (m *mockWSManager) IsUserOnline(userID uint) bool {
	if m.isOnlineFn != nil {
		return m.isOnlineFn(userID)
	}

	return false
}

func TestMessageService_CreateMessage_SetsParentID(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}
	post := &model.Message{ID: 7, ChannelID: 1, ParentID: nil}

	var created *model.Message

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) {
			if id == 7 {
				return post, nil
			}

			return created, nil
		},
		CreateFn: func(m *model.Message) error { m.ID = 8; created = m; return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	parent := uint(7)
	_, err := svc.CreateMessage(5, &service.CreateMessageRequest{
		ChannelID: 1, Content: "nice post", ParentID: &parent,
	})
	require.NoError(t, err)
	require.NotNil(t, created.ParentID)
	assert.Equal(t, uint(7), *created.ParentID)
}

func TestMessageService_CreateMessage_RejectsCommentOnComment(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}
	comment := &model.Message{
		ID:        7,
		ChannelID: 1,
		ParentID:  testutil.PtrUint(3),
	} // already a comment

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return comment, nil },
		CreateFn:  func(m *model.Message) error { return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	parent := uint(7)
	_, err := svc.CreateMessage(
		5,
		&service.CreateMessageRequest{ChannelID: 1, Content: "x", ParentID: &parent},
	)
	require.Error(t, err)
}

func TestMessageService_LikePost_ReturnsCountAndBroadcasts(t *testing.T) {
	msg := &model.Message{ID: 7, ChannelID: 1}
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	mockLike := &testutil.MockMessageLikeRepository{
		CreateFn:           func(l *model.MessageLike) error { return nil },
		CountByMessageIDFn: func(id uint) (int64, error) { return 4, nil },
	}
	ws := &testutil.MockWebSocketManager{}

	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)
	svc.SetLikeRepo(mockLike)
	svc.SetWebSocketManager(ws)

	n, err := svc.LikePost(7, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(4), n)
	require.Len(t, ws.ChannelBroadcasts, 1)
	assert.Equal(t, "post_like", ws.ChannelBroadcasts[0].MsgType)
	assert.Equal(t, uint(1), ws.ChannelBroadcasts[0].ID)
}

func TestMessageService_ListPosts_EnrichesCounts(t *testing.T) {
	posts := []*model.Message{{ID: 7, ChannelID: 1}, {ID: 6, ChannelID: 1}}
	mockMsg := &testutil.MockMessageRepository{
		GetPostsByChannelCursorFn: func(cid, before uint, limit int) ([]*model.Message, error) { return posts, nil },
		CountCommentsByPostIDsFn:  func(ids []uint) (map[uint]int64, error) { return map[uint]int64{7: 2}, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	mockLike := &testutil.MockMessageLikeRepository{
		CountByMessageIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{7: 4}, nil },
		LikedMessageIDsFn:   func(uid uint, ids []uint) (map[uint]bool, error) { return map[uint]bool{7: true}, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)
	svc.SetLikeRepo(mockLike)

	resp, err := svc.ListPosts(1, 5, 20, 0)
	require.NoError(t, err)
	require.Len(t, resp.Posts, 2)
	assert.Equal(t, int64(4), resp.Posts[0].LikeCount)
	assert.Equal(t, int64(2), resp.Posts[0].CommentCount)
	assert.True(t, resp.Posts[0].LikedByMe)
	assert.Equal(t, int64(0), resp.Posts[1].LikeCount)
}
