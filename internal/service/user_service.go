package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

// RegisterRequest 註冊請求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=128"`
	Nickname string `json:"nickname" binding:"max=64"`
}

// LoginRequest 登入請求
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登入回應
type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"` // 秒數
	User         *model.User `json:"user"`
}

// PublicUser 使用者公開資料（不含 email 等敏感資訊）
type PublicUser struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateUserRequest 更新使用者請求
type UpdateUserRequest struct {
	Nickname      string `json:"nickname"       binding:"max=64"`
	Avatar        string `json:"avatar"         binding:"max=256"`
	Status        string `json:"status"         binding:"omitempty,oneof=offline busy away"`
	PreferredLang string `json:"preferred_lang" binding:"omitempty,oneof=zh zh-tw ja en"`
	UILocale      string `json:"ui_locale"      binding:"omitempty,oneof=zh zh-tw ja en"`
}

// OAuthUserInfo OAuth 登入時由 provider 提供的使用者資訊
type OAuthUserInfo struct {
	Provider   string // e.g. "google", "github"
	ProviderID string // 各家給的 subject ID
	Email      string
	Name       string
	Avatar     string
}

// UserService 使用者服務介面
type UserService interface {
	Register(req *RegisterRequest) (*model.User, error)
	Login(req *LoginRequest) (*LoginResponse, error)
	GetByID(id uint) (*model.User, error)
	GetPublicByID(id uint) (*PublicUser, error)
	Update(id uint, req *UpdateUserRequest) (*model.User, error)
	UpdateStatus(id uint, status string) error
	RefreshAccessToken(refreshToken string) (*LoginResponse, error)
	RevokeRefreshToken(refreshToken string) error
	OAuthLoginOrRegister(req *OAuthUserInfo) (*LoginResponse, error)
	SearchUsers(query string, excludeID uint) ([]*PublicUser, error)
}

type userService struct {
	repo              repository.UserRepository
	refreshTokenRepo  repository.RefreshTokenRepository
	oauthProviderRepo repository.OAuthProviderRepository
	jwtManager        *auth.JWTManager
}

// NewUserService 建立使用者服務
func NewUserService(
	repo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	oauthProviderRepo repository.OAuthProviderRepository,
	jwtManager *auth.JWTManager,
) UserService {
	return &userService{
		repo:              repo,
		refreshTokenRepo:  refreshTokenRepo,
		oauthProviderRepo: oauthProviderRepo,
		jwtManager:        jwtManager,
	}
}

// Register 註冊新使用者
func (s *userService) Register(req *RegisterRequest) (*model.User, error) {
	// 檢查 email 是否已存在
	existingUser, _ := s.repo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 檢查 username 是否已存在
	existingUser, _ = s.repo.GetByUsername(req.Username)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 加密密碼
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 建立使用者
	user := &model.User{
		Username:      req.Username,
		Email:         req.Email,
		Password:      string(hashedPassword),
		Nickname:      req.Nickname,
		Status:        "offline",
		PreferredLang: "zh",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 如果沒有提供 nickname，使用 username
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login 使用者登入
func (s *userService) Login(req *LoginRequest) (*LoginResponse, error) {
	// 查找使用者
	user, err := s.repo.GetByEmail(req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 驗證密碼
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 生成 access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	// 生成 refresh token
	refreshToken, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	rt := &model.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.refreshTokenRepo.Create(rt); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtManager.TokenDuration().Seconds()),
		User:         user,
	}, nil
}

// GetByID 透過 ID 取得使用者
func (s *userService) GetByID(id uint) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// Update 更新使用者資訊
func (s *userService) Update(id uint, req *UpdateUserRequest) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 更新欄位
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}

	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if req.Status != "" {
		user.Status = req.Status
	}

	if req.PreferredLang != "" {
		user.PreferredLang = req.PreferredLang
	}

	if req.UILocale != "" {
		user.UILocale = req.UILocale
	}

	user.UpdatedAt = time.Now()

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateStatus 更新使用者狀態
func (s *userService) UpdateStatus(id uint, status string) error {
	return s.repo.UpdateStatus(id, status)
}

// generateSecureToken 生成安全的隨機 token（64 字元 hex）
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// GetPublicByID 取得使用者公開資料（不含 email）
func (s *userService) GetPublicByID(id uint) (*PublicUser, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &PublicUser{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
	}, nil
}

