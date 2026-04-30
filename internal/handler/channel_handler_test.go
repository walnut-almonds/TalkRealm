package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func newChannelTestRouter(svc service.ChannelService) *gin.Engine {
	r := gin.New()
	h := handler.NewChannelHandler(svc)

	guilds := r.Group("/api/v1/guilds")
	{
		guilds.POST("/:id/channels", authMiddleware(1), h.CreateChannel)
		guilds.GET("/:id/channels", authMiddleware(1), h.ListGuildChannels)
	}

	channels := r.Group("/api/v1/channels")
	{
		channels.GET("/:id", authMiddleware(1), h.GetChannel)
		channels.PUT("/:id", authMiddleware(1), h.UpdateChannel)
		channels.DELETE("/:id", authMiddleware(1), h.DeleteChannel)
		channels.PUT("/:id/position", authMiddleware(1), h.UpdateChannelPosition)
	}

	return r
}

func TestChannelHandler_CreateChannel_Success(t *testing.T) {
	svc := &testutil.MockChannelService{
		CreateChannelFn: func(userID uint, req *service.CreateChannelRequest) (*model.Channel, error) {
			return &model.Channel{ID: 1, Name: req.Name, Type: req.Type}, nil
		},
	}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"name": "general", "type": "text"})
	req, _ := http.NewRequest("POST", "/api/v1/guilds/1/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestChannelHandler_CreateChannel_InvalidID(t *testing.T) {
	svc := &testutil.MockChannelService{}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"name": "x", "type": "text"})
	req, _ := http.NewRequest("POST", "/api/v1/guilds/abc/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChannelHandler_CreateChannel_GuildNotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		CreateChannelFn: func(userID uint, req *service.CreateChannelRequest) (*model.Channel, error) {
			return nil, service.ErrGuildNotFound
		},
	}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"name": "x", "type": "text"})
	req, _ := http.NewRequest("POST", "/api/v1/guilds/1/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChannelHandler_GetChannel_Success(t *testing.T) {
	svc := &testutil.MockChannelService{
		GetChannelFn: func(channelID, userID uint) (*model.Channel, error) {
			return &model.Channel{ID: channelID, Name: "general"}, nil
		},
	}
	r := newChannelTestRouter(svc)

	req, _ := http.NewRequest("GET", "/api/v1/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChannelHandler_GetChannel_NotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		GetChannelFn: func(channelID, userID uint) (*model.Channel, error) {
			return nil, service.ErrChannelNotFound
		},
	}
	r := newChannelTestRouter(svc)

	req, _ := http.NewRequest("GET", "/api/v1/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChannelHandler_ListGuildChannels_Success(t *testing.T) {
	svc := &testutil.MockChannelService{
		ListGuildChannelsFn: func(guildID, userID uint) ([]*model.Channel, error) {
			return []*model.Channel{{ID: 1, Name: "general"}}, nil
		},
	}
	r := newChannelTestRouter(svc)

	req, _ := http.NewRequest("GET", "/api/v1/guilds/1/channels", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChannelHandler_UpdateChannel_Success(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelFn: func(channelID, userID uint, req *service.UpdateChannelRequest) (*model.Channel, error) {
			return &model.Channel{ID: channelID, Name: "updated"}, nil
		},
	}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"name": "updated"})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChannelHandler_DeleteChannel_Success(t *testing.T) {
	svc := &testutil.MockChannelService{
		DeleteChannelFn: func(channelID, userID uint) error { return nil },
	}
	r := newChannelTestRouter(svc)

	req, _ := http.NewRequest("DELETE", "/api/v1/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChannelHandler_DeleteChannel_NotFound(t *testing.T) {
	svc := &testutil.MockChannelService{
		DeleteChannelFn: func(channelID, userID uint) error {
			return service.ErrChannelNotFound
		},
	}
	r := newChannelTestRouter(svc)

	req, _ := http.NewRequest("DELETE", "/api/v1/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChannelHandler_UpdateChannelPosition_Success(t *testing.T) {
	svc := &testutil.MockChannelService{
		UpdateChannelPositionFn: func(channelID, userID uint, position int) error { return nil },
	}
	r := newChannelTestRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{"position": 2})
	req, _ := http.NewRequest("PUT", "/api/v1/channels/1/position", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
