package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

// InteractionHandler 互動統計處理器
type InteractionHandler struct {
	messageRepo     repository.MessageRepository
	guildMemberRepo repository.GuildMemberRepository
}

// NewInteractionHandler 建立互動統計處理器
func NewInteractionHandler(
	messageRepo repository.MessageRepository,
	guildMemberRepo repository.GuildMemberRepository,
) *InteractionHandler {
	return &InteractionHandler{
		messageRepo:     messageRepo,
		guildMemberRepo: guildMemberRepo,
	}
}

// InteractionStatItem 單一 guild 的互動統計
type InteractionStatItem struct {
	GuildID      uint  `json:"guild_id"`
	MessageCount int64 `json:"message_count"`
}

// GetInteractionStats 取得當前使用者在各 guild 的訊息互動統計
//
//	@Summary	互動強度統計
//	@Tags		users
//	@Produce	json
//	@Security	BearerAuth
//	@Param		days	query		int	false	"統計天數（預設 30）"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/api/v1/users/me/interaction-stats [get]
func (h *InteractionHandler) GetInteractionStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	days := 30

	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	uid := userID.(uint)
	since := time.Now().UTC().AddDate(0, 0, -days)

	guildIDs, err := h.guildMemberRepo.GetUserGuildIDs(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user guilds"})
		return
	}

	counts, err := h.messageRepo.CountByUserInGuildsSince(uid, guildIDs, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get interaction stats"})
		return
	}

	stats := make([]InteractionStatItem, 0, len(guildIDs))
	for _, gid := range guildIDs {
		stats = append(stats, InteractionStatItem{
			GuildID:      gid,
			MessageCount: counts[gid], // zero if not in map
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
		"days":  days,
	})
}