// SearchUsers 以 username/nickname 模糊搜尋使用者（最多回傳 20 筆，排除自己）
func (s *userService) SearchUsers(query string, excludeID uint) ([]*PublicUser, error) {
	if len(query) < 1 {
		return []*PublicUser{}, nil
	}

	users, err := s.repo.SearchUsers(query, excludeID, 20)
	if err != nil {
		return nil, err
	}

	result := make([]*PublicUser, 0, len(users))
	for _, u := range users {
		result = append(result, &PublicUser{
			ID:        u.ID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Status:    u.Status,
			CreatedAt: u.CreatedAt,
		})
	}

	return result, nil
}

// RefreshAccessToken 使用 refresh token 換發新的 access token（token rotation）
func (s *userService) RefreshAccessToken(refreshToken string) (*LoginResponse, error) {
	rt, err := s.refreshTokenRepo.GetByToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 檢查是否已撤銷
	if rt.Revoked {
		return nil, ErrInvalidCredentials
	}

	// 檢查是否過期
	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrInvalidCredentials
	}

	// 取得使用者
	user, err := s.repo.GetByID(rt.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 生成新 access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	// Token rotation：撤銷舊 refresh token，生成新的
	if err := s.refreshTokenRepo.RevokeByToken(refreshToken); err != nil {
		return nil, err
	}

	newRefreshToken, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	nrt := &model.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshToken,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.refreshTokenRepo.Create(nrt); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtManager.TokenDuration().Seconds()),
		User:         user,
	}, nil
}

// RevokeRefreshToken 撤銷 refresh token（登出）
func (s *userService) RevokeRefreshToken(refreshToken string) error {
	return s.refreshTokenRepo.RevokeByToken(refreshToken)
}

// OAuthLoginOrRegister 透過 OAuth 資訊登入或自動建立帳號
func (s *userService) OAuthLoginOrRegister(info *OAuthUserInfo) (*LoginResponse, error) {
	var user *model.User

	// 查詢此 provider + provider_id 是否已綁定與某個帳號
	link, err := s.oauthProviderRepo.FindByProvider(info.Provider, info.ProviderID)
	if err != nil {
		return nil, err
	}

	if link != nil {
		// 已綁定：直接取得對應使用者
		user, err = s.repo.GetByID(link.UserID)
		if err != nil {
			return nil, err
		}
	} else {
		// 找不到對應連結，嘗試以 email 匹配就有帳號（本地或其他 OAuth 帳號）
		if info.Email != "" {
			existing, _ := s.repo.GetByEmail(info.Email)
			if existing != nil {
				user = existing
			}
		}

		// email 也找不到，自動建立新帳號
		if user == nil {
			username := generateUsernameFromEmail(info.Email)
			base := username

			for i := 1; ; i++ {
				if u, _ := s.repo.GetByUsername(username); u == nil {
					break
				}

				username = fmt.Sprintf("%s%d", base, i)
			}

			nickname := info.Name
			if nickname == "" {
				nickname = username
			}

			user = &model.User{
				Username:      username,
				Email:         info.Email,
				Nickname:      nickname,
				Avatar:        info.Avatar,
				Status:        "offline",
				PreferredLang: "zh",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if err := s.repo.Create(user); err != nil {
				return nil, err
			}
		}

		// 建立新的 OAuth 連結記錄
		newLink := &model.UserOAuthProvider{
			UserID:     user.ID,
			Provider:   info.Provider,
			ProviderID: info.ProviderID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := s.oauthProviderRepo.Create(newLink); err != nil {
			return nil, err
		}
	}

	// 更新頭像（若沒有且 OAuth 有提供）
	if user.Avatar == "" && info.Avatar != "" {
		user.Avatar = info.Avatar
		user.UpdatedAt = time.Now()
		_ = s.repo.Update(user)
	}

	// 生成 access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	// 生成 refresh token
	refreshToken, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	rt := &model.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.refreshTokenRepo.Create(rt); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtManager.TokenDuration().Seconds()),
		User:         user,
	}, nil
}

// generateUsernameFromEmail 從 email 產生基礎 username（取 @ 前半段，去除特殊字元）
func generateUsernameFromEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "user"
	}
	// 只保留英數字與底線
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}

		return '_'
	}, parts[0])
	if len(result) < 3 {
		result = result + "user"
	}

	return result
}
