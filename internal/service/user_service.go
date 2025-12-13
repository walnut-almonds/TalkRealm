package service

import (
	"errors"
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
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// UpdateUserRequest 更新使用者請求
type UpdateUserRequest struct {
	Nickname string `json:"nickname" binding:"max=64"`
	Avatar   string `json:"avatar"   binding:"max=256"`
	Status   string `json:"status"   binding:"omitempty,oneof=online offline busy away"`
}

// UserService 使用者服務介面
type UserService interface {
	Register(req *RegisterRequest) (*model.User, error)
	Login(req *LoginRequest) (*LoginResponse, error)
	GetByID(id uint) (*model.User, error)
	Update(id uint, req *UpdateUserRequest) (*model.User, error)
	UpdateStatus(id uint, status string) error
}

type userService struct {
	repo       repository.UserRepository
	jwtManager *auth.JWTManager
}

// NewUserService 建立使用者服務
func NewUserService(repo repository.UserRepository, jwtManager *auth.JWTManager) UserService {
	return &userService{
		repo:       repo,
		jwtManager: jwtManager,
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
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Nickname:  req.Nickname,
		Status:    "offline",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 生成 JWT token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	// 更新使用者狀態為上線
	_ = s.repo.UpdateStatus(user.ID, "online")

	return &LoginResponse{
		Token: token,
		User:  user,
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
