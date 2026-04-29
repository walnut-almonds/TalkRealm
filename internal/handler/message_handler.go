package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// MessageHandler 訊息處理器
type MessageHandler struct {
	messageService service.MessageService
}

// NewMessageHandler 建立訊息處理器實例
func NewMessageHandler(messageService service.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// CreateMessage 建立訊息
//
//	@Summary		建立訊息
//	@Description	在指定頻道中建立新訊息
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"頻道 ID"
//	@Param			request	body		service.CreateMessageRequest	true	"建立訊息請求"
//	@Success		201		{object}	model.Message
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/channels/{id}/messages [post]
func (h *MessageHandler) CreateMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 從 URL 路徑參數取得頻道 ID
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	var req service.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 設定從 URL 取得的 channelID
	req.ChannelID = uint(channelID)

	message, err := h.messageService.CreateMessage(userID.(uint), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "you are not a member of this channel's guild"},
			)
		case errors.Is(err, service.ErrEmptyMessageContent):
			c.JSON(http.StatusBadRequest, gin.H{"error": "message content cannot be empty"})
		case errors.Is(err, service.ErrInvalidMessageType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusCreated, message)
}

// GetMessage 取得訊息
//
//	@Summary		取得訊息
//	@Description	取得指定 ID 的訊息詳情
//	@Tags			messages
//	@Produce		json
//	@Param			id	path		int	true	"訊息 ID"
//	@Success		200	{object}	model.Message
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/messages/{id} [get]
func (h *MessageHandler) GetMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 取得訊息 ID
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	message, err := h.messageService.GetMessage(uint(messageID), userID.(uint))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "you are not a member of this channel's guild"},
			)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, message)
}

// ListChannelMessages 列出頻道訊息
//
//	@Summary		列出頻道訊息
//	@Description	列出指定頻道的訊息（cursor-based 分頁）
//	@Tags			messages
//	@Produce		json
//	@Param			id		path		int	true	"頻道 ID"
//	@Param			before	query		int	false	"訊息 ID，取得此 ID 之前的訊息"
//	@Param			limit	query		int	false	"每次取得數量 (1-100)"	default(50)
//	@Success		200		{object}	service.MessageListResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/api/v1/channels/{id}/messages [get]
func (h *MessageHandler) ListChannelMessages(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 取得頻道 ID
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	// cursor: before message ID
	var before uint
	if beforeStr := c.Query("before"); beforeStr != "" {
		if b, err := strconv.ParseUint(beforeStr, 10, 32); err == nil {
			before = uint(b)
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	response, err := h.messageService.ListChannelMessages(
		uint(channelID),
		userID.(uint),
		limit,
		before,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "you are not a member of this channel's guild"},
			)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateMessage 更新訊息
//
//	@Summary		更新訊息
//	@Description	更新自己發送的訊息內容
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"訊息 ID"
//	@Param			request	body		service.UpdateMessageRequest	true	"更新訊息請求"
//	@Success		200		{object}	model.Message
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/api/v1/messages/{id} [put]
func (h *MessageHandler) UpdateMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 取得訊息 ID
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	var req service.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.messageService.UpdateMessage(uint(messageID), userID.(uint), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotMessageOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this message"})
		case errors.Is(err, service.ErrEmptyMessageContent):
			c.JSON(http.StatusBadRequest, gin.H{"error": "message content cannot be empty"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, message)
}

// DeleteMessage 刪除訊息
//
//	@Summary		刪除訊息
//	@Description	刪除自己的訊息，或管理員刪除任何訊息
//	@Tags			messages
//	@Param			id	path		int	true	"訊息 ID"
//	@Success		200	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/messages/{id} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 取得訊息 ID
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	err = h.messageService.DeleteMessage(uint(messageID), userID.(uint))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotMessageOwner):
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "you are not authorized to delete this message"},
			)
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "you are not a member of this channel's guild"},
			)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted successfully"})
}
