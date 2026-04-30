package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func TestChannelHandler_UpdateChannelPosition_NotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelPositionFn: func(channelID, userID uint, position int) error {
			return service.ErrChannelNotFound
		},
	}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"position": 2})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1/position", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChannelHandler_UpdateChannelPosition_Forbidden(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelPositionFn: func(channelID, userID uint, position int) error {
			return errors.New("only owner or admin can update channel position")
		},
	}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"position": 3})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1/position", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_UpdateChannel_GuildNotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelFn: func(channelID, userID uint, req *service.UpdateChannelRequest) (*model.Channel, error) {
			return nil, service.ErrGuildNotFound
		},
	}
	r := newChannelTestRouter(svc)
	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGuildHandler_KickMember_NotMember(t *testing.T) {
	mockMember := &testutil.MockGuildMemberService{
		KickMemberFn: func(guildID, targetUserID, operatorUserID uint) error {
			return service.ErrNotGuildMember
		},
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, mockMember, &testutil.MockGuildInviteService{})
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequest("DELETE", "/guilds/1/members/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGuildHandler_GetInvite_Expired(t *testing.T) {
	mockInvite := &testutil.MockGuildInviteService{
		GetInviteByCodeFn: func(code string) (*model.GuildInvite, error) {
			return nil, service.ErrInviteExpired
		},
	}
	h := handler.NewGuildHandler(&testutil.MockGuildService{}, &testutil.MockGuildMemberService{}, mockInvite)
	r := newGuildMemberRouter(h)

	req, _ := http.NewRequest("GET", "/invites/EXPIRED", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusGone, w.Code)
}

