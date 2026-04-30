package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/testutil"
)

func newUserTestRouter(h *handler.UserHandler) *gin.Engine {
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.GET("/users/me", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	}, h.GetCurrentUser)
	r.PATCH("/users/me", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	}, h.UpdateCurrentUser)
	r.POST("/auth/refresh", h.RefreshToken)
	r.POST("/auth/logout", h.Logout)
	r.GET("/users/:id", h.GetPublicUser)
	return r
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestUserHandler_Register_Success(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		RegisterFn: func(req *service.RegisterRequest) (*model.User, error) {
			return &model.User{ID: 1, Username: req.Username, Email: req.Email}, nil
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "pass1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestUserHandler_Register_InvalidBody(t *testing.T) {
	h := handler.NewUserHandler(&testutil.MockUserService{})
	router := newUserTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Register_UserExists(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		RegisterFn: func(req *service.RegisterRequest) (*model.User, error) {
			return nil, service.ErrUserExists
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "pass1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestUserHandler_Login_Success(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		LoginFn: func(req *service.LoginRequest) (*service.LoginResponse, error) {
			return &service.LoginResponse{
				AccessToken:  "at",
				RefreshToken: "rt",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				User:         &model.User{ID: 1},
			}, nil
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "password": "pw"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "at", resp["access_token"])
}

func TestUserHandler_Login_InvalidCredentials(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		LoginFn: func(req *service.LoginRequest) (*service.LoginResponse, error) {
			return nil, service.ErrInvalidCredentials
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// GetCurrentUser
// ---------------------------------------------------------------------------

func TestUserHandler_GetCurrentUser_Success(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		GetByIDFn: func(id uint) (*model.User, error) {
			return &model.User{ID: 1, Username: "alice"}, nil
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetCurrentUser_NotFound(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		GetByIDFn: func(id uint) (*model.User, error) { return nil, service.ErrUserNotFound },
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateCurrentUser
// ---------------------------------------------------------------------------

func TestUserHandler_UpdateCurrentUser_Success(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		UpdateFn: func(id uint, req *service.UpdateUserRequest) (*model.User, error) {
			return &model.User{ID: 1, Nickname: req.Nickname}, nil
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{"nickname": "New Nick"})
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// RefreshToken
// ---------------------------------------------------------------------------

func TestUserHandler_RefreshToken_Success(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		RefreshAccessTokenFn: func(refreshToken string) (*service.LoginResponse, error) {
			return &service.LoginResponse{
				AccessToken:  "new-at",
				RefreshToken: "new-rt",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				User:         &model.User{ID: 1},
			}, nil
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{"refresh_token": "old-rt"})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_RefreshToken_Invalid(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		RefreshAccessTokenFn: func(refreshToken string) (*service.LoginResponse, error) {
			return nil, errors.New("invalid token")
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{"refresh_token": "bad-rt"})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestUserHandler_Logout_Success(t *testing.T) {
	h := handler.NewUserHandler(&testutil.MockUserService{
		RevokeRefreshTokenFn: func(refreshToken string) error { return nil },
	})
	router := newUserTestRouter(h)

	body, _ := json.Marshal(map[string]string{"refresh_token": "rt"})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetPublicUser
// ---------------------------------------------------------------------------

func TestUserHandler_GetPublicUser_Success(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		GetPublicByIDFn: func(id uint) (*service.PublicUser, error) {
			return &service.PublicUser{ID: id, Username: "alice", CreatedAt: time.Now()}, nil
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetPublicUser_NotFound(t *testing.T) {
	mockSvc := &testutil.MockUserService{
		GetPublicByIDFn: func(id uint) (*service.PublicUser, error) {
			return nil, service.ErrUserNotFound
		},
	}
	h := handler.NewUserHandler(mockSvc)
	router := newUserTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
