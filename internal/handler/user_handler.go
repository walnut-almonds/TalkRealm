package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// UserHandler 使用者處理器
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 建立使用者處理器
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Register 使用者註冊
// @Summary 使用者註冊
// @Tags auth
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "註冊資訊"
// @Success 201 {object} model.User
// @Router /api/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	user, err := h.userService.Register(&req)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "user already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to register user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user":    user,
	})
}

// Login 使用者登入
// @Summary 使用者登入
// @Tags auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "登入資訊"
// @Success 200 {object} service.LoginResponse
// @Router /api/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	resp, err := h.userService.Login(&req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to login: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"token":   resp.Token,
		"user":    resp.User,
	})
}

// GetCurrentUser 取得當前使用者資訊
// @Summary 取得當前使用者資訊
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.User
// @Router /api/users/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// 從 context 取得使用者 ID（由認證中間件設定）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	user, err := h.userService.GetByID(userID.(uint))
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// UpdateCurrentUser 更新當前使用者資訊
// @Summary 更新當前使用者資訊
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.UpdateUserRequest true "更新資訊"
// @Success 200 {object} model.User
// @Router /api/users/me [patch]
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	user, err := h.userService.Update(userID.(uint), &req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user updated successfully",
		"user":    user,
	})
}
