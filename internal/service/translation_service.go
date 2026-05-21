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

var ErrTranslationNotFound = errors.New("translation not found")

// TranslationService 翻譯服務介面
type TranslationService interface {
	// TranslateAndPush 非同步翻譯訊息後透過 WS 推送（供 MessageService 呼叫）
	TranslateAndPush(messageID uint, content string, channelID uint)
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

// TranslateAndPush 非同步翻譯所有三種語言後寫 DB 並推 WS 事件
func (s *translationService) TranslateAndPush(messageID uint, content string, channelID uint) {
	if !s.cfg.Enabled {
		return
	}

	go func() {
		t := &model.MessageTranslation{
			MessageID:         messageID,
			TranslationStatus: "pending",
			TranslatedAt:      time.Now().UTC(),
		}

		langs := []string{"ZH", "JA", "EN-US"}
		results := make(map[string]string, len(langs))
		origLang := "unknown"

		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, target := range langs {
			wg.Add(1)
			target := target

			go func() {
				defer wg.Done()

				translated, sourceLang, err := s.callDeepL(content, target)
				if err != nil {
					return
				}

				mu.Lock()
				defer mu.Unlock()
				results[target] = translated

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
			t.ContentZH = results["ZH"]
			t.ContentJA = results["JA"]
			t.ContentEN = results["EN-US"]
			t.OriginalLang = origLang
			t.TranslationStatus = "completed"
			t.TranslatedAt = time.Now().UTC()
		}

		if err := s.repo.Upsert(t); err != nil {
			return
		}

		if t.TranslationStatus == "completed" && s.wsManager != nil {
			s.wsManager.BroadcastToChannel(channelID, "translation_ready", map[string]any{
				"message_id":    messageID,
				"original_lang": t.OriginalLang,
				"translations": map[string]string{
					"zh": t.ContentZH,
					"ja": t.ContentJA,
					"en": t.ContentEN,
				},
			})
		}
	}()
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
func (s *translationService) callDeepL(text, targetLang string) (translated, sourceLang string, err error) {
	reqBody := deepLRequest{
		Text:       []string{text},
		TargetLang: targetLang,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	apiURL := s.cfg.APIURL + "/translate"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiURL, bytes.NewReader(body))

	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "DeepL-Auth-Key "+s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

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

// normalizeLang 將 DeepL 語言代碼正規化為 zh/ja/en
func normalizeLang(lang string) string {
	switch lang {
	case "ZH", "ZH-HANS", "ZH-HANT":
		return "zh"
	case "JA":
		return "ja"
	default:
		return "en"
	}
}
