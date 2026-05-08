package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/pkg/voice"
)

// VoiceHandler 處理語音相關 API
type VoiceHandler struct {
	voiceManager *voice.Manager
}

// NewVoiceHandler 建立 VoiceHandler
func NewVoiceHandler(vm *voice.Manager) *VoiceHandler {
	return &VoiceHandler{voiceManager: vm}
}

// GetVoiceToken 取得 LiveKit Room Token
//
//	@Summary		取得語音頻道 Token
//	@Description	回傳可用於連接 LiveKit 語音房間的 JWT token 及伺服器 URL
//	@Tags			Voice
//	@Produce		json
//	@Param			id	path		int	true	"頻道 ID"
//	@Success		200	{object}	voice.RoomTokenResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Router			/api/v1/channels/{id}/voice/token [get]
func (h *VoiceHandler) GetVoiceToken(c *gin.Context) {
	if !h.voiceManager.IsConfigured() {
		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{"error": "voice service unavailable: livekit not configured"},
		)

		return
	}

	channelIDStr := c.Param("id")

	channelID, err := strconv.ParseUint(channelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel_id"})
		return
	}

	userID := c.GetUint("user_id")
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)
	resp, err := h.voiceManager.GenerateRoomToken(uint(channelID), userID, usernameStr)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
