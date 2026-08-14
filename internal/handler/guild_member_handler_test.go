package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func newGuildMemberRouter(h *handler.GuildHandler) *gin.Engine {
	r := gin.New()
	auth := authMiddleware(1)
	r.POST("/guilds/:id/join", auth, h.JoinGuild)
	r.POST("/guilds/:id/leave", auth, h.LeaveGuild)
	r.DELETE("/guilds/:id/members/:userId", auth, h.KickMember)
	r.PUT("/guilds/:id/members/:userId/role", auth, h.UpdateMemberRole)
	r.POST("/guilds/:id/invites", auth, h.CreateInvite)
	r.GET("/invites/:code", h.GetInvite)
	r.POST("/guilds/join-by-invite", auth, h.JoinByInvite)

	return r
}

func TestGuildHandler_JoinGuild_Success(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		JoinGuildFn: func(guildID, userID uint) error { return nil },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/guilds/1/join",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_JoinGuild_AlreadyInGuild(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		JoinGuildFn: func(guildID, userID uint) error { return service.ErrAlreadyInGuild },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/guilds/1/join",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGuildHandler_LeaveGuild_Success(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		LeaveGuildFn: func(guildID, userID uint) error { return nil },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/guilds/1/leave",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_LeaveGuild_OwnerCannotLeave(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		LeaveGuildFn: func(guildID, userID uint) error { return service.ErrCannotLeaveAsOwner },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/guilds/1/leave",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGuildHandler_KickMember_Success(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		KickMemberFn: func(guildID, targetUserID, operatorUserID uint) error { return nil },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/guilds/1/members/2",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_KickMember_Forbidden(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		KickMemberFn: func(guildID, targetUserID, operatorUserID uint) error {
			return service.ErrNotGuildOwner
		},
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/guilds/1/members/2",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGuildHandler_UpdateMemberRole_Success(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		UpdateMemberRoleFn: func(guildID, targetUserID, operatorUserID uint, role string) error { return nil },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		mockMember,
		&testutil.MockGuildInviteService{},
	)
	r := newGuildMemberRouter(h)

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		"/guilds/1/members/2/role",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_CreateInvite_Success(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		CreateInviteFn: func(guildID, userID uint, req *service.CreateInviteRequest) (*model.GuildInvite, error) {
			return &model.GuildInvite{Code: "ABC123"}, nil
		},
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		&testutil.MockGuildMemberService{},
		mockInvite,
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/guilds/1/invites",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestGuildHandler_GetInvite_Success(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		GetInviteByCodeFn: func(code string) (*model.GuildInvite, error) {
			return &model.GuildInvite{Code: code}, nil
		},
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		&testutil.MockGuildMemberService{},
		mockInvite,
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/invites/ABC123",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_GetInvite_NotFound(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		GetInviteByCodeFn: func(code string) (*model.GuildInvite, error) {
			return nil, service.ErrInviteNotFound
		},
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		&testutil.MockGuildMemberService{},
		mockInvite,
	)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/invites/NOTEXIST",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGuildHandler_JoinByInvite_Success(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		JoinByInviteFn: func(code string, userID uint) error { return nil },
	}
	h := handler.NewGuildHandler(
		&testutil.MockGuildService{},
		&testutil.MockGuildMemberService{},
		mockInvite,
	)
	r := newGuildMemberRouter(h)

	body, _ := json.Marshal(map[string]string{"code": "ABC123"})
	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/guilds/join-by-invite",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// handler.go stub endpoints
// ---------------------------------------------------------------------------

func TestHandler_HealthCheck(t *testing.T) {
	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Ping(t *testing.T) {
	r := gin.New()
	r.GET("/ping", handler.Ping)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
