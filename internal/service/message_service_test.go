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

	svc := service.NewMessageService(mockMsg, mockCh, mockMember)

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
	)

	_, err := svc.CreateMessage(1, &service.CreateMessageRequest{ChannelID: 1, Content: ""})
	assert.ErrorIs(t, err, service.ErrEmptyMessageContent)
}

func TestMessageService_CreateMessage_InvalidType(t *testing.T) {
	svc := service.NewMessageService(
		&testutil.MockMessageRepository{},
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
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
	svc := service.NewMessageService(&testutil.MockMessageRepository{}, mockCh, mockMember)

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

	svc := service.NewMessageService(mockMsg, mockCh, mockMember)
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
	svc := service.NewMessageService(mockMsg, mockCh, mockMember)

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
	svc := service.NewMessageService(mockMsg, mockCh, mockMember)

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
	svc := service.NewMessageService(mockMsg, mockCh, mockMember)

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
	svc := service.NewMessageService(mockMsg, mockCh, mockMember)

	err := svc.DeleteMessage(1, 5)
	assert.ErrorIs(t, err, service.ErrNotMessageOwner)
}

// ---------------------------------------------------------------------------
// mockWSManager helper
// ---------------------------------------------------------------------------

type mockWSManager struct {
	broadcastFn func(channelID uint, msgType string, data any)
}

func (m *mockWSManager) BroadcastToChannel(channelID uint, msgType string, data any) {
	if m.broadcastFn != nil {
		m.broadcastFn(channelID, msgType, data)
	}
}
