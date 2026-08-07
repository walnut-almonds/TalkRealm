package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// DMHandler 私訊處理器
type DMHandler struct {
	dmService          service.DMService
	messageService     service.MessageService
	translationService service.TranslationService
}

// NewDMHandler 建立私訊處理器
func NewDMHandler(
	dmService service.DMService,
	messageService service.MessageService,
	translationService service.TranslationService,
) *DMHandler {
	return &DMHandler{
		dmService:          dmService,
		messageService:     messageService,
		translationService: translationService,
	}
}

// GetOrCreateDMChannel 取得或建立與目標使用者的 DM 頻道
func (h *DMHandler) GetOrCreateDMChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	requesterID := userID.(uint)

	var req struct {
		TargetUserID uint `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_user_id is required"})
		return
	}

	ch, err := h.dmService.GetOrCreateChannel(requesterID, req.TargetUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDMSelfMessage):
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open dm with yourself"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, ch)
}

// ListDMChannels 列出使用者的所有 DM 頻道
func (h *DMHandler) ListDMChannels(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	channels, err := h.dmService.ListDMChannels(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// ListDMMessages 取得 DM 頻道的訊息（重用 MessageService.ListChannelMessages）
func (h *DMHandler) ListDMMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	before, _ := strconv.ParseUint(c.Query("before"), 10, 32)

	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 50
	}

	result, err := h.messageService.ListChannelMessages(uint(channelID), uid, limit, uint(before))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this dm channel"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, result)
}

// SendDMMessage 發送私訊（REST 方式）
func (h *DMHandler) SendDMMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	senderID := userID.(uint)

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	var req struct {
		Content string `json:"content"`
		Nonce   string `json:"nonce"`
		FileIDs []uint `json:"file_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Content == "" && len(req.FileIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content or file_ids is required"})
		return
	}

	msg, err := h.messageService.CreateMessage(senderID, &service.CreateMessageRequest{
		ChannelID: uint(channelID),
		Content:   req.Content,
		Type:      "text",
		Nonce:     req.Nonce,
		FileIDs:   req.FileIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this dm channel"})
		case errors.Is(err, service.ErrEmptyMessageContent):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrDuplicateNonce):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrFileNotAttachable):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusCreated, msg)
}

// UpdateDMMessage 編輯私訊
func (h *DMHandler) UpdateDMMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.messageService.UpdateMessage(
		uint(messageID),
		uid,
		&service.UpdateMessageRequest{Content: req.Content},
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotMessageOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": "not the owner of this message"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, msg)
}

// DeleteDMMessage 刪除私訊
func (h *DMHandler) DeleteDMMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	if err := h.messageService.DeleteMessage(uint(messageID), uid); err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotMessageOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": "not the owner of this message"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// EnsureDMTranslation 取得或觸發 DM 訊息翻譯
func (h *DMHandler) EnsureDMTranslation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	lang := c.Query("lang")

	// 先取得訊息以驗證存取權並獲得 content
	msg, err := h.messageService.GetMessage(uint(messageID), uid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this dm channel"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	result, err := h.translationService.EnsureTranslation(
		uint(messageID),
		msg.Content,
		msg.ChannelID,
		lang,
	)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	if result == nil {
		c.JSON(http.StatusOK, gin.H{"status": "processing"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDMTranslation 取得已存在的 DM 訊息翻譯
func (h *DMHandler) GetDMTranslation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	if _, err := h.messageService.GetMessage(uint(messageID), uid); err != nil {
		switch {
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		case errors.Is(err, service.ErrNotChannelMemberMsg):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this dm channel"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	result, err := h.translationService.GetTranslation(uint(messageID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "translation not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}
