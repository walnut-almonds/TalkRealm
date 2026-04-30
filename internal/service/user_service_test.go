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
	"github.com/walnut-almonds/talkrealm/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret-key", time.Hour)
}

func hashedPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestUserService_Register_Success(t *testing.T) {
	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn:    func(email string) (*model.User, error) { return nil, errors.New("not found") },
		GetByUsernameFn: func(username string) (*model.User, error) { return nil, errors.New("not found") },
		CreateFn:        func(user *model.User) error { user.ID = 1; return nil },
	}
	mockRTRepo := &testutil.MockRefreshTokenRepository{}
	svc := service.NewUserService(mockRepo, mockRTRepo, newTestJWTManager())

	user, err := svc.Register(&service.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Nickname) // defaults to username
}

func TestUserService_Register_NicknameSet(t *testing.T) {
	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn:    func(email string) (*model.User, error) { return nil, errors.New("not found") },
		GetByUsernameFn: func(username string) (*model.User, error) { return nil, errors.New("not found") },
		CreateFn:        func(user *model.User) error { user.ID = 2; return nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	user, err := svc.Register(&service.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "pass1234",
		Nickname: "Alice W",
	})

	require.NoError(t, err)
	assert.Equal(t, "Alice W", user.Nickname)
}

func TestUserService_Register_DuplicateEmail(t *testing.T) {
	existing := &model.User{ID: 1, Email: "test@example.com"}
	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn: func(email string) (*model.User, error) { return existing, nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	_, err := svc.Register(&service.RegisterRequest{
		Username: "other",
		Email:    "test@example.com",
		Password: "pass1234",
	})

	assert.ErrorIs(t, err, service.ErrUserExists)
}

func TestUserService_Register_DuplicateUsername(t *testing.T) {
	existing := &model.User{ID: 1, Username: "testuser"}
	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn:    func(email string) (*model.User, error) { return nil, errors.New("not found") },
		GetByUsernameFn: func(username string) (*model.User, error) { return existing, nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	_, err := svc.Register(&service.RegisterRequest{
		Username: "testuser",
		Email:    "new@example.com",
		Password: "pass1234",
	})

	assert.ErrorIs(t, err, service.ErrUserExists)
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestUserService_Login_Success(t *testing.T) {
	pw := hashedPassword(t, "correct-pw")
	existingUser := &model.User{ID: 1, Username: "bob", Email: "bob@example.com", Password: pw}

	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn:   func(email string) (*model.User, error) { return existingUser, nil },
		UpdateStatusFn: func(id uint, status string) error { return nil },
	}
	mockRTRepo := &testutil.MockRefreshTokenRepository{
		CreateFn: func(token *model.RefreshToken) error { return nil },
	}
	svc := service.NewUserService(mockRepo, mockRTRepo, newTestJWTManager())

	resp, err := svc.Login(&service.LoginRequest{Email: "bob@example.com", Password: "correct-pw"})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	pw := hashedPassword(t, "correct-pw")
	existingUser := &model.User{ID: 1, Password: pw}

	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn: func(email string) (*model.User, error) { return existingUser, nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	_, err := svc.Login(&service.LoginRequest{Email: "bob@example.com", Password: "wrong-pw"})

	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	mockRepo := &testutil.MockUserRepository{
		GetByEmailFn: func(email string) (*model.User, error) { return nil, errors.New("not found") },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	_, err := svc.Login(&service.LoginRequest{Email: "missing@example.com", Password: "pw"})

	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestUserService_GetByID_Success(t *testing.T) {
	user := &model.User{ID: 5, Username: "charlie"}
	mockRepo := &testutil.MockUserRepository{
		GetByIDFn: func(id uint) (*model.User, error) { return user, nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	got, err := svc.GetByID(5)
	require.NoError(t, err)
	assert.Equal(t, uint(5), got.ID)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	mockRepo := &testutil.MockUserRepository{
		GetByIDFn: func(id uint) (*model.User, error) { return nil, errors.New("user not found") },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	_, err := svc.GetByID(999)
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

// ---------------------------------------------------------------------------
// GetPublicByID
// ---------------------------------------------------------------------------

func TestUserService_GetPublicByID_Success(t *testing.T) {
	user := &model.User{ID: 3, Username: "diana", Nickname: "Di", Status: "online"}
	mockRepo := &testutil.MockUserRepository{
		GetByIDFn: func(id uint) (*model.User, error) { return user, nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	pub, err := svc.GetPublicByID(3)
	require.NoError(t, err)
	assert.Equal(t, uint(3), pub.ID)
	assert.Equal(t, "diana", pub.Username)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserService_Update_Success(t *testing.T) {
	user := &model.User{ID: 1, Username: "eve", Nickname: "Eve"}
	mockRepo := &testutil.MockUserRepository{
		GetByIDFn: func(id uint) (*model.User, error) { return user, nil },
		UpdateFn:  func(u *model.User) error { return nil },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	updated, err := svc.Update(1, &service.UpdateUserRequest{Nickname: "Eve Updated"})
	require.NoError(t, err)
	assert.Equal(t, "Eve Updated", updated.Nickname)
}

func TestUserService_Update_UserNotFound(t *testing.T) {
	mockRepo := &testutil.MockUserRepository{
		GetByIDFn: func(id uint) (*model.User, error) { return nil, errors.New("not found") },
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	_, err := svc.Update(999, &service.UpdateUserRequest{Nickname: "New"})
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

// ---------------------------------------------------------------------------
// RefreshAccessToken
// ---------------------------------------------------------------------------

func TestUserService_RefreshAccessToken_Success(t *testing.T) {
	user := &model.User{ID: 1, Username: "frank", Email: "frank@example.com"}
	rt := &model.RefreshToken{
		Token:     "valid-rt",
		UserID:    1,
		Revoked:   false,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	mockRepo := &testutil.MockUserRepository{
		GetByIDFn: func(id uint) (*model.User, error) { return user, nil },
	}
	mockRTRepo := &testutil.MockRefreshTokenRepository{
		GetByTokenFn:    func(token string) (*model.RefreshToken, error) { return rt, nil },
		RevokeByTokenFn: func(token string) error { return nil },
		CreateFn:        func(token *model.RefreshToken) error { return nil },
	}
	svc := service.NewUserService(mockRepo, mockRTRepo, newTestJWTManager())

	resp, err := svc.RefreshAccessToken("valid-rt")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, "valid-rt", resp.RefreshToken) // token rotation
}

func TestUserService_RefreshAccessToken_RevokedToken(t *testing.T) {
	rt := &model.RefreshToken{Token: "revoked-rt", Revoked: true}
	mockRTRepo := &testutil.MockRefreshTokenRepository{
		GetByTokenFn: func(token string) (*model.RefreshToken, error) { return rt, nil },
	}
	svc := service.NewUserService(&testutil.MockUserRepository{}, mockRTRepo, newTestJWTManager())

	_, err := svc.RefreshAccessToken("revoked-rt")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestUserService_RefreshAccessToken_ExpiredToken(t *testing.T) {
	rt := &model.RefreshToken{Token: "expired-rt", Revoked: false, ExpiresAt: time.Now().Add(-time.Hour)}
	mockRTRepo := &testutil.MockRefreshTokenRepository{
		GetByTokenFn: func(token string) (*model.RefreshToken, error) { return rt, nil },
	}
	svc := service.NewUserService(&testutil.MockUserRepository{}, mockRTRepo, newTestJWTManager())

	_, err := svc.RefreshAccessToken("expired-rt")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

// ---------------------------------------------------------------------------
// RevokeRefreshToken
// ---------------------------------------------------------------------------

func TestUserService_RevokeRefreshToken(t *testing.T) {
	called := false
	mockRTRepo := &testutil.MockRefreshTokenRepository{
		RevokeByTokenFn: func(token string) error { called = true; return nil },
	}
	svc := service.NewUserService(&testutil.MockUserRepository{}, mockRTRepo, newTestJWTManager())

	err := svc.RevokeRefreshToken("some-token")
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestUserService_UpdateStatus(t *testing.T) {
	var gotID uint
	var gotStatus string
	mockRepo := &testutil.MockUserRepository{
		UpdateStatusFn: func(id uint, status string) error {
			gotID = id
			gotStatus = status
			return nil
		},
	}
	svc := service.NewUserService(mockRepo, &testutil.MockRefreshTokenRepository{}, newTestJWTManager())

	err := svc.UpdateStatus(42, "online")
	require.NoError(t, err)
	assert.Equal(t, uint(42), gotID)
	assert.Equal(t, "online", gotStatus)
}
