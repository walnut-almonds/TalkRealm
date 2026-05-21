package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"gorm.io/gorm"
)

var (
	ErrGameStateNotFound = errors.New("game state not found")
	ErrAlreadyGuessed    = errors.New("already guessed this message")
	ErrInvalidHiddenLang = errors.New("invalid hidden_lang: must be zh, ja, or en")
	ErrTranslationNeeded = errors.New("translation not ready yet")
	ErrLLMDisabled       = errors.New("LLM service not configured")
)

// GuessRequest 猜測請求
type GuessRequest struct {
	GuessContent string `json:"guess_content" binding:"required"`
	HiddenLang   string `json:"hidden_lang"   binding:"required"` // zh, ja, en
}

// GuessResult 猜測結果
type GuessResult struct {
	IsCorrect       bool    `json:"is_correct"`
	SimilarityScore float64 `json:"similarity_score"`
	CorrectContent  string  `json:"correct_content,omitempty"` // 回傳正確答案（猜中或用戶已猜過）
}

// GameStatus 遊戲狀態回應
type GameStatus struct {
	HasGuessed      bool    `json:"has_guessed"`
	IsCorrect       bool    `json:"is_correct"`
	HiddenLang      string  `json:"hidden_lang"`
	SimilarityScore float64 `json:"similarity_score"`
}

// GuessService 猜測服務介面
type GuessService interface {
	SubmitGuess(messageID, guesserID uint, req *GuessRequest) (*GuessResult, error)
	GetGameStatus(messageID, guesserID uint, hiddenLang string) (*GameStatus, error)
}

type guessService struct {
	gameStateRepo   repository.GameStateRepository
	translationRepo repository.TranslationRepository
	cfg             *config.LLMConfig
	client          *http.Client
}

// NewGuessService 建立猜測服務
func NewGuessService(
	gameStateRepo repository.GameStateRepository,
	translationRepo repository.TranslationRepository,
	cfg *config.LLMConfig,
) GuessService {
	return &guessService{
		gameStateRepo:   gameStateRepo,
		translationRepo: translationRepo,
		cfg:             cfg,
		client:          &http.Client{Timeout: 30 * time.Second},
	}
}

// SubmitGuess 提交猜測，呼叫 LLM 評估語意相似度
func (s *guessService) SubmitGuess(messageID, guesserID uint, req *GuessRequest) (*GuessResult, error) {
	if !isValidLang(req.HiddenLang) {
		return nil, ErrInvalidHiddenLang
	}

	// 檢查是否已猜過
	existing, err := s.gameStateRepo.GetByMessageAndGuesser(messageID, guesserID, req.HiddenLang)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if existing != nil {
		return nil, ErrAlreadyGuessed
	}

	// 取得翻譯結果，擷取正確答案
	translation, err := s.translationRepo.GetByMessageID(messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTranslationNeeded
		}

		return nil, err
	}

	if translation.TranslationStatus != "completed" {
		return nil, ErrTranslationNeeded
	}

	correctContent := extractContent(translation, req.HiddenLang)

	if !s.cfg.Enabled {
		return nil, ErrLLMDisabled
	}

	// 呼叫 LLM 評估語意相似度
	score, err := s.evaluateSimilarity(correctContent, req.GuessContent, req.HiddenLang)
	if err != nil {
		return nil, fmt.Errorf("evaluate similarity: %w", err)
	}

	isCorrect := score >= s.cfg.SimilarityThreshold

	gs := &model.GameState{
		MessageID:       messageID,
		GuesserID:       guesserID,
		HiddenLang:      req.HiddenLang,
		GuessContent:    req.GuessContent,
		Mode:            "semantic",
		IsCorrect:       isCorrect,
		SimilarityScore: score,
		GuessedAt:       time.Now().UTC(),
	}

	if err := s.gameStateRepo.Create(gs); err != nil {
		return nil, err
	}

	result := &GuessResult{
		IsCorrect:       isCorrect,
		SimilarityScore: score,
	}

	if isCorrect {
		result.CorrectContent = correctContent
	}

	return result, nil
}

// GetGameStatus 取得猜測狀態
func (s *guessService) GetGameStatus(messageID, guesserID uint, hiddenLang string) (*GameStatus, error) {
	if !isValidLang(hiddenLang) {
		return nil, ErrInvalidHiddenLang
	}

	gs, err := s.gameStateRepo.GetByMessageAndGuesser(messageID, guesserID, hiddenLang)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &GameStatus{HasGuessed: false, HiddenLang: hiddenLang}, nil
		}

		return nil, err
	}

	return &GameStatus{
		HasGuessed:      true,
		IsCorrect:       gs.IsCorrect,
		HiddenLang:      gs.HiddenLang,
		SimilarityScore: gs.SimilarityScore,
	}, nil
}

// evaluateSimilarity 呼叫 LLM 評估兩段文字的語意相似度（回傳 0..1）
func (s *guessService) evaluateSimilarity(reference, guess, lang string) (float64, error) {
	prompt := buildSimilarityPrompt(reference, guess, lang)

	switch s.cfg.Provider {
	case "gemini":
		return s.callGemini(prompt)
	case "groq":
		return s.callGroq(prompt)
	default:
		return 0, fmt.Errorf("unsupported LLM provider: %s", s.cfg.Provider)
	}
}

// buildSimilarityPrompt 建立語意相似度評估 prompt
func buildSimilarityPrompt(reference, guess, lang string) string {
	langName := map[string]string{"zh": "Chinese", "ja": "Japanese", "en": "English"}[lang]
	if langName == "" {
		langName = lang
	}

	return fmt.Sprintf(
		`You are evaluating how semantically similar two %s sentences are.
Reference: "%s"
Guess: "%s"
Respond with ONLY a number between 0.00 and 1.00 representing similarity (1.00 = identical meaning, 0.00 = completely different). No explanation.`,
		langName, reference, guess,
	)
}

// --- Gemini ---

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (s *guessService) callGemini(prompt string) (float64, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	apiURL := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		s.cfg.Model, s.cfg.APIKey,
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		return 0, fmt.Errorf("gemini api error %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return 0, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return 0, fmt.Errorf("empty gemini response")
	}

	return parseScore(geminiResp.Candidates[0].Content.Parts[0].Text)
}

// --- Groq ---

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (s *guessService) callGroq(prompt string) (float64, error) {
	reqBody := groqRequest{
		Model:    s.cfg.Model,
		Messages: []groqMessage{{Role: "user", Content: prompt}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		return 0, fmt.Errorf("groq api error %d: %s", resp.StatusCode, string(respBody))
	}

	var groqResp groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return 0, err
	}

	if len(groqResp.Choices) == 0 {
		return 0, fmt.Errorf("empty groq response")
	}

	return parseScore(groqResp.Choices[0].Message.Content)
}

// parseScore 解析 LLM 回傳的分數字串（例如 "0.87"）
func parseScore(text string) (float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("empty score text")
	}

	var score float64

	if _, err := fmt.Sscanf(text, "%f", &score); err != nil {
		return 0, fmt.Errorf("parse score %q: %w", text, err)
	}

	if score < 0 {
		score = 0
	}

	if score > 1 {
		score = 1
	}

	return score, nil
}

// extractContent 從翻譯記錄中取出指定語言的內容
func extractContent(t *model.MessageTranslation, lang string) string {
	switch lang {
	case "zh":
		return t.ContentZH
	case "ja":
		return t.ContentJA
	default:
		return t.ContentEN
	}
}

// isValidLang 驗證語言代碼
func isValidLang(lang string) bool {
	return lang == "zh" || lang == "ja" || lang == "en"
}
