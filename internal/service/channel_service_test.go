package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func newChannelService(
	channelRepo *testutil.MockChannelRepository,
	guildRepo *testutil.MockGuildRepository,
	memberRepo *testutil.MockGuildMemberRepository,
) service.ChannelService {
	return service.NewChannelService(channelRepo, guildRepo, memberRepo)
}

func TestChannelService_CreateChannel_Success_Owner(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		CreateFn:       func(ch *model.Channel) error { return nil },
		GetByGuildIDFn: func(guildID uint) ([]*model.Channel, error) { return []*model.Channel{}, nil },
	}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 1}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	req := &service.CreateChannelRequest{GuildID: 1, Name: "general", Type: "text"}
	ch, err := svc.CreateChannel(1, req)
	require.NoError(t, err)
	assert.Equal(t, "general", ch.Name)
}

func TestChannelService_CreateChannel_AllowsFeedType(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		CreateFn:       func(ch *model.Channel) error { return nil },
		GetByGuildIDFn: func(guildID uint) ([]*model.Channel, error) { return []*model.Channel{}, nil },
	}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 1}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	_, err := svc.CreateChannel(1, &service.CreateChannelRequest{
		GuildID: 10,
		Name:    "動態牆",
		Type:    "feed",
	})
	require.NoError(t, err)
}

func TestChannelService_CreateChannel_GuildNotFound(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return nil, assert.AnError
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	req := &service.CreateChannelRequest{GuildID: 99, Name: "x", Type: "text"}
	_, err := svc.CreateChannel(1, req)
	assert.Error(t, err)
}

func TestChannelService_CreateChannel_NotMember(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 2}, nil // owner is user 2
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return nil, assert.AnError
		},
	}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	req := &service.CreateChannelRequest{GuildID: 1, Name: "x", Type: "text"}
	_, err := svc.CreateChannel(1, req) // user 1 is not owner (2), not member
	assert.Error(t, err)
}

func TestChannelService_CreateChannel_NotAdmin(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 2}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return &model.GuildMember{Role: "member"}, nil
		},
	}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	req := &service.CreateChannelRequest{GuildID: 1, Name: "x", Type: "text"}
	_, err := svc.CreateChannel(1, req)
	assert.Error(t, err)
}

func TestChannelService_GetChannel_Success(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{
				ID:      id,
				GuildID: testutil.PtrUint(1),
				Name:    "general",
				Type:    "text",
			}, nil
		},
	}
	guildRepo := &testutil.MockGuildRepository{}
	memberRepo := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return &model.GuildMember{Role: "member"}, nil
		},
	}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	ch, err := svc.GetChannel(1, 1)
	require.NoError(t, err)
	assert.Equal(t, "general", ch.Name)
}

func TestChannelService_GetChannel_NotFound(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return nil, assert.AnError
		},
	}
	guildRepo := &testutil.MockGuildRepository{}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	_, err := svc.GetChannel(99, 1)
	assert.Error(t, err)
}

func TestChannelService_GetChannel_NotMember(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: id, GuildID: testutil.PtrUint(1)}, nil
		},
	}
	guildRepo := &testutil.MockGuildRepository{}
	memberRepo := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return nil, assert.AnError
		},
	}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	_, err := svc.GetChannel(1, 99)
	assert.Error(t, err)
}

func TestChannelService_ListGuildChannels_Success(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByGuildIDFn: func(guildID uint) ([]*model.Channel, error) {
			return []*model.Channel{{ID: 1, Name: "general"}}, nil
		},
	}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return &model.GuildMember{}, nil
		},
	}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	channels, err := svc.ListGuildChannels(1, 1)
	require.NoError(t, err)
	assert.Len(t, channels, 1)
}

func TestChannelService_ListGuildChannels_GuildNotFound(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return nil, assert.AnError },
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	_, err := svc.ListGuildChannels(99, 1)
	assert.Error(t, err)
}

func TestChannelService_UpdateChannel_Success_Owner(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{
				ID:      id,
				GuildID: testutil.PtrUint(1),
				Name:    "old",
				Type:    "text",
			}, nil
		},
		UpdateFn: func(ch *model.Channel) error { return nil },
	}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 1}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	req := &service.UpdateChannelRequest{Name: "new-name"}
	ch, err := svc.UpdateChannel(1, 1, req)
	require.NoError(t, err)
	assert.Equal(t, "new-name", ch.Name)
}

func TestChannelService_UpdateChannel_ChannelNotFound(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return nil, assert.AnError },
	}
	guildRepo := &testutil.MockGuildRepository{}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	_, err := svc.UpdateChannel(99, 1, &service.UpdateChannelRequest{})
	assert.Error(t, err)
}

func TestChannelService_DeleteChannel_Success(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: id, GuildID: testutil.PtrUint(1)}, nil
		},
		DeleteFn: func(id uint) error { return nil },
	}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 1}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	err := svc.DeleteChannel(1, 1)
	require.NoError(t, err)
}

func TestChannelService_DeleteChannel_NotFound(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return nil, assert.AnError },
	}
	guildRepo := &testutil.MockGuildRepository{}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	err := svc.DeleteChannel(99, 1)
	assert.Error(t, err)
}

func TestChannelService_UpdateChannelPosition_Success(t *testing.T) {
	channelRepo := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: id, GuildID: testutil.PtrUint(1)}, nil
		},
		UpdateFn: func(ch *model.Channel) error { return nil },
	}
	guildRepo := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) {
			return &model.Guild{ID: id, OwnerID: 1}, nil
		},
	}
	memberRepo := &testutil.MockGuildMemberRepository{}
	svc := newChannelService(channelRepo, guildRepo, memberRepo)

	err := svc.UpdateChannelPosition(1, 1, 3)
	require.NoError(t, err)
}
