package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// LearnHandler 處理單字學習 API
type LearnHandler struct {
	learnService service.LearnService
}

// NewLearnHandler 建立 LearnHandler
func NewLearnHandler(learnService service.LearnService) *LearnHandler {
	return &LearnHandler{learnService: learnService}
}

type createLevelReq struct {
	Mode   string `json:"mode"   binding:"required"`
	Tier   int    `json:"tier"   binding:"required"`
	Locale string `json:"locale"` // 前端 i18n locale；空值 fallback en
}

// CreateLevel 生成新關卡
//
//	@Summary	生成單字關卡
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		request	body		createLevelReq	true	"模式與難度"
//	@Success	200		{object}	service.LevelView
//	@Failure	400		{object}	ErrorResponse
//	@Router		/api/v1/learn/levels [post]
func (h *LearnHandler) CreateLevel(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req createLevelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lv, err := h.learnService.CreateLevel(userID, req.Mode, req.Tier, req.Locale)
	if err != nil {
		if errors.Is(err, service.ErrLearnInvalidMode) ||
			errors.Is(err, service.ErrLearnInvalidTier) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, lv)
}

type createCrosswordReq struct {
	Tier   int    `json:"tier"   binding:"required"`
	Locale string `json:"locale"`
}

// CreateCrossword 生成交叉字謎網格關卡
//
//	@Summary	生成交叉字謎網格關卡
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		request	body		createCrosswordReq	true	"難度"
//	@Success	200		{object}	service.CrosswordView
//	@Failure	400		{object}	ErrorResponse
//	@Router		/api/v1/learn/levels/crossword [post]
func (h *LearnHandler) CreateCrossword(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req createCrosswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cw, err := h.learnService.CreateCrosswordLevel(userID, req.Tier, req.Locale)
	if err != nil {
		if errors.Is(err, service.ErrLearnInvalidTier) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, cw)
}

// Guess 作答
//
//	@Summary	提交單字作答
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"關卡 ID"
//	@Param		request	body		service.LearnGuessRequest	true	"作答內容"
//	@Success	200		{object}	service.GuessOutcome
//	@Failure	409		{object}	ErrorResponse
//	@Failure	410		{object}	ErrorResponse
//	@Router		/api/v1/learn/levels/{id}/guess [post]
func (h *LearnHandler) Guess(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.LearnGuessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.learnService.Guess(userID, c.Param("id"), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLearnLevelNotFound):
			c.JSON(http.StatusGone, gin.H{"error": "level expired"})
		case errors.Is(err, service.ErrLearnSlotSolved):
			c.JSON(http.StatusConflict, gin.H{"error": "slot already solved"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, out)
}

// GetDaily 今日挑戰
//
//	@Summary	取得每日挑戰
//	@Tags		Learn
//	@Produce	json
//	@Success	200	{object}	service.DailyView
//	@Router		/api/v1/learn/daily [get]
func (h *LearnHandler) GetDaily(c *gin.Context) {
	userID := c.GetUint("user_id")
	locale := c.Query("locale")

	d, err := h.learnService.DailyLevel(userID, locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, d)
}

// GetDailyLeaderboard 每日排行榜
//
//	@Summary	取得每日排行榜
//	@Tags		Learn
//	@Produce	json
//	@Success	200	{object}	service.LeaderboardView
//	@Router		/api/v1/learn/daily/leaderboard [get]
func (h *LearnHandler) GetDailyLeaderboard(c *gin.Context) {
	userID := c.GetUint("user_id")

	lb, err := h.learnService.DailyLeaderboard(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lb)
}

// GetStats 個人統計
//
//	@Summary	取得學習統計
//	@Tags		Learn
//	@Produce	json
//	@Success	200	{object}	service.LearnStatsView
//	@Router		/api/v1/learn/stats [get]
func (h *LearnHandler) GetStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := h.learnService.Stats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

type learnSlotReq struct {
	Slot int `json:"slot"`
}

// Hint 讓指定 slot 的提示前進一階
//
//	@Summary	單字提示
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"關卡 ID"
//	@Param		request	body		learnSlotReq	true	"slot 索引"
//	@Success	200		{object}	service.HintOutcome
//	@Failure	400		{object}	ErrorResponse
//	@Failure	409		{object}	ErrorResponse
//	@Failure	410		{object}	ErrorResponse
//	@Router		/api/v1/learn/levels/{id}/hint [post]
func (h *LearnHandler) Hint(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req learnSlotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.learnService.Hint(userID, c.Param("id"), req.Slot)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLearnLevelNotFound):
			c.JSON(http.StatusGone, gin.H{"error": "level expired"})
		case errors.Is(err, service.ErrLearnSlotSolved):
			c.JSON(http.StatusConflict, gin.H{"error": "slot already solved"})
		case errors.Is(err, service.ErrLearnHintNotSupported):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, out)
}

// Reveal 直接揭曉答案（0 XP）
//
//	@Summary	揭曉答案
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"關卡 ID"
//	@Param		request	body		learnSlotReq	true	"slot 索引"
//	@Success	200		{object}	service.GuessOutcome
//	@Failure	409		{object}	ErrorResponse
//	@Failure	410		{object}	ErrorResponse
//	@Router		/api/v1/learn/levels/{id}/reveal [post]
func (h *LearnHandler) Reveal(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req learnSlotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.learnService.Reveal(userID, c.Param("id"), req.Slot)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLearnLevelNotFound):
			c.JSON(http.StatusGone, gin.H{"error": "level expired"})
		case errors.Is(err, service.ErrLearnSlotSolved):
			c.JSON(http.StatusConflict, gin.H{"error": "slot already solved"})
		case errors.Is(err, service.ErrLearnHintNotSupported):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, out)
}
