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

// ---------------------------------------------------------------------------
// GuildService
// ---------------------------------------------------------------------------

func TestGuildService_CreateGuild_Success(t *testing.T) {
	mockGuild := &testutil.MockGuildRepository{
		CreateFn: func(g *model.Guild) error { g.ID = 10; return nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		CreateFn: func(m *model.GuildMember) error { return nil },
	}
	svc := service.NewGuildService(mockGuild, mockMember)

	guild, err := svc.CreateGuild(1, &service.CreateGuildRequest{Name: "My Guild"})

	require.NoError(t, err)
	assert.Equal(t, "My Guild", guild.Name)
	assert.Equal(t, uint(1), guild.OwnerID)
}

func TestGuildService_CreateGuild_RepoError(t *testing.T) {
	mockGuild := &testutil.MockGuildRepository{
		CreateFn: func(g *model.Guild) error { return errors.New("db error") },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	_, err := svc.CreateGuild(1, &service.CreateGuildRequest{Name: "Guild"})
	assert.Error(t, err)
}

func TestGuildService_GetGuild_Success(t *testing.T) {
	guild := &model.Guild{ID: 5, Name: "Test"}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	got, err := svc.GetGuild(5)
	require.NoError(t, err)
	assert.Equal(t, uint(5), got.ID)
}

func TestGuildService_GetGuild_NotFound(t *testing.T) {
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return nil, errors.New("not found") },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	_, err := svc.GetGuild(999)
	assert.ErrorIs(t, err, service.ErrGuildNotFound)
}

func TestGuildService_UpdateGuild_Success(t *testing.T) {
	guild := &model.Guild{ID: 1, Name: "Old Name", OwnerID: 1}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
		UpdateFn:  func(g *model.Guild) error { return nil },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	updated, err := svc.UpdateGuild(1, 1, &service.UpdateGuildRequest{Name: "New Name"})
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestGuildService_UpdateGuild_NotOwner(t *testing.T) {
	guild := &model.Guild{ID: 1, Name: "Guild", OwnerID: 99}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	_, err := svc.UpdateGuild(1, 1, &service.UpdateGuildRequest{Name: "New"})
	assert.ErrorIs(t, err, service.ErrNotGuildOwner)
}

func TestGuildService_DeleteGuild_Success(t *testing.T) {
	guild := &model.Guild{ID: 1, OwnerID: 1}
	deleted := false
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
		DeleteFn:  func(id uint) error { deleted = true; return nil },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	err := svc.DeleteGuild(1, 1)
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestGuildService_DeleteGuild_NotOwner(t *testing.T) {
	guild := &model.Guild{ID: 1, OwnerID: 99}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	err := svc.DeleteGuild(1, 1)
	assert.ErrorIs(t, err, service.ErrNotGuildOwner)
}

func TestGuildService_IsGuildMember_True(t *testing.T) {
	member := &model.GuildMember{ID: 1, GuildID: 1, UserID: 2}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}
	svc := service.NewGuildService(&testutil.MockGuildRepository{}, mockMember)

	ok, err := svc.IsGuildMember(1, 2)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestGuildService_IsGuildMember_False(t *testing.T) {
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return nil, errors.New("not found") },
	}
	svc := service.NewGuildService(&testutil.MockGuildRepository{}, mockMember)

	ok, err := svc.IsGuildMember(1, 999)
	require.NoError(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// GuildMemberService
// ---------------------------------------------------------------------------

func TestGuildMemberService_JoinGuild_Success(t *testing.T) {
	guild := &model.Guild{ID: 1}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return nil, errors.New("not found") },
		CreateFn:    func(m *model.GuildMember) error { return nil },
	}
	svc := service.NewGuildMemberService(mockGuild, mockMember)

	err := svc.JoinGuild(1, 5)
	require.NoError(t, err)
}

func TestGuildMemberService_JoinGuild_AlreadyMember(t *testing.T) {
	guild := &model.Guild{ID: 1}
	existing := &model.GuildMember{ID: 1, GuildID: 1, UserID: 5}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return existing, nil },
	}
	svc := service.NewGuildMemberService(mockGuild, mockMember)

	err := svc.JoinGuild(1, 5)
	assert.ErrorIs(t, err, service.ErrAlreadyInGuild)
}

func TestGuildMemberService_LeaveGuild_Success(t *testing.T) {
	guild := &model.Guild{ID: 1, OwnerID: 99}
	member := &model.GuildMember{ID: 10, GuildID: 1, UserID: 5}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	deleted := false
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
		DeleteFn:    func(id uint) error { deleted = true; return nil },
	}
	svc := service.NewGuildMemberService(mockGuild, mockMember)

	err := svc.LeaveGuild(1, 5)
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestGuildMemberService_LeaveGuild_OwnerCannotLeave(t *testing.T) {
	guild := &model.Guild{ID: 1, OwnerID: 5}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	svc := service.NewGuildMemberService(mockGuild, &testutil.MockGuildMemberRepository{})

	err := svc.LeaveGuild(1, 5)
	assert.ErrorIs(t, err, service.ErrCannotLeaveAsOwner)
}

