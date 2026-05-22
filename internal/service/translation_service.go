package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"gorm.io/gorm"
)

const translationStatusCompleted = "completed"

var ErrTranslationNotFound = errors.New("translation not found")

// langEntry 定義單一目標語言的 DeepL 代碼、model 欄位存取函式與 WS event key。
// 新增語言只需在 targetLangs 加一筆，其餘邏輯（翻譯、stale 檢查、WS 推送）自動支援。
type langEntry struct {
	DeepLCode  string
	WSKey      string
	SetContent func(*model.MessageTranslation, string)
	GetContent func(*model.MessageTranslation) string
}

// targetLangs 是目前支援的翻譯目標語言清單，集中管理避免散落各處。
// 新增語言步驟：
//  1. 在 model.MessageTranslation 加欄位（如 ContentZHTW）
//  2. 加一筆 langEntry
//  3. 更新 normalizeLang（區分偵測到的來源語言）
//  4. 執行 DB migration
var targetLangs = []langEntry{
	{
		DeepLCode:  "ZH",
		WSKey:      "zh",
		SetContent: func(t *model.MessageTranslation, s string) { t.ContentZH = s },
		GetContent: func(t *model.MessageTranslation) string { return t.ContentZH },
	},
	{
		DeepLCode:  "ZH-HANT",
		WSKey:      "zh-tw",
		SetContent: func(t *model.MessageTranslation, s string) { t.ContentZHTW = s },
		GetContent: func(t *model.MessageTranslation) string { return t.ContentZHTW },
	},
	{
		DeepLCode:  "JA",
		WSKey:      "ja",
		SetContent: func(t *model.MessageTranslation, s string) { t.ContentJA = s },
		GetContent: func(t *model.MessageTranslation) string { return t.ContentJA },
	},
	{
		DeepLCode:  "EN-US",
		WSKey:      "en",
		SetContent: func(t *model.MessageTranslation, s string) { t.ContentEN = s },
		GetContent: func(t *model.MessageTranslation) string { return t.ContentEN },
	},
}

// hasAllLangs 檢查翻譯記錄是否已填入所有目前設定的語言欄位。
// completed 但某欄位為空，代表這是「舊記錄，缺少後來新增的語言」（stale）。
func hasAllLangs(t *model.MessageTranslation) bool {
	for _, entry := range targetLangs {
		if entry.GetContent(t) == "" {
			return false
		}
	}

	return true
}

// buildWSTranslations 從翻譯記錄組出 WS event 的 translations map。
func buildWSTranslations(t *model.MessageTranslation) map[string]string {
	m := make(map[string]string, len(targetLangs))
	for _, entry := range targetLangs {
		m[entry.WSKey] = entry.GetContent(t)
	}

	return m
}

// findLangEntry 依 WSKey 查找 langEntry；回傳 nil 代表不支援該語言。
func findLangEntry(wsKey string) *langEntry {
	for i := range targetLangs {
		if targetLangs[i].WSKey == wsKey {
			return &targetLangs[i]
		}
	}

	return nil
}

// TranslationService 翻譯服務介面
type TranslationService interface {
	// TranslateAndPush 非同步翻譯訊息後透過 WS 推送（供 MessageService 呼叫）
	// 僅在 cfg.AutoTranslate=true 時執行翻譯
	TranslateAndPush(messageID uint, content string, channelID uint)
	// RequestTranslation 按需翻譯（使用者點擊翻譯按鈕時呼叫）。
	// targetLang 為使用者偏好語言的 WS key（zh/zh-tw/ja/en）；空字串表示翻譯所有語言。
	// 若指定語言已翻譯完成則直接透過 WS 推送；否則只呼叫 DeepL 補齊該語言。
	RequestTranslation(messageID uint, content string, channelID uint, targetLang string) error
	// EnsureTranslation 單次呼叫「取得或觸發」翻譯：
	// - 指定語言已翻譯完成 → 回傳結果（前端免等 WS）
	// - 尚未翻譯 → 只非同步觸發該語言，回傳 nil（等候 WS translation_ready）
	// targetLang 為 WS key（zh/zh-tw/ja/en）；空字串表示翻譯所有語言。
	EnsureTranslation(
		messageID uint,
		content string,
		channelID uint,
		targetLang string,
	) (*model.MessageTranslation, error)
	// GetTranslation 取得已完成的翻譯結果
	GetTranslation(messageID uint) (*model.MessageTranslation, error)
	// SetWebSocketManager 注入 WS manager（由 server.go 組裝時呼叫）
	SetWebSocketManager(manager WebSocketManager)
}

