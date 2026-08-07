package websocket //nolint:testpackage // Tests verify private subscription indexes after access revocation.

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
)

type channelAccessStub struct {
	channel       *model.Channel
	channelErr    error
	dmParticipant bool
	dmErr         error
}

func (s channelAccessStub) GetByID(uint) (*model.Channel, error) {
	return s.channel, s.channelErr
}

func (s channelAccessStub) IsDMParticipant(uint, uint) (bool, error) {
	return s.dmParticipant, s.dmErr
}

type guildAccessStub struct {
	ids []uint
	err error
}

func (s guildAccessStub) GetUserGuildIDs(uint) ([]uint, error) {
	return s.ids, s.err
}

func TestManagerCanAccessChannel(t *testing.T) {
	guildID := uint(7)

	tests := []struct {
		name    string
		channel *model.Channel
		guilds  []uint
		dmOK    bool
		want    bool
	}{
		{
			name:    "allows guild member",
			channel: &model.Channel{GuildID: &guildID, Type: "text"},
			guilds:  []uint{guildID},
			want:    true,
		},
		{
			name:    "rejects nonmember guild channel",
			channel: &model.Channel{GuildID: &guildID, Type: "text"},
			guilds:  []uint{8},
			want:    false,
		},
		{
			name:    "allows DM participant",
			channel: &model.Channel{Type: "dm"},
			dmOK:    true,
			want:    true,
		},
		{
			name:    "rejects nonparticipant DM",
			channel: &model.Channel{Type: "dm"},
			dmOK:    false,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewManager(nil)
			manager.SetGuildLookup(guildAccessStub{ids: test.guilds})
			manager.SetChannelAccessLookup(channelAccessStub{
				channel:       test.channel,
				dmParticipant: test.dmOK,
			})

			assert.Equal(t, test.want, manager.CanAccessChannel(42, 9))
		})
	}

	t.Run("rejects when access lookup is unavailable", func(t *testing.T) {
		manager := NewManager(nil)
		assert.False(t, manager.CanAccessChannel(42, 9))
	})

	t.Run("rejects lookup errors", func(t *testing.T) {
		manager := NewManager(nil)
		manager.SetGuildLookup(guildAccessStub{err: errors.New("database unavailable")})
		manager.SetChannelAccessLookup(channelAccessStub{
			channel: &model.Channel{GuildID: &guildID, Type: "text"},
		})

		assert.False(t, manager.CanAccessChannel(42, 9))
	})

	t.Run("does not subscribe an unauthorized client", func(t *testing.T) {
		manager := NewManager(nil)
		manager.SetGuildLookup(guildAccessStub{ids: []uint{8}})
		manager.SetChannelAccessLookup(channelAccessStub{
			channel: &model.Channel{GuildID: &guildID, Type: "text"},
		})
		client := NewClient(nil, manager)
		client.userID = 42

		assert.False(t, manager.SubscribeToChannel(client, 9))
		assert.Empty(t, client.channels)
		assert.Empty(t, manager.channelSubscriptions)
	})

	t.Run("revokes guild subscriptions after membership removal", func(t *testing.T) {
		manager := NewManager(nil)
		manager.SetGuildLookup(guildAccessStub{ids: []uint{guildID}})
		manager.SetChannelAccessLookup(channelAccessStub{
			channel: &model.Channel{GuildID: &guildID, Type: "text"},
		})
		client := NewClient(nil, manager)
		client.userID = 42
		client.identified = true
		manager.clients[client] = true

		assert.True(t, manager.SubscribeToChannel(client, 9))
		manager.SubscribeToGuild(client, guildID)

		manager.RevokeGuildAccess(42, guildID)

		assert.Empty(t, client.channels)
		assert.Empty(t, client.channelGuilds)
		assert.Empty(t, client.guilds)
		assert.Empty(t, manager.channelSubscriptions[9])
		assert.Empty(t, manager.guildSubscriptions[guildID])
	})

	t.Run("drops voice participation in the revoked guild", func(t *testing.T) {
		manager := NewManager(nil)
		client := NewClient(nil, manager)
		client.userID = 42
		client.identified = true
		client.currentVoiceChannelID = 9
		client.currentVoiceGuildID = guildID
		manager.clients[client] = true
		manager.UpsertVoiceParticipant(9, 42, "alice")

		manager.RevokeGuildAccess(42, guildID)

		assert.Empty(t, manager.GetVoiceParticipants(9))
		assert.Zero(t, client.currentVoiceChannelID)
		assert.Zero(t, client.currentVoiceGuildID)
	})
}

func TestManagerIsSubscribedToChannel(t *testing.T) {
	guildID := uint(7)
	manager := NewManager(nil)
	manager.SetGuildLookup(guildAccessStub{ids: []uint{guildID}})
	manager.SetChannelAccessLookup(channelAccessStub{
		channel: &model.Channel{GuildID: &guildID, Type: "text"},
	})

	client := NewClient(nil, manager)
	client.userID = 42

	assert.False(t, manager.IsSubscribedToChannel(client, 9))
	require.True(t, manager.SubscribeToChannel(client, 9))
	assert.True(t, manager.IsSubscribedToChannel(client, 9))

	manager.UnsubscribeFromChannel(client, 9)
	assert.False(t, manager.IsSubscribedToChannel(client, 9))
}