func TestGuildMemberService_KickMember_Success(t *testing.T) {
	operator := &model.GuildMember{ID: 1, UserID: 1, Role: "admin"}
	target := &model.GuildMember{ID: 2, UserID: 2, Role: "member"}
	deleted := false

	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			if userID == 1 {
				return operator, nil
			}
			return target, nil
		},
		DeleteFn: func(id uint) error { deleted = true; return nil },
	}
	svc := service.NewGuildMemberService(&testutil.MockGuildRepository{}, mockMember)

	err := svc.KickMember(1, 2, 1) // guildID=1, target=2, operator=1
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestGuildMemberService_KickMember_InsufficientPermission(t *testing.T) {
	operator := &model.GuildMember{ID: 1, UserID: 1, Role: "member"}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return operator, nil },
	}
	svc := service.NewGuildMemberService(&testutil.MockGuildRepository{}, mockMember)

	err := svc.KickMember(1, 2, 1)
	assert.ErrorIs(t, err, service.ErrInsufficientPermission)
}

func TestGuildMemberService_UpdateMemberRole_Success(t *testing.T) {
	guild := &model.Guild{ID: 1, OwnerID: 1}
	operator := &model.GuildMember{ID: 1, UserID: 1, Role: "owner"}
	target := &model.GuildMember{ID: 2, UserID: 2, Role: "member"}
	updated := false

	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			if userID == 1 {
				return operator, nil
			}
			return target, nil
		},
		UpdateFn: func(m *model.GuildMember) error { updated = true; return nil },
	}
	svc := service.NewGuildMemberService(mockGuild, mockMember)

	err := svc.UpdateMemberRole(1, 2, 1, "moderator")
	require.NoError(t, err)
	assert.True(t, updated)
}

func TestGuildMemberService_ListGuildMembers_Success(t *testing.T) {
	guild := &model.Guild{ID: 1}
	members := []*model.GuildMember{{ID: 1}, {ID: 2}}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetByGuildIDFn: func(guildID uint) ([]*model.GuildMember, error) { return members, nil },
	}
	svc := service.NewGuildMemberService(mockGuild, mockMember)

	result, err := svc.ListGuildMembers(1)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGuildMemberService_ListGuildMembers_GuildNotFound(t *testing.T) {
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return nil, errors.New("not found") },
	}
	svc := service.NewGuildMemberService(mockGuild, &testutil.MockGuildMemberRepository{})

	_, err := svc.ListGuildMembers(999)
	assert.ErrorIs(t, err, service.ErrGuildNotFound)
}

// ---------------------------------------------------------------------------
// hasMinRole (internal via exported HasMinRole if present, else test indirectly)
// ---------------------------------------------------------------------------

func TestGuildService_ListUserGuilds(t *testing.T) {
	guilds := []*model.Guild{{ID: 1}, {ID: 2}}
	mockGuild := &testutil.MockGuildRepository{
		GetMemberGuildsFn: func(userID uint, offset, limit int) ([]*model.Guild, error) { return guilds, nil },
	}
	svc := service.NewGuildService(mockGuild, &testutil.MockGuildMemberRepository{})

	result, err := svc.ListUserGuilds(1)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

// GuildInviteService tests
func TestGuildInviteService_CreateInvite_Success(t *testing.T) {
	guild := &model.Guild{ID: 1}
	member := &model.GuildMember{ID: 1, GuildID: 1, UserID: 1, Role: "member"}

	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return member, nil },
	}
	mockInvite := &testutil.MockGuildInviteRepository{
		CreateFn: func(invite *model.GuildInvite) error { invite.ID = 1; return nil },
	}

	svc := service.NewGuildInviteService(mockInvite, mockGuild, mockMember)
	invite, err := svc.CreateInvite(1, 1, &service.CreateInviteRequest{MaxUses: 10, ExpiresIn: 3600})

	require.NoError(t, err)
	assert.NotEmpty(t, invite.Code)
	assert.Equal(t, uint(1), invite.GuildID)
}

func TestGuildInviteService_CreateInvite_NotMember(t *testing.T) {
	guild := &model.Guild{ID: 1}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) {
			return nil, errors.New("not found")
		},
	}
	svc := service.NewGuildInviteService(&testutil.MockGuildInviteRepository{}, mockGuild, mockMember)

	_, err := svc.CreateInvite(1, 99, &service.CreateInviteRequest{})
	assert.ErrorIs(t, err, service.ErrNotGuildMember)
}

func TestGuildInviteService_JoinByInvite_Success(t *testing.T) {
	future := time.Now().Add(time.Hour)
	invite := &model.GuildInvite{
		ID:        1,
		GuildID:   1,
		Code:      "TESTCODE",
		MaxUses:   10,
		Uses:      3,
		ExpiresAt: &future,
	}
	guild := &model.Guild{ID: 1}

	mockInvite := &testutil.MockGuildInviteRepository{
		GetByCodeFn:     func(code string) (*model.GuildInvite, error) { return invite, nil },
		IncrementUsesFn: func(id uint) error { return nil },
	}
	mockGuild := &testutil.MockGuildRepository{
		GetByIDFn: func(id uint) (*model.Guild, error) { return guild, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(guildID, userID uint) (*model.GuildMember, error) { return nil, errors.New("not found") },
		CreateFn:    func(m *model.GuildMember) error { return nil },
	}
	svc := service.NewGuildInviteService(mockInvite, mockGuild, mockMember)

	err := svc.JoinByInvite("TESTCODE", 5)
	require.NoError(t, err)
}

func TestGuildInviteService_JoinByInvite_NotFound(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteRepository{
		GetByCodeFn: func(code string) (*model.GuildInvite, error) { return nil, errors.New("not found") },
	}
	svc := service.NewGuildInviteService(mockInvite, &testutil.MockGuildRepository{}, &testutil.MockGuildMemberRepository{})

	err := svc.JoinByInvite("BADCODE", 5)
	assert.ErrorIs(t, err, service.ErrInviteNotFound)
}