type translationService struct {
	repo      repository.TranslationRepository
	cfg       *config.DeepLConfig
	wsManager WebSocketManager
	client    *http.Client
}

// NewTranslationService 建立翻譯服務
func NewTranslationService(
	repo repository.TranslationRepository,
	cfg *config.DeepLConfig,
) TranslationService {
	return &translationService{
		repo:   repo,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetWebSocketManager 設定 WS 管理器（由 server.go 在組裝時注入）
func (s *translationService) SetWebSocketManager(manager WebSocketManager) {
	s.wsManager = manager
}

// TranslateAndPush 非同步翻譯所有語言後寫 DB 並推 WS 事件
// 僅在 cfg.Enabled=true 且 cfg.AutoTranslate=true 時執行
func (s *translationService) TranslateAndPush(messageID uint, content string, channelID uint) {
	if !s.cfg.Enabled || !s.cfg.AutoTranslate {
		return
	}

	go s.asyncTranslate(messageID, content, channelID, nil) // nil = 翻譯全部語言
}

// RequestTranslation 按需翻譯（使用者點擊翻譯按鈕時呼叫）。
// targetLang 為使用者偏好語言的 WS key（zh/zh-tw/ja/en）；空字串表示翻譯所有語言。
// 若指定語言已翻譯完成則直接推 WS；否則只呼叫 DeepL 補齊該語言，節省 API 費用。
func (s *translationService) RequestTranslation(
	messageID uint,
	content string,
	channelID uint,
	targetLang string,
) error {
	if !s.cfg.Enabled {
		return errors.New("translation not enabled")
	}

	existing, err := s.repo.GetByMessageID(messageID)
	if err == nil {
		if targetLang != "" {
			entry := findLangEntry(targetLang)
			if entry != nil && entry.GetContent(existing) != "" {
				// 指定語言已翻譯，直接推送（含所有已翻譯語言）
				if s.wsManager != nil {
					s.wsManager.BroadcastToChannel(channelID, "translation_ready", map[string]any{
						"message_id":    messageID,
						"original_lang": existing.OriginalLang,
						"translations":  buildWSTranslations(existing),
					})
				}

				return nil
			}
		} else if existing.TranslationStatus == translationStatusCompleted && hasAllLangs(existing) {
			if s.wsManager != nil {
				s.wsManager.BroadcastToChannel(channelID, "translation_ready", map[string]any{
					"message_id":    messageID,
					"original_lang": existing.OriginalLang,
					"translations":  buildWSTranslations(existing),
				})
			}

			return nil
		}
	}

	// 計算需要翻譯的語言清單
	var langs []langEntry
	if targetLang != "" {
		if entry := findLangEntry(targetLang); entry != nil {
			langs = []langEntry{*entry}
		}
	}

	go s.asyncTranslate(messageID, content, channelID, langs) // nil / subset = 由 asyncTranslate 決定

	return nil
}

// EnsureTranslation 單次呼叫「取得或觸發」翻譯：
// - 指定語言已翻譯完成 → 直接回傳結果（前端免等 WS）
// - 尚未翻譯 → 只非同步觸發指定語言，回傳 nil（前端等候 WS translation_ready 事件）
// targetLang 為 WS key（zh/zh-tw/ja/en）；空字串表示翻譯所有語言。
func (s *translationService) EnsureTranslation(
	messageID uint, content string, channelID uint, targetLang string,
) (*model.MessageTranslation, error) {
	if !s.cfg.Enabled {
		return nil, errors.New("translation not enabled")
	}

	existing, err := s.repo.GetByMessageID(messageID)
	if err == nil {
		if targetLang != "" {
			entry := findLangEntry(targetLang)
			if entry != nil && entry.GetContent(existing) != "" {
				return existing, nil
			}
		} else if existing.TranslationStatus == translationStatusCompleted && hasAllLangs(existing) {
			return existing, nil
		}
	}

	// 計算需要翻譯的語言清單
	var langs []langEntry
	if targetLang != "" {
		if entry := findLangEntry(targetLang); entry != nil {
			langs = []langEntry{*entry}
		}
	}

	go s.asyncTranslate(messageID, content, channelID, langs)

	return nil, nil //nolint:nilnil
}

// asyncTranslate 實際翻譯邏輯，在 goroutine 內執行。
// langs：指定要翻譯的語言清單（nil = 翻譯所有 targetLangs）。
// 函式會先載入現有翻譯記錄，跳過已翻譯的語言，只呼叫 DeepL 補齊缺少的部分，
// 節省 API 費用並避免覆寫已有翻譯。
func (s *translationService) asyncTranslate(
	messageID uint,
	content string,
	channelID uint,
	langs []langEntry,
) {
	if langs == nil {
		langs = targetLangs
	}

	// 載入現有記錄；若不存在則建立空記錄
	t, err := s.repo.GetByMessageID(messageID)
	if err != nil {
		t = &model.MessageTranslation{
			MessageID:         messageID,
			TranslationStatus: "pending",
			TranslatedAt:      time.Now().UTC(),
		}
	}

	// 只翻譯尚未填入的語言
	toTranslate := make([]langEntry, 0, len(langs))
	for _, entry := range langs {
		if entry.GetContent(t) == "" {
			toTranslate = append(toTranslate, entry)
		}
	}

	if len(toTranslate) == 0 {
		// 所有請求的語言都已翻譯；若已完成則重新推送 WS（例如頁面重整後重新觸發）
		if t.TranslationStatus == translationStatusCompleted && s.wsManager != nil {
			s.wsManager.BroadcastToChannel(channelID, "translation_ready", map[string]any{
				"message_id":    messageID,
				"original_lang": t.OriginalLang,
				"translations":  buildWSTranslations(t),
			})
		}

		return
	}

	origLang := t.OriginalLang
	if origLang == "" {
		origLang = "unknown"
	}

	results := make(map[string]string, len(toTranslate))

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for _, entry := range toTranslate {
		wg.Add(1)

		entry := entry

		go func() {
			defer wg.Done()

			translated, sourceLang, err := s.callDeepL(content, entry.DeepLCode)
			if err != nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			results[entry.DeepLCode] = translated

			// DeepL 偵測到的來源語言（第一個非 EN-US 的結果更可靠，但任一個都行）
			if origLang == "unknown" {
				origLang = normalizeLang(sourceLang)
			}
		}()
	}

	wg.Wait()

	if len(results) == 0 {
		t.TranslationStatus = "failed"
	} else {
		for _, entry := range toTranslate {
			if v, ok := results[entry.DeepLCode]; ok {
				entry.SetContent(t, v)
			}
		}

		t.OriginalLang = origLang
		t.TranslationStatus = translationStatusCompleted
		t.TranslatedAt = time.Now().UTC()
	}

	if err := s.repo.Upsert(t); err != nil {
		return
	}

	if t.TranslationStatus == translationStatusCompleted && s.wsManager != nil {
		s.wsManager.BroadcastToChannel(channelID, "translation_ready", map[string]any{
			"message_id":    messageID,
			"original_lang": t.OriginalLang,
			"translations":  buildWSTranslations(t),
		})
	}
}

// GetTranslation 取得已完成的翻譯結果
func (s *translationService) GetTranslation(messageID uint) (*model.MessageTranslation, error) {
	t, err := s.repo.GetByMessageID(messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTranslationNotFound
		}

		return nil, err
	}

	return t, nil
}

// deepLRequest DeepL API 請求體
type deepLRequest struct {
	Text       []string `json:"text"`
	TargetLang string   `json:"target_lang"`
}

// deepLResponse DeepL API 回應體
type deepLResponse struct {
	Translations []struct {
		DetectedSourceLanguage string `json:"detected_source_language"`
		Text                   string `json:"text"`
	} `json:"translations"`
}

// callDeepL 呼叫 DeepL Free API 翻譯單語
func (s *translationService) callDeepL(
	text, targetLang string,
) (translated, sourceLang string, err error) {
	reqBody := deepLRequest{
		Text:       []string{text},
		TargetLang: targetLang,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	apiURL := s.cfg.APIURL + "/translate"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		apiURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "DeepL-Auth-Key "+s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)

		return "", "", fmt.Errorf("deepl api error %d: %s", resp.StatusCode, string(respBody))
	}

	var dlResp deepLResponse
	if err := json.NewDecoder(resp.Body).Decode(&dlResp); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	if len(dlResp.Translations) == 0 {
		return "", "", fmt.Errorf("empty translation response")
	}

	return dlResp.Translations[0].Text, dlResp.Translations[0].DetectedSourceLanguage, nil
}

// normalizeLang 將 DeepL 偵測到的來源語言代碼正規化為 WS/前端使用的 key。
// 需與 targetLangs 中的 WSKey 保持一致。
func normalizeLang(lang string) string {
	switch lang {
	case "ZH", "ZH-HANS":
		return "zh"
	case "ZH-HANT":
		return "zh-tw"
	case "JA":
		return "ja"
	default:
		return "en"
	}
}
