package handler

import (
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
// @Summary 建立訊息
// @Description 在指定頻道中建立新訊息
// @Tags messages
// @Accept json
// @Produce json
// @Param request body service.CreateMessageRequest true "建立訊息請求"
// @Success 201 {object} model.Message
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/messages [post]
func (h *MessageHandler) CreateMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req service.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.messageService.CreateMessage(userID.(uint), &req)
	if err != nil {
		switch err {
		case service.ErrNotChannelMemberMsg:
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not a member of this channel's guild"})
		case service.ErrEmptyMessageContent:
			c.JSON(http.StatusBadRequest, gin.H{"error": "message content cannot be empty"})
		case service.ErrInvalidMessageType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, message)
}

// GetMessage 取得訊息
// @Summary 取得訊息
// @Description 取得指定 ID 的訊息詳情
// @Tags messages
// @Produce json
// @Param id path int true "訊息 ID"
// @Success 200 {object} model.Message
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/messages/{id} [get]
func (h *MessageHandler) GetMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("userID")
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
		switch err {
		case service.ErrMessageNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case service.ErrNotChannelMemberMsg:
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not a member of this channel's guild"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, message)
}

// ListChannelMessages 列出頻道訊息
// @Summary 列出頻道訊息
// @Description 列出指定頻道的所有訊息（分頁）
// @Tags messages
// @Produce json
// @Param channel_id query int true "頻道 ID"
// @Param page query int false "頁碼" default(1)
// @Param page_size query int false "每頁數量" default(50)
// @Success 200 {object} service.MessageListResponse
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/messages [get]
func (h *MessageHandler) ListChannelMessages(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 取得頻道 ID
	channelIDStr := c.Query("channel_id")
	if channelIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id is required"})
		return
	}

	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel_id"})
		return
	}

	// 取得分頁參數
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 50
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	response, err := h.messageService.ListChannelMessages(uint(channelID), userID.(uint), page, pageSize)
	if err != nil {
		switch err {
		case service.ErrNotChannelMemberMsg:
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not a member of this channel's guild"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateMessage 更新訊息
// @Summary 更新訊息
// @Description 更新自己發送的訊息內容
// @Tags messages
// @Accept json
// @Produce json
// @Param id path int true "訊息 ID"
// @Param request body service.UpdateMessageRequest true "更新訊息請求"
// @Success 200 {object} model.Message
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/messages/{id} [put]
func (h *MessageHandler) UpdateMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("userID")
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
		switch err {
		case service.ErrMessageNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case service.ErrNotMessageOwner:
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not the owner of this message"})
		case service.ErrEmptyMessageContent:
			c.JSON(http.StatusBadRequest, gin.H{"error": "message content cannot be empty"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, message)
}

// DeleteMessage 刪除訊息
// @Summary 刪除訊息
// @Description 刪除自己的訊息，或管理員刪除任何訊息
// @Tags messages
// @Param id path int true "訊息 ID"
// @Success 200 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/messages/{id} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	// 從 context 取得使用者 ID
	userID, exists := c.Get("userID")
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
		switch err {
		case service.ErrMessageNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case service.ErrNotMessageOwner:
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not authorized to delete this message"})
		case service.ErrNotChannelMemberMsg:
			c.JSON(http.StatusForbidden, gin.H{"error": "you are not a member of this channel's guild"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted successfully"})
}
