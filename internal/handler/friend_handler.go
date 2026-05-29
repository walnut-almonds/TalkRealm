package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// FriendHandler 好友系統 HTTP 處理器
type FriendHandler struct {
	friendService service.FriendService
}

// NewFriendHandler 建立好友處理器
func NewFriendHandler(friendService service.FriendService) *FriendHandler {
	return &FriendHandler{friendService: friendService}
}

// SendRequest POST /api/v1/friends — 透過 username 送出好友申請
func (h *FriendHandler) SendRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := h.friendService.SendRequest(userID.(uint), req.Username)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case errors.Is(err, service.ErrCannotFriendSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot send friend request to yourself"})
		case errors.Is(err, service.ErrAlreadyFriends):
			c.JSON(http.StatusConflict, gin.H{"error": "already friends"})
		case errors.Is(err, service.ErrFriendRequestExists):
			c.JSON(http.StatusConflict, gin.H{"error": "friend request already sent"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusCreated, f)
}

// ListFriends GET /api/v1/friends — 列出已接受的好友
func (h *FriendHandler) ListFriends(c *gin.Context) {
	userID, _ := c.Get("user_id")

	friends, err := h.friendService.ListFriends(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"friends": friends})
}

// ListIncomingRequests GET /api/v1/friends/requests/incoming — 收到的待處理申請
func (h *FriendHandler) ListIncomingRequests(c *gin.Context) {
	userID, _ := c.Get("user_id")

	reqs, err := h.friendService.ListIncomingRequests(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": reqs})
}

// ListOutgoingRequests GET /api/v1/friends/requests/outgoing — 送出的待處理申請
func (h *FriendHandler) ListOutgoingRequests(c *gin.Context) {
	userID, _ := c.Get("user_id")

	reqs, err := h.friendService.ListOutgoingRequests(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": reqs})
}

// AcceptRequest PUT /api/v1/friends/:userId/accept — 接受好友申請
func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")

	requesterID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	f, err := h.friendService.Accept(userID.(uint), uint(requesterID))
	if err != nil {
		if errors.Is(err, service.ErrFriendRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friend request not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, f)
}

// RemoveFriend DELETE /api/v1/friends/:userId — 拒絕申請或解除好友
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	userID, _ := c.Get("user_id")

	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	uid := userID.(uint)
	tid := uint(targetID)

	// 先嘗試拒絕對方送來的申請，再嘗試解除好友關係
	if rejectErr := h.friendService.Reject(uid, tid); rejectErr != nil {
		// 不是待處理申請，嘗試解除已接受的好友關係
		if err := h.friendService.Unfriend(uid, tid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}
