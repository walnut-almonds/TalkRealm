package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
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

// ---------------------------------------------------------------------------
// Channel handler additional tests
// ---------------------------------------------------------------------------

func TestChannelHandler_CreateChannel_NotMember(t *testing.T) {
	svc := &testutil.MockChannelService{
		CreateChannelFn: func(userID uint, req *service.CreateChannelRequest) (*model.Channel, error) {
			return nil, service.ErrNotGuildMemberCh
		},
	}
	r := newChannelTestRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"name": "x", "type": "text"})
	req, _ := http.NewRequest("POST", "/api/v1/guilds/1/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_GetChannel_NotMember(t *testing.T) {
	svc := &testutil.MockChannelService{
		GetChannelFn: func(channelID, userID uint) (*model.Channel, error) {
			return nil, service.ErrNotGuildMemberCh
		},
	}
	r := newChannelTestRouter(svc)
	req, _ := http.NewRequest("GET", "/api/v1/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_UpdateChannel_NotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelFn: func(channelID, userID uint, req *service.UpdateChannelRequest) (*model.Channel, error) {
			return nil, service.ErrChannelNotFound
		},
	}
	r := newChannelTestRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChannelHandler_UpdateChannel_Forbidden(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelFn: func(channelID, userID uint, req *service.UpdateChannelRequest) (*model.Channel, error) {
			return nil, errors.New("only owner or admin can update channels")
		},
	}
	r := newChannelTestRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_ListGuildChannels_NotMember(t *testing.T) {
	svc := &testutil.MockChannelService{
		ListGuildChannelsFn: func(guildID, userID uint) ([]*model.Channel, error) {
			return nil, service.ErrNotGuildMemberCh
		},
	}
	r := newChannelTestRouter(svc)
	req, _ := http.NewRequest("GET", "/api/v1/guilds/1/channels", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_ListGuildChannels_GuildNotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		ListGuildChannelsFn: func(guildID, userID uint) ([]*model.Channel, error) {
			return nil, service.ErrGuildNotFound
		},
	}
	r := newChannelTestRouter(svc)
	req, _ := http.NewRequest("GET", "/api/v1/guilds/1/channels", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChannelHandler_DeleteChannel_Forbidden(t *testing.T) {
	svc := &testutil.MockChannelService{
		DeleteChannelFn: func(channelID, userID uint) error {
			return errors.New("only owner or admin can delete channels")
		},
	}
	r := newChannelTestRouter(svc)
	req, _ := http.NewRequest("DELETE", "/api/v1/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// Guild handler additional tests
// ---------------------------------------------------------------------------

func newFullGuildRouter(h *handler.GuildHandler) *gin.Engine {
	r := gin.New()
	auth := authMiddleware(1)
	r.POST("/guilds", auth, h.CreateGuild)
	r.GET("/guilds/:id", auth, h.GetGuild)
	r.GET("/guilds", auth, h.ListUserGuilds)
	r.PATCH("/guilds/:id", auth, h.UpdateGuild)
	r.DELETE("/guilds/:id", auth, h.DeleteGuild)
	r.GET("/guilds/:id/members", auth, h.ListGuildMembers)
	r.PUT("/guilds/:id/members/:userId/role", auth, h.UpdateMemberRole)
	r.POST("/guilds/:id/invites", auth, h.CreateInvite)
	r.GET("/invites/:code", h.GetInvite)
	r.POST("/guilds/join-by-invite", auth, h.JoinByInvite)
	r.POST("/guilds/:id/join", auth, h.JoinGuild)
	r.POST("/guilds/:id/leave", auth, h.LeaveGuild)
	r.DELETE("/guilds/:id/members/:userId", auth, h.KickMember)
	return r
}

func TestGuildHandler_UpdateMemberRole_Forbidden(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		UpdateMemberRoleFn: func(guildID, targetUserID, operatorUserID uint, role string) error {
			return service.ErrNotGuildOwner
		},
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, mockMember, &testutil.MockGuildInviteService{})
	r := newFullGuildRouter(h)

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req, _ := http.NewRequest("PUT", "/guilds/1/members/2/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGuildHandler_UpdateMemberRole_BadBody(t *testing.T) {
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, &testutil.MockGuildMemberService{}, &testutil.MockGuildInviteService{})
	r := newFullGuildRouter(h)

	req, _ := http.NewRequest("PUT", "/guilds/1/members/2/role", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGuildHandler_CreateInvite_NotMember(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		CreateInviteFn: func(guildID, userID uint, req *service.CreateInviteRequest) (*model.GuildInvite, error) {
			return nil, service.ErrNotGuildMember
		},
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, &testutil.MockGuildMemberService{}, mockInvite)
	r := newFullGuildRouter(h)

	req, _ := http.NewRequest("POST", "/guilds/1/invites", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGuildHandler_JoinByInvite_NotFound(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		JoinByInviteFn: func(code string, userID uint) error { return service.ErrInviteNotFound },
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, &testutil.MockGuildMemberService{}, mockInvite)
	r := newFullGuildRouter(h)

	body, _ := json.Marshal(map[string]string{"code": "NOPE"})
	req, _ := http.NewRequest("POST", "/guilds/join-by-invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGuildHandler_JoinByInvite_AlreadyIn(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		JoinByInviteFn: func(code string, userID uint) error { return service.ErrAlreadyInGuild },
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, &testutil.MockGuildMemberService{}, mockInvite)
	r := newFullGuildRouter(h)

	body, _ := json.Marshal(map[string]string{"code": "ABC"})
	req, _ := http.NewRequest("POST", "/guilds/join-by-invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGuildHandler_JoinGuild_GuildNotFound(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		JoinGuildFn: func(guildID, userID uint) error { return service.ErrGuildNotFound },
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, mockMember, &testutil.MockGuildInviteService{})
	r := newFullGuildRouter(h)

	req, _ := http.NewRequest("POST", "/guilds/1/join", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGuildHandler_ListGuildMembers_NotFound(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		ListGuildMembersFn: func(guildID uint) ([]*model.GuildMember, error) {
			return nil, service.ErrGuildNotFound
		},
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, mockMember, &testutil.MockGuildInviteService{})
	r := newFullGuildRouter(h)

	req, _ := http.NewRequest("GET", "/guilds/1/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Message handler additional tests
// ---------------------------------------------------------------------------

func newExtraMessageRouter(svc service.MessageService) *gin.Engine {
	r := gin.New()
	h := handler.NewMessageHandler(svc)
	r.POST("/channels/:id/messages", authMiddleware(1), h.CreateMessage)
	r.GET("/messages/:id", authMiddleware(1), h.GetMessage)
	r.GET("/channels/:id/messages", authMiddleware(1), h.ListChannelMessages)
	r.PUT("/messages/:id", authMiddleware(1), h.UpdateMessage)
	r.DELETE("/messages/:id", authMiddleware(1), h.DeleteMessage)
	return r
}

func TestMessageHandler_CreateMessage_NotMember2(t *testing.T) {
	svc := &testutil.MockMessageService{
		CreateMessageFn: func(userID uint, req *service.CreateMessageRequest) (*model.Message, error) {
			return nil, service.ErrNotChannelMemberMsg
		},
	}
	r := newExtraMessageRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"content": "hi", "type": "text"})
	req, _ := http.NewRequest("POST", "/channels/1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMessageHandler_GetMessage_NotFound2(t *testing.T) {
	svc := &testutil.MockMessageService{
		GetMessageFn: func(messageID, userID uint) (*model.Message, error) {
			return nil, service.ErrMessageNotFound
		},
	}
	r := newExtraMessageRouter(svc)
	req, _ := http.NewRequest("GET", "/messages/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMessageHandler_UpdateMessage_NotOwner2(t *testing.T) {
	svc := &testutil.MockMessageService{
		UpdateMessageFn: func(messageID, userID uint, req *service.UpdateMessageRequest) (*model.Message, error) {
			return nil, service.ErrNotMessageOwner
		},
	}
	r := newExtraMessageRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"content": "edited"})
	req, _ := http.NewRequest("PUT", "/messages/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMessageHandler_DeleteMessage_NotOwner2(t *testing.T) {
	svc := &testutil.MockMessageService{
		DeleteMessageFn: func(messageID, userID uint) error {
			return service.ErrNotMessageOwner
		},
	}
	r := newExtraMessageRouter(svc)
	req, _ := http.NewRequest("DELETE", "/messages/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMessageHandler_ListChannelMessages_NotMember2(t *testing.T) {
	svc := &testutil.MockMessageService{
		ListChannelMessagesFn: func(channelID, userID uint, limit int, before uint) (*service.MessageListResponse, error) {
			return nil, service.ErrNotChannelMemberMsg
		},
	}
	r := newExtraMessageRouter(svc)
	req, _ := http.NewRequest("GET", "/channels/1/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
