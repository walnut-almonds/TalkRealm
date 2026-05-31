package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func TestUnreadService_AckChannel_AccessDenied(t *testing.T) {
	svc := service.NewUnreadService(
		&testutil.MockChannelReadStateRepository{},
		&testutil.MockChannelRepository{
			GetByIDFn: func(id uint) (*model.Channel, error) {
				return &model.Channel{ID: id, GuildID: testutil.PtrUint(100), Type: "text"}, nil
			},
		},
		&testutil.MockGuildMemberRepository{
			GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
				return nil, errors.New("not member")
			},
		},
		&testutil.MockMessageRepository{},
	)

	err := svc.AckChannel(7, 9, 101)
	assert.ErrorIs(t, err, service.ErrUnreadAccessDenied)
}

func TestUnreadService_AckChannel_InvalidTargetMessage(t *testing.T) {
	upsertCalled := false
	svc := service.NewUnreadService(
		&testutil.MockChannelReadStateRepository{
			UpsertFn: func(userID, channelID, lastMessageID uint) error {
				upsertCalled = true
				return nil
			},
		},
		&testutil.MockChannelRepository{
			GetByIDFn: func(id uint) (*model.Channel, error) {
				return &model.Channel{ID: id, GuildID: testutil.PtrUint(100), Type: "text"}, nil
			},
		},
		&testutil.MockGuildMemberRepository{
			GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
				return &model.GuildMember{GuildID: guildID, UserID: userID}, nil
			},
		},
		&testutil.MockMessageRepository{
			GetByIDFn: func(id uint) (*model.Message, error) {
				return &model.Message{ID: id, ChannelID: 999}, nil
			},
		},
	)

	err := svc.AckChannel(7, 9, 101)
	require.ErrorIs(t, err, service.ErrUnreadInvalidAckTarget)
	assert.False(t, upsertCalled)
}

func TestUnreadService_AckChannel_Success(t *testing.T) {
	called := false
	svc := service.NewUnreadService(
		&testutil.MockChannelReadStateRepository{
			UpsertFn: func(userID, channelID, lastMessageID uint) error {
				called = true

				assert.Equal(t, uint(7), userID)
				assert.Equal(t, uint(9), channelID)
				assert.Equal(t, uint(101), lastMessageID)

				return nil
			},
		},
		&testutil.MockChannelRepository{
			GetByIDFn: func(id uint) (*model.Channel, error) {
				return &model.Channel{ID: id, GuildID: testutil.PtrUint(100), Type: "text"}, nil
			},
		},
		&testutil.MockGuildMemberRepository{
			GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
				return &model.GuildMember{GuildID: guildID, UserID: userID}, nil
			},
		},
		&testutil.MockMessageRepository{
			GetByIDFn: func(id uint) (*model.Message, error) {
				return &model.Message{ID: id, ChannelID: 9}, nil
			},
		},
	)

	err := svc.AckChannel(7, 9, 101)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestUnreadService_GetChannelUnread_AccessDenied(t *testing.T) {
	svc := service.NewUnreadService(
		&testutil.MockChannelReadStateRepository{},
		&testutil.MockChannelRepository{
			GetByIDFn: func(id uint) (*model.Channel, error) {
				return &model.Channel{ID: id, Type: "dm"}, nil
			},
			IsDMParticipantFn: func(channelID, userID uint) (bool, error) {
				return false, nil
			},
		},
		&testutil.MockGuildMemberRepository{},
		&testutil.MockMessageRepository{},
	)

	_, err := svc.GetChannelUnread(7, 9)
	assert.ErrorIs(t, err, service.ErrUnreadAccessDenied)
}

func TestUnreadService_GetChannelUnread_Success(t *testing.T) {
	expected := &repository.ChannelUnreadCount{ChannelID: 9, UnreadCount: 3, MentionCount: 1}
	svc := service.NewUnreadService(
		&testutil.MockChannelReadStateRepository{
			GetChannelUnreadFn: func(userID, channelID uint) (*repository.ChannelUnreadCount, error) {
				return expected, nil
			},
		},
		&testutil.MockChannelRepository{
			GetByIDFn: func(id uint) (*model.Channel, error) {
				return &model.Channel{ID: id, GuildID: testutil.PtrUint(100), Type: "text"}, nil
			},
		},
		&testutil.MockGuildMemberRepository{
			GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
				return &model.GuildMember{GuildID: guildID, UserID: userID}, nil
			},
		},
		&testutil.MockMessageRepository{},
	)

	got, err := svc.GetChannelUnread(7, 9)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUnreadService_GetAllUnread_ForwardRepoError(t *testing.T) {
	expectedErr := errors.New("db error")
	svc := service.NewUnreadService(
		&testutil.MockChannelReadStateRepository{
			GetAllUnreadFn: func(userID uint) ([]*repository.ChannelUnreadCount, error) {
				return nil, expectedErr
			},
		},
		&testutil.MockChannelRepository{},
		&testutil.MockGuildMemberRepository{},
		&testutil.MockMessageRepository{},
	)

	_, err := svc.GetAllUnread(7)
	assert.ErrorIs(t, err, expectedErr)
}
