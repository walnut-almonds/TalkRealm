package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

type ChannelHandler struct {
	channelService service.ChannelService
}

func NewChannelHandler(channelService service.ChannelService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
	}
}

// CreateChannel 建立頻道
//
//	@Summary		建立頻道
//	@Description	在社群中建立新的文字或語音頻道（僅擁有者或管理員）
//	@Tags			Channel
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateChannelRequest	true	"建立頻道請求"
//	@Success		201		{object}	model.Channel
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Router			/api/v1/channels [post]
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req service.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	channel, err := h.channelService.CreateChannel(userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		if errors.Is(err, service.ErrNotGuildMemberCh) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this guild"})
			return
		}

		if errors.Is(err, service.ErrInvalidChannelType) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "invalid channel type, must be 'text' or 'voice'"},
			)

			return
		}

		if err.Error() == "only owner or admin can create channels" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusCreated, channel)
}

// GetChannel 取得頻道詳情
//
//	@Summary		取得頻道詳情
//	@Description	取得指定頻道的詳細資訊
//	@Tags			Channel
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"頻道 ID"
//	@Success		200	{object}	model.Channel
//	@Failure		400	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/channels/{id} [get]
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	userID := c.GetUint("user_id")

	channel, err := h.channelService.GetChannel(uint(channelID), userID)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		if errors.Is(err, service.ErrNotGuildMemberCh) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this guild"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, channel)
}

// ListGuildChannels 列出社群的頻道
//
//	@Summary		列出社群的頻道
//	@Description	列出指定社群的所有頻道
//	@Tags			Channel
//	@Accept			json
//	@Produce		json
//	@Param			guild_id	query		int	true	"社群 ID"
//	@Success		200			{array}		model.Channel
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/v1/channels [get]
func (h *ChannelHandler) ListGuildChannels(c *gin.Context) {
	guildIDStr := c.Query("guild_id")
	if guildIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guild_id is required"})
		return
	}

	guildID, err := strconv.ParseUint(guildIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild_id"})
		return
	}

	userID := c.GetUint("user_id")

	channels, err := h.channelService.ListGuildChannels(uint(guildID), userID)
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		if errors.Is(err, service.ErrNotGuildMemberCh) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this guild"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, channels)
}

// UpdateChannel 更新頻道
//
//	@Summary		更新頻道
//	@Description	更新頻道資訊（僅擁有者或管理員）
//	@Tags			Channel
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"頻道 ID"
//	@Param			request	body		service.UpdateChannelRequest	true	"更新頻道請求"
//	@Success		200		{object}	model.Channel
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/api/v1/channels/{id} [put]
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	var req service.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	channel, err := h.channelService.UpdateChannel(uint(channelID), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		if errors.Is(err, service.ErrInvalidChannelType) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "invalid channel type, must be 'text' or 'voice'"},
			)

			return
		}

		if err.Error() == "only owner or admin can update channels" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, channel)
}

// DeleteChannel 刪除頻道
//
//	@Summary		刪除頻道
//	@Description	刪除頻道（僅擁有者或管理員）
//	@Tags			Channel
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"頻道 ID"
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/channels/{id} [delete]
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = h.channelService.DeleteChannel(uint(channelID), userID)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		if err.Error() == "only owner or admin can delete channels" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "channel deleted successfully"})
}

// UpdateChannelPosition 更新頻道位置
//
//	@Summary		更新頻道位置
//	@Description	更新頻道在列表中的位置（僅擁有者或管理員）
//	@Tags			Channel
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"頻道 ID"
//	@Param			request	body		PositionRequest	true	"位置請求"
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/api/v1/channels/{id}/position [put]
func (h *ChannelHandler) UpdateChannelPosition(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	var req PositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	err = h.channelService.UpdateChannelPosition(uint(channelID), userID, req.Position)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		if err.Error() == "only owner or admin can update channel position" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "channel position updated successfully"})
}

type PositionRequest struct {
	Position int `json:"position" binding:"required,min=0"`
}
