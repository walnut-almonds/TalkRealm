package handler

import (
	"errors"
	"net/http"
	"strconv"

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
	Count  int    `json:"count"`  // 答案數（少3/中6/多9）；0 = 該模式預設
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

	lv, err := h.learnService.CreateLevel(userID, req.Mode, req.Tier, req.Count, req.Locale)
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
	Count  int    `json:"count"` // 答案數（少3/中6/多9）；0 = 預設
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

	cw, err := h.learnService.CreateCrosswordLevel(userID, req.Tier, req.Count, req.Locale)
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

// GetCampaign 固定關卡總覽（含個人進度）
//
//	@Summary	取得固定關卡總覽
//	@Tags		Learn
//	@Produce	json
//	@Success	200	{object}	service.CampaignOverviewView
//	@Router		/api/v1/learn/campaign [get]
func (h *LearnHandler) GetCampaign(c *gin.Context) {
	userID := c.GetUint("user_id")

	ov, err := h.learnService.CampaignOverview(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ov)
}

type startCampaignReq struct {
	Locale string `json:"locale"`
}

// StartCampaign 開始固定關卡
//
//	@Summary	開始固定關卡
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		no		path		int					true	"關卡編號"
//	@Param		request	body		startCampaignReq	true	"語系"
//	@Success	200		{object}	service.CrosswordView
//	@Failure	403		{object}	ErrorResponse
//	@Failure	404		{object}	ErrorResponse
//	@Router		/api/v1/learn/campaign/{no} [post]
func (h *LearnHandler) StartCampaign(c *gin.Context) {
	userID := c.GetUint("user_id")

	no, err := strconv.Atoi(c.Param("no"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid level number"})
		return
	}

	var req startCampaignReq

	_ = c.ShouldBindJSON(&req) // body 只有選填 locale，缺 body 也可

	view, err := h.learnService.StartCampaignLevel(userID, no, req.Locale)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLearnLevelNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "campaign level not found"})
		case errors.Is(err, service.ErrLearnCampaignLocked):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, view)
}

// GetCampaignLeaderboard 關卡榜（?scope=friends 限好友）
//
//	@Summary	取得固定關卡排行榜
//	@Tags		Learn
//	@Produce	json
//	@Param		scope	query		string	false	"friends 表示好友榜"
//	@Success	200		{object}	service.LeaderboardView
//	@Router		/api/v1/learn/campaign/leaderboard [get]
func (h *LearnHandler) GetCampaignLeaderboard(c *gin.Context) {
	userID := c.GetUint("user_id")

	lb, err := h.learnService.CampaignLeaderboard(userID, c.Query("scope") == "friends")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lb)
}

// GetWeeklyLeaderboard 本週 XP 成長榜（?scope=friends 限好友）
//
//	@Summary	取得本週 XP 排行榜
//	@Tags		Learn
//	@Produce	json
//	@Param		scope	query		string	false	"friends 表示好友榜"
//	@Success	200		{object}	service.LeaderboardView
//	@Router		/api/v1/learn/leaderboard/weekly [get]
func (h *LearnHandler) GetWeeklyLeaderboard(c *gin.Context) {
	userID := c.GetUint("user_id")

	lb, err := h.learnService.WeeklyLeaderboard(userID, c.Query("scope") == "friends")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lb)
}

// GetSRSOverview 複習概況（今日到期 + 可學新字）
//
//	@Summary	取得 SRS 複習概況
//	@Tags		Learn
//	@Produce	json
//	@Success	200	{object}	service.SRSOverviewView
//	@Router		/api/v1/learn/srs/overview [get]
func (h *LearnHandler) GetSRSOverview(c *gin.Context) {
	userID := c.GetUint("user_id")

	ov, err := h.learnService.SRSOverview(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ov)
}

type startSRSReq struct {
	Count  int    `json:"count"`
	Locale string `json:"locale"`
}

// StartSRS 開始一場例句填空複習
//
//	@Summary	開始 SRS 複習 session
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		request	body		startSRSReq	true	"題數與語系"
//	@Success	200		{object}	service.SRSSessionView
//	@Failure	404		{object}	ErrorResponse
//	@Router		/api/v1/learn/srs [post]
func (h *LearnHandler) StartSRS(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req startSRSReq

	_ = c.ShouldBindJSON(&req) // 全選填，缺 body 也可

	view, err := h.learnService.CreateSRSSession(userID, req.Count, req.Locale)
	if err != nil {
		if errors.Is(err, service.ErrLearnNoSentences) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, view)
}

type srsAnswerReq struct {
	Index int    `json:"index"`
	Guess string `json:"guess"`
}

// AnswerSRS 作答一張複習卡
//
//	@Summary	作答 SRS 複習卡
//	@Tags		Learn
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"session ID"
//	@Param		request	body		srsAnswerReq	true	"卡片索引與作答"
//	@Success	200		{object}	service.SRSAnswerOutcome
//	@Failure	409		{object}	ErrorResponse
//	@Failure	410		{object}	ErrorResponse
//	@Router		/api/v1/learn/srs/{id}/answer [post]
func (h *LearnHandler) AnswerSRS(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req srsAnswerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.learnService.AnswerSRS(userID, c.Param("id"), req.Index, req.Guess)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLearnLevelNotFound):
			c.JSON(http.StatusGone, gin.H{"error": "session expired"})
		case errors.Is(err, service.ErrLearnCardGraded):
			c.JSON(http.StatusConflict, gin.H{"error": "card already answered"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	c.JSON(http.StatusOK, out)
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
