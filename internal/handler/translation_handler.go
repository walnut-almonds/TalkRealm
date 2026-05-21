package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// TranslationHandler 翻譯與猜測遊戲處理器
type TranslationHandler struct {
	translationService service.TranslationService
	guessService       service.GuessService
}

// NewTranslationHandler 建立翻譯處理器
func NewTranslationHandler(ts service.TranslationService, gs service.GuessService) *TranslationHandler {
	return &TranslationHandler{
		translationService: ts,
		guessService:       gs,
	}
}

// GetTranslation 取得訊息翻譯結果
//
//	@Summary		取得訊息翻譯
//	@Description	取得指定訊息的翻譯結果（三語）
//	@Tags			translation
//	@Produce		json
//	@Param			id	path		int	true	"訊息 ID"
//	@Success		200	{object}	model.MessageTranslation
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/messages/{id}/translation [get]
func (h *TranslationHandler) GetTranslation(c *gin.Context) {
	messageID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	translation, err := h.translationService.GetTranslation(messageID)
	if err != nil {
		if errors.Is(err, service.ErrTranslationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "translation not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get translation"})

		return
	}

	c.JSON(http.StatusOK, translation)
}

// SubmitGuess 提交猜測
//
//	@Summary		提交猜測
//	@Description	對指定訊息提交語意猜測，呼叫 LLM 評估相似度
//	@Tags			translation
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"訊息 ID"
//	@Param			request	body		service.GuessRequest	true	"猜測請求"
//	@Success		200		{object}	service.GuessResult
//	@Failure		400		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/api/v1/messages/{id}/guess [post]
func (h *TranslationHandler) SubmitGuess(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	var req service.GuessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.guessService.SubmitGuess(messageID, userID.(uint), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAlreadyGuessed):
			c.JSON(http.StatusConflict, gin.H{"error": "already guessed this message"})
		case errors.Is(err, service.ErrInvalidHiddenLang):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrTranslationNeeded):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "translation not ready yet"})
		case errors.Is(err, service.ErrLLMDisabled):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "guess service not configured"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to evaluate guess"})
		}

		return
	}

	c.JSON(http.StatusOK, result)
}

// GetGameStatus 取得猜測狀態
//
//	@Summary		取得猜測狀態
//	@Description	取得目前使用者對指定訊息的猜測狀態
//	@Tags			translation
//	@Produce		json
//	@Param			id			path		int		true	"訊息 ID"
//	@Param			hidden_lang	query		string	true	"隱藏語言 (zh/ja/en)"
//	@Success		200			{object}	service.GameStatus
//	@Router			/api/v1/messages/{id}/game [get]
func (h *TranslationHandler) GetGameStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	hiddenLang := c.Query("hidden_lang")
	if hiddenLang == "" {
		hiddenLang = "zh" // default
	}

	status, err := h.guessService.GetGameStatus(messageID, userID.(uint), hiddenLang)
	if err != nil {
		if errors.Is(err, service.ErrInvalidHiddenLang) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get game status"})

		return
	}

	c.JSON(http.StatusOK, status)
}

// parseUintParam 解析 URL path 參數為 uint
func parseUintParam(c *gin.Context, name string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		return 0, err
	}

	return uint(val), nil
}
