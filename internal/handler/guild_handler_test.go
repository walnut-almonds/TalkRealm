package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	_ = logger.Init("error") // suppress logs during tests
	os.Exit(m.Run())
}

func authMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func newGuildTestRouter(h *handler.GuildHandler) *gin.Engine {
	r := gin.New()
	auth := authMiddleware(uint(1))
	r.POST("/guilds", auth, h.CreateGuild)
	r.GET("/guilds/:id", auth, h.GetGuild)
	r.GET("/guilds", auth, h.ListUserGuilds)
	r.PATCH("/guilds/:id", auth, h.UpdateGuild)
	r.DELETE("/guilds/:id", auth, h.DeleteGuild)
	r.GET("/guilds/:id/members", auth, h.ListGuildMembers)
	return r
}

func newGuildHandler() (*handler.GuildHandler, *testutil.MockGuildService, *testutil.MockGuildMemberService, *testutil.MockGuildInviteService) {
	mockGuild := &testutil.MockGuildService{}
	mockMember := &testutil.MockGuildMemberService{}
	mockInvite := &testutil.MockGuildInviteService{}
	h := handler.NewGuildHandler(mockGuild, mockMember, mockInvite)
	return h, mockGuild, mockMember, mockInvite
}

// ---------------------------------------------------------------------------
// CreateGuild
// ---------------------------------------------------------------------------

func TestGuildHandler_CreateGuild_Success(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.CreateGuildFn = func(ownerID uint, req *service.CreateGuildRequest) (*model.Guild, error) {
		return &model.Guild{ID: 1, Name: req.Name, OwnerID: ownerID}, nil
	}
	router := newGuildTestRouter(h)

	body, _ := json.Marshal(map[string]string{"name": "My Guild"})
	req := httptest.NewRequest(http.MethodPost, "/guilds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var guild model.Guild
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &guild))
	assert.Equal(t, "My Guild", guild.Name)
}

func TestGuildHandler_CreateGuild_BadRequest(t *testing.T) {
	h, _, _, _ := newGuildHandler()
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/guilds", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetGuild
// ---------------------------------------------------------------------------

func TestGuildHandler_GetGuild_Success(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.GetGuildFn = func(guildID uint) (*model.Guild, error) {
		return &model.Guild{ID: guildID, Name: "Test"}, nil
	}
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/guilds/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_GetGuild_NotFound(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.GetGuildFn = func(guildID uint) (*model.Guild, error) {
		return nil, service.ErrGuildNotFound
	}
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/guilds/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// ListGuilds
// ---------------------------------------------------------------------------

func TestGuildHandler_ListGuilds_Success(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.ListUserGuildsFn = func(userID uint) ([]*model.Guild, error) {
		return []*model.Guild{{ID: 1}, {ID: 2}}, nil
	}
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/guilds", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateGuild
// ---------------------------------------------------------------------------

func TestGuildHandler_UpdateGuild_Success(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.UpdateGuildFn = func(guildID, userID uint, req *service.UpdateGuildRequest) (*model.Guild, error) {
		return &model.Guild{ID: guildID, Name: req.Name}, nil
	}
	router := newGuildTestRouter(h)

	body, _ := json.Marshal(map[string]string{"name": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/guilds/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_UpdateGuild_NotOwner(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.UpdateGuildFn = func(guildID, userID uint, req *service.UpdateGuildRequest) (*model.Guild, error) {
		return nil, service.ErrNotGuildOwner
	}
	router := newGuildTestRouter(h)

	body, _ := json.Marshal(map[string]string{"name": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/guilds/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteGuild
// ---------------------------------------------------------------------------

func TestGuildHandler_DeleteGuild_Success(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.DeleteGuildFn = func(guildID, userID uint) error { return nil }
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/guilds/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_DeleteGuild_NotOwner(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.DeleteGuildFn = func(guildID, userID uint) error {
		return service.ErrNotGuildOwner
	}
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/guilds/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// ListGuildMembers
// ---------------------------------------------------------------------------

func TestGuildHandler_ListGuildMembers_Success(t *testing.T) {
	h, _, mockMember, _ := newGuildHandler()
	mockMember.ListGuildMembersFn = func(guildID uint) ([]*model.GuildMember, error) {
		return []*model.GuildMember{{ID: 1, UserID: 1, Role: "owner"}}, nil
	}
	router := newGuildTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/guilds/1/members", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
