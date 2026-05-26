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
	dmService service.DMService
}

// NewDMHandler 建立私訊處理器
func NewDMHandler(dmService service.DMService) *DMHandler {
	return &DMHandler{dmService: dmService}
}

// GetOrCreateDMChannel 取得或建立與目標使用者的 DM 頻道
//
//	@Summary		取得或建立 DM 頻道
//	@Description	與目標使用者開啟私訊（若頻道已存在則直接回傳）
//	@Tags			dm
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{target_user_id=integer}	true	"目標使用者 ID"
//	@Success		200		{object}	model.DirectMessageChannel
//	@Router			/api/v1/dm/channels [post]
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
//
//	@Summary		列出 DM 頻道
//	@Description	取得目前使用者的所有私訊對話
//	@Tags			dm
//	@Produce		json
//	@Success		200	{array}		model.DirectMessageChannel
//	@Router			/api/v1/dm/channels [get]
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

// ListDMMessages 取得 DM 頻道的訊息
//
//	@Summary		取得 DM 訊息
//	@Tags			dm
//	@Produce		json
//	@Param			id		path	int	true	"DM 頻道 ID"
//	@Param			before	query	int	false	"Cursor（訊息 ID，取此 ID 之前的訊息）"
//	@Param			limit	query	int	false	"筆數限制（預設 50，最大 100）"
//	@Success		200		{array}	model.DirectMessage
//	@Router			/api/v1/dm/channels/{id}/messages [get]
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

	msgs, err := h.dmService.ListMessages(uid, uint(channelID), uint(before), limit)
	if err != nil {
		if errors.Is(err, service.ErrDMNotParticipant) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this dm channel"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

// SendDMMessage 發送私訊（REST 方式）
//
//	@Summary		發送私訊
//	@Tags			dm
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int									true	"DM 頻道 ID"
//	@Param			request	body		object{content=string,nonce=string}	true	"訊息內容"
//	@Success		201		{object}	model.DirectMessage
//	@Router			/api/v1/dm/channels/{id}/messages [post]
func (h *DMHandler) SendDMMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	senderID := userID.(uint)

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
		Nonce   string `json:"nonce"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	msg, err := h.dmService.SendDM(senderID, uint(channelID), req.Content, req.Nonce)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDMNotParticipant):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this dm channel"})
		case errors.Is(err, service.ErrDMEmptyContent):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrDMDuplicateNonce):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusCreated, msg)
}
