package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/pkg/config"
)

const langKeyZHTW = "zh-tw"

// ────────────────────────────────────────────────────────────────
// 語言設定（群組訊息 & DM 共用 DeepL 代碼 / WS key）
// 新增語言步驟：
//  1. 在 model.MessageTranslation / model.DMMessageTranslation 加欄位
//  2. 分別在 targetLangs / targetDMLangs 加一筆 entry
//  3. 更新 normalizeLang（區分偵測到的來源語言）
//  4. 執行 DB migration
// ────────────────────────────────────────────────────────────────

// langEntry 定義群組訊息的單一目標語言配置。
type langEntry struct {
	DeepLCode  string
	WSKey      string
	SetContent func(*model.MessageTranslation, string)
	GetContent func(*model.MessageTranslation) string
}

// targetLangs 群組訊息翻譯語言清單。
var targetLangs = []langEntry{
	{
		DeepLCode:  "ZH",
		WSKey:      "zh",
		SetContent: func(t *model.MessageTranslation, s string) { t.ContentZH = s },
		GetContent: func(t *model.MessageTranslation) string { return t.ContentZH },
	},
	{
		DeepLCode:  "ZH-HANT",
		WSKey:      langKeyZHTW,
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

// ────────────────────────────────────────────────────────────────
// 群組翻譯輔助函式
// ────────────────────────────────────────────────────────────────

// hasAllLangs 檢查群組翻譯記錄是否已填入所有語言欄位。
func hasAllLangs(t *model.MessageTranslation) bool {
	for _, e := range targetLangs {
		if e.GetContent(t) == "" {
			return false
		}
	}

	return true
}

// buildWSTranslations 從群組翻譯記錄組出 WS event 的 translations map。
func buildWSTranslations(t *model.MessageTranslation) map[string]string {
	m := make(map[string]string, len(targetLangs))
	for _, e := range targetLangs {
		m[e.WSKey] = e.GetContent(t)
	}

	return m
}

// findLangEntry 依 WSKey 查找群組語言設定；nil 表示不支援。
func findLangEntry(wsKey string) *langEntry {
	for i := range targetLangs {
		if targetLangs[i].WSKey == wsKey {
			return &targetLangs[i]
		}
	}

	return nil
}

// ────────────────────────────────────────────────────────────────
// 共用：語言正規化 & DeepL HTTP 呼叫
// ────────────────────────────────────────────────────────────────

// normalizeLang 將 DeepL 偵測到的來源語言代碼正規化為 WS/前端使用的 key。
// 群組訊息與 DM 共用此函式。
func normalizeLang(lang string) string {
	switch lang {
	case "ZH", "ZH-HANS":
		return "zh"
	case "ZH-HANT":
		return langKeyZHTW
	case "JA":
		return "ja"
	default:
		return "en"
	}
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

// callDeepL 呼叫 DeepL Free API 翻譯單語。群組訊息與 DM 共用此函式。
func callDeepL(
	client *http.Client,
	cfg *config.DeepLConfig,
	text, targetLang string,
) (translated, sourceLang string, err error) {
	body, err := json.Marshal(deepLRequest{
		Text:       []string{text},
		TargetLang: targetLang,
	})
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	apiURL := cfg.APIURL + "/translate"

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
	req.Header.Set("Authorization", "DeepL-Auth-Key "+cfg.APIKey)

	resp, err := client.Do(req)
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
