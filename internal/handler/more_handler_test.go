package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func TestUserHandler_UpdateCurrentUser_Success2(t *testing.T) {
	svc := &testutil.MockUserService{
		UpdateFn: func(userID uint, req *service.UpdateUserRequest) (*model.User, error) {
			return &model.User{ID: userID, Nickname: req.Nickname}, nil
		},
	}
	router := newUserTestRouter(handler.NewUserHandler(svc))

	body, _ := json.Marshal(map[string]string{"nickname": "NewNick"})
	req, _ := http.NewRequest("PATCH", "/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_UpdateCurrentUser_NotFound(t *testing.T) {
	svc := &testutil.MockUserService{
		UpdateFn: func(userID uint, req *service.UpdateUserRequest) (*model.User, error) {
			return nil, service.ErrUserNotFound
		},
	}
	router := newUserTestRouter(handler.NewUserHandler(svc))

	body, _ := json.Marshal(map[string]string{"nickname": "x"})
	req, _ := http.NewRequest("PATCH", "/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserHandler_UpdateCurrentUser_BadBody(t *testing.T) {
	svc := &testutil.MockUserService{}
	router := newUserTestRouter(handler.NewUserHandler(svc))

	req, _ := http.NewRequest("PATCH", "/users/me", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Logout_Success2(t *testing.T) {
	svc := &testutil.MockUserService{
		RevokeRefreshTokenFn: func(token string) error { return nil },
	}
	router := newUserTestRouter(handler.NewUserHandler(svc))

	body, _ := json.Marshal(map[string]string{"refresh_token": "tok"})
	req, _ := http.NewRequest("POST", "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGuildHandler_DeleteGuild_InternalError(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.DeleteGuildFn = func(guildID, userID uint) error {
		return assert.AnError
	}
	router := newGuildTestRouter(h)

	req, _ := http.NewRequest("DELETE", "/guilds/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGuildHandler_UpdateGuild_InternalError(t *testing.T) {
	h, mockGuild, _, _ := newGuildHandler()
	mockGuild.UpdateGuildFn = func(guildID, userID uint, req *service.UpdateGuildRequest) (*model.Guild, error) {
		return nil, assert.AnError
	}
	router := newGuildTestRouter(h)

	body, _ := json.Marshal(map[string]string{"name": "xx"})
	req, _ := http.NewRequest("PATCH", "/guilds/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGuildHandler_LeaveGuild_NotMember(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		LeaveGuildFn: func(guildID, userID uint) error { return service.ErrNotGuildMember },
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, mockMember, &testutil.MockGuildInviteService{})
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequest("POST", "/guilds/1/leave", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMessageHandler_CreateMessage_EmptyContent(t *testing.T) {
	svc := &testutil.MockMessageService{
		CreateMessageFn: func(userID uint, req *service.CreateMessageRequest) (*model.Message, error) {
			return nil, service.ErrEmptyMessageContent
		},
	}
	r := newExtraMessageRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"content": "", "type": "text"})
	req, _ := http.NewRequest("POST", "/channels/1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMessageHandler_DeleteMessage_NotFound(t *testing.T) {
	svc := &testutil.MockMessageService{
		DeleteMessageFn: func(messageID, userID uint) error {
			return service.ErrMessageNotFound
		},
	}
	r := newExtraMessageRouter(svc)
	req, _ := http.NewRequest("DELETE", "/messages/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
