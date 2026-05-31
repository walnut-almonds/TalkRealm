package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// UnreadHandler 處理未讀計數相關 API
type UnreadHandler struct {
	unreadService service.UnreadService
}

// NewUnreadHandler 建立 UnreadHandler
func NewUnreadHandler(unreadService service.UnreadService) *UnreadHandler {
	return &UnreadHandler{unreadService: unreadService}
}

// GetAllUnread 取得目前使用者所有頻道的未讀計數
//
//	@Summary	取得所有頻道未讀計數
//	@Tags		Unread
//	@Produce	json
//	@Success	200	{array}		service.ChannelUnreadCount
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/v1/users/me/unread [get]
func (h *UnreadHandler) GetAllUnread(c *gin.Context) {
	userID := c.GetUint("user_id")

	counts, err := h.unreadService.GetAllUnread(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, counts)
}

// GetChannelUnread 取得特定頻道的未讀計數
//
//	@Summary	取得頻道未讀計數
//	@Tags		Unread
//	@Produce	json
//	@Param		id	path		int	true	"頻道 ID"
//	@Success	200	{object}	service.ChannelUnreadCount
//	@Failure	400	{object}	ErrorResponse
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/v1/channels/{id}/unread [get]
func (h *UnreadHandler) GetChannelUnread(c *gin.Context) {
	userID := c.GetUint("user_id")

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	count, err := h.unreadService.GetChannelUnread(userID, uint(channelID))
	if err != nil {
		if errors.Is(err, service.ErrUnreadAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, count)
}

// AckChannel 標記頻道已讀
//
//	@Summary	標記頻道已讀
//	@Tags		Unread
//	@Accept		json
//	@Produce	json
//	@Param		id		path	int				true	"頻道 ID"
//	@Param		request	body	ackChannelReq	true	"最後讀取的訊息 ID"
//	@Success	204
//	@Failure	400	{object}	ErrorResponse
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/v1/channels/{id}/ack [post]
func (h *UnreadHandler) AckChannel(c *gin.Context) {
	userID := c.GetUint("user_id")

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	var req ackChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.unreadService.AckChannel(userID, uint(channelID), req.LastMessageID); err != nil {
		if errors.Is(err, service.ErrUnreadAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		if errors.Is(err, service.ErrUnreadInvalidAckTarget) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ack target"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.Status(http.StatusNoContent)
}

type ackChannelReq struct {
	LastMessageID uint `json:"last_message_id" binding:"required"`
}
