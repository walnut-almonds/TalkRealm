package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

type GuildHandler struct {
	guildService       service.GuildService
	guildMemberService service.GuildMemberService
}

func NewGuildHandler(
	guildService service.GuildService,
	guildMemberService service.GuildMemberService,
) *GuildHandler {
	return &GuildHandler{
		guildService:       guildService,
		guildMemberService: guildMemberService,
	}
}

// CreateGuild 建立社群
//
//	@Summary		建立社群
//	@Description	建立一個新的社群，建立者自動成為擁有者
//	@Tags			Guild
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateGuildRequest	true	"建立社群請求"
//	@Success		201		{object}	model.Guild
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/api/v1/guilds [post]
func (h *GuildHandler) CreateGuild(c *gin.Context) {
	var req service.CreateGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get userID from context with detailed logging
	userIDValue, exists := c.Get("user_id")
	logger.Info("CreateGuild context check",
		"user_id_exists", exists,
		"user_id_value", userIDValue,
		"user_id_type", fmt.Sprintf("%T", userIDValue))

	userID := c.GetUint("user_id")
	logger.Info("CreateGuild userID retrieved", "userID", userID)

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
		return
	}

	guild, err := h.guildService.CreateGuild(userID, &req)
	if err != nil {
		logger.Error("CreateGuild failed", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, guild)
}

// GetGuild 取得社群詳情
//
//	@Summary		取得社群詳情
//	@Description	取得指定社群的詳細資訊
//	@Tags			Guild
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"社群 ID"
//	@Success		200	{object}	model.Guild
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id} [get]
func (h *GuildHandler) GetGuild(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	guild, err := h.guildService.GetGuild(uint(guildID))
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, guild)
}

// ListUserGuilds 列出使用者的社群
//
//	@Summary		列出使用者的社群
//	@Description	列出當前使用者所屬的所有社群
//	@Tags			Guild
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		model.Guild
//	@Failure		401	{object}	ErrorResponse
//	@Router			/api/v1/guilds [get]
func (h *GuildHandler) ListUserGuilds(c *gin.Context) {
	userID := c.GetUint("user_id")

	guilds, err := h.guildService.ListUserGuilds(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, guilds)
}

// UpdateGuild 更新社群
//
//	@Summary		更新社群
//	@Description	更新社群資訊（僅擁有者）
//	@Tags			Guild
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"社群 ID"
//	@Param			request	body		service.UpdateGuildRequest	true	"更新社群請求"
//	@Success		200		{object}	model.Guild
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id} [put]
func (h *GuildHandler) UpdateGuild(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	var req service.UpdateGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	guild, err := h.guildService.UpdateGuild(uint(guildID), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrNotGuildOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can update guild"})
			return
		}

		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, guild)
}

// DeleteGuild 刪除社群
//
//	@Summary		刪除社群
//	@Description	刪除社群（僅擁有者）
//	@Tags			Guild
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"社群 ID"
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id} [delete]
func (h *GuildHandler) DeleteGuild(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = h.guildService.DeleteGuild(uint(guildID), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotGuildOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can delete guild"})
			return
		}

		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "guild deleted successfully"})
}

// JoinGuild 加入社群
//
//	@Summary		加入社群
//	@Description	使用者加入指定社群
//	@Tags			GuildMember
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"社群 ID"
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id}/join [post]
func (h *GuildHandler) JoinGuild(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = h.guildMemberService.JoinGuild(uint(guildID), userID)
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		if errors.Is(err, service.ErrAlreadyInGuild) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "already in guild"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "joined guild successfully"})
}

// LeaveGuild 離開社群
//
//	@Summary		離開社群
//	@Description	使用者離開社群（擁有者需先轉移所有權）
//	@Tags			GuildMember
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"社群 ID"
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id}/leave [post]
func (h *GuildHandler) LeaveGuild(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = h.guildMemberService.LeaveGuild(uint(guildID), userID)
	if err != nil {
		if errors.Is(err, service.ErrCannotLeaveAsOwner) {
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "owner cannot leave, transfer ownership first"},
			)

			return
		}

		if errors.Is(err, service.ErrNotGuildMember) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not a member of this guild"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left guild successfully"})
}

// KickMember 踢出成員
//
//	@Summary		踢出成員
//	@Description	擁有者踢出社群成員
//	@Tags			GuildMember
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int	true	"社群 ID"
//	@Param			userId	path		int	true	"使用者 ID"
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id}/members/{userId} [delete]
func (h *GuildHandler) KickMember(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	operatorUserID := c.GetUint("user_id")

	err = h.guildMemberService.KickMember(uint(guildID), uint(targetUserID), operatorUserID)
	if err != nil {
		if errors.Is(err, service.ErrNotGuildOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can kick members"})
			return
		}

		if errors.Is(err, service.ErrNotGuildMember) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user is not a member of this guild"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member kicked successfully"})
}

// ListGuildMembers 列出社群成員
//
//	@Summary		列出社群成員
//	@Description	列出社群的所有成員
//	@Tags			GuildMember
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"社群 ID"
//	@Success		200	{array}		model.GuildMember
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id}/members [get]
func (h *GuildHandler) ListGuildMembers(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	members, err := h.guildMemberService.ListGuildMembers(uint(guildID))
	if err != nil {
		if errors.Is(err, service.ErrGuildNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "guild not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, members)
}

// UpdateMemberRole 更新成員角色
//
//	@Summary		更新成員角色
//	@Description	擁有者更新成員角色
//	@Tags			GuildMember
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"社群 ID"
//	@Param			userId	path		int						true	"使用者 ID"
//	@Param			request	body		UpdateMemberRoleRequest	true	"更新角色請求"
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Router			/api/v1/guilds/{id}/members/{userId}/role [put]
func (h *GuildHandler) UpdateMemberRole(c *gin.Context) {
	guildID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorUserID := c.GetUint("user_id")

	err = h.guildMemberService.UpdateMemberRole(
		uint(guildID),
		uint(targetUserID),
		operatorUserID,
		req.Role,
	)
	if err != nil {
		if errors.Is(err, service.ErrNotGuildOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can update member roles"})
			return
		}

		if errors.Is(err, service.ErrNotGuildMember) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user is not a member of this guild"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member role updated successfully"})
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin moderator member"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}
