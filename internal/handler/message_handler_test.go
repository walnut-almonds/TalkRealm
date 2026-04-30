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
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func newMessageTestRouter(h *handler.MessageHandler) *gin.Engine {
	r := gin.New()
	auth := authMiddleware(uint(5))
	r.POST("/channels/:id/messages", auth, h.CreateMessage)
	r.GET("/channels/:id/messages", auth, h.ListChannelMessages)
	r.GET("/messages/:id", auth, h.GetMessage)
	r.PATCH("/messages/:id", auth, h.UpdateMessage)
	r.DELETE("/messages/:id", auth, h.DeleteMessage)
	return r
}

// ---------------------------------------------------------------------------
// CreateMessage
// ---------------------------------------------------------------------------

func TestMessageHandler_CreateMessage_Success(t *testing.T) {
	msg := &model.Message{ID: 1, Content: "hello", UserID: 5, ChannelID: 1}
	mockSvc := &testutil.MockMessageService{
		CreateMessageFn: func(userID uint, req *service.CreateMessageRequest) (*model.Message, error) {
			return msg, nil
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	body, _ := json.Marshal(map[string]string{"content": "hello"})
	req := httptest.NewRequest(http.MethodPost, "/channels/1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestMessageHandler_CreateMessage_NotMember(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		CreateMessageFn: func(userID uint, req *service.CreateMessageRequest) (*model.Message, error) {
			return nil, service.ErrNotChannelMemberMsg
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	body, _ := json.Marshal(map[string]string{"content": "hello"})
	req := httptest.NewRequest(http.MethodPost, "/channels/1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// ListChannelMessages
// ---------------------------------------------------------------------------

func TestMessageHandler_ListChannelMessages_Success(t *testing.T) {
	msgs := []*model.Message{{ID: 3}, {ID: 2}, {ID: 1}}
	mockSvc := &testutil.MockMessageService{
		ListChannelMessagesFn: func(channelID, userID uint, limit int, before uint) (*service.MessageListResponse, error) {
			return &service.MessageListResponse{Messages: msgs, HasMore: false}, nil
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/channels/1/messages?limit=50", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp service.MessageListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Messages, 3)
}

func TestMessageHandler_ListChannelMessages_WithCursor(t *testing.T) {
	var gotBefore uint
	mockSvc := &testutil.MockMessageService{
		ListChannelMessagesFn: func(channelID, userID uint, limit int, before uint) (*service.MessageListResponse, error) {
			gotBefore = before
			return &service.MessageListResponse{Messages: nil, HasMore: false}, nil
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/channels/1/messages?before=10&limit=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uint(10), gotBefore)
}

// ---------------------------------------------------------------------------
// GetMessage
// ---------------------------------------------------------------------------

func TestMessageHandler_GetMessage_Success(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		GetMessageFn: func(messageID, userID uint) (*model.Message, error) {
			return &model.Message{ID: messageID}, nil
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/messages/5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMessageHandler_GetMessage_NotFound(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		GetMessageFn: func(messageID, userID uint) (*model.Message, error) {
			return nil, service.ErrMessageNotFound
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/messages/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateMessage
// ---------------------------------------------------------------------------

func TestMessageHandler_UpdateMessage_Success(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		UpdateMessageFn: func(messageID, userID uint, req *service.UpdateMessageRequest) (*model.Message, error) {
			return &model.Message{ID: messageID, Content: req.Content}, nil
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	body, _ := json.Marshal(map[string]string{"content": "updated"})
	req := httptest.NewRequest(http.MethodPatch, "/messages/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMessageHandler_UpdateMessage_NotOwner(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		UpdateMessageFn: func(messageID, userID uint, req *service.UpdateMessageRequest) (*model.Message, error) {
			return nil, service.ErrNotMessageOwner
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	body, _ := json.Marshal(map[string]string{"content": "updated"})
	req := httptest.NewRequest(http.MethodPatch, "/messages/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// DeleteMessage
// ---------------------------------------------------------------------------

func TestMessageHandler_DeleteMessage_Success(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		DeleteMessageFn: func(messageID, userID uint) error { return nil },
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/messages/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMessageHandler_DeleteMessage_NotOwner(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		DeleteMessageFn: func(messageID, userID uint) error {
			return service.ErrNotMessageOwner
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/messages/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMessageHandler_DeleteMessage_ServiceError(t *testing.T) {
	mockSvc := &testutil.MockMessageService{
		DeleteMessageFn: func(messageID, userID uint) error {
			return errors.New("unexpected error")
		},
	}
	h := handler.NewMessageHandler(mockSvc)
	router := newMessageTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/messages/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
