package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrLearnLevelNotFound = errors.New("learn level not found or expired")
	ErrLearnInvalidMode   = errors.New("invalid mode: must be fill or wheel")
	ErrLearnInvalidTier   = errors.New("invalid tier: must be 1..5")
	ErrLearnSlotSolved    = errors.New("slot already solved")
)

const (
	levelTTL      = 2 * time.Hour
	fillWordCount = 5
	levelKeyFmt   = "learn:level:%s"

	// ModeFill / ModeWheel 關卡模式
	ModeFill  = "fill"
	ModeWheel = "wheel"
)

// LearnUserLookup 提供排行榜顯示用的使用者查詢（模組邊界：不直接依賴 user repo 實作）
type LearnUserLookup interface {
	GetByID(id uint) (*model.User, error)
}

// LearnGuessRequest 作答請求
type LearnGuessRequest struct {
	Slot int    `json:"slot"`
	Word string `json:"word" binding:"required"`
}

// SlotView 單一答題格（下發 client，不含未解出的答案）
type SlotView struct {
	Masked     string `json:"masked"`
	Definition string `json:"definition,omitempty"`
	Length     int    `json:"length"`
	Solved     bool   `json:"solved"`
	Word       string `json:"word,omitempty"`
}

// LevelView 關卡謎面
type LevelView struct {
	LevelID string     `json:"level_id"`
	Mode    string     `json:"mode"`
	Tier    int        `json:"tier"`
	Slots   []SlotView `json:"slots"`
	Letters string     `json:"letters,omitempty"`
}

// GuessOutcome 作答結果
type GuessOutcome struct {
	Correct    bool   `json:"correct"`
	Slot       int    `json:"slot"`
	Word       string `json:"word,omitempty"`
	Phonetic   string `json:"phonetic,omitempty"`
	Definition string `json:"definition,omitempty"`
	XPAwarded  int    `json:"xp_awarded"`
	Completed  bool   `json:"completed"`
	TotalXP    int    `json:"total_xp,omitempty"`
}

// RecentWordView 最近作答的字
type RecentWordView struct {
	Word       string `json:"word"`
	Definition string `json:"definition"`
	Correct    bool   `json:"correct"`
}

// LearnStatsView 個人統計
type LearnStatsView struct {
	XP           int              `json:"xp"`
	Streak       int              `json:"streak"`
	WordsLearned int64            `json:"words_learned"`
	RecentWords  []RecentWordView `json:"recent_words"`
}

// LearnService 單字學習服務介面
type LearnService interface {
	CreateLevel(userID uint, mode string, tier int, locale string) (*LevelView, error)
	Guess(userID uint, levelID string, req *LearnGuessRequest) (*GuessOutcome, error)
	Stats(userID uint) (*LearnStatsView, error)
}

// learnLevel 進行中關卡（含答案，只存 LevelStore）
type learnLevel struct {
	ID        string    `json:"id"`
	UserID    uint      `json:"user_id"`
	Mode      string    `json:"mode"`
	Tier      int       `json:"tier"`
	Daily     string    `json:"daily,omitempty"`
	WordIDs   []uint    `json:"word_ids"`
	Words     []string  `json:"words"`
	Phonetics []string  `json:"phonetics"`
	Defs      []string  `json:"defs"`
	Masks     [][]int   `json:"masks"`
	Letters   string    `json:"letters"`
	Solved    []bool    `json:"solved"`
	XP        int       `json:"xp"`
	CreatedAt time.Time `json:"created_at"`
}

type learnService struct {
	repo  repository.LearnRepository
	users LearnUserLookup
	store LevelStore
}

// NewLearnService 建立單字學習服務
func NewLearnService(
	repo repository.LearnRepository,
	users LearnUserLookup,
	store LevelStore,
) LearnService {
	return &learnService{repo: repo, users: users, store: store}
}

// CreateLevel 生成新關卡
func (s *learnService) CreateLevel(
	userID uint, mode string, tier int, locale string,
) (*LevelView, error) {
	if mode != ModeFill && mode != ModeWheel {
		return nil, ErrLearnInvalidMode
	}

	if tier < 1 || tier > 5 {
		return nil, ErrLearnInvalidTier
	}

	if mode == ModeWheel {
		return nil, ErrLearnInvalidMode // ponytail: wheel 於 Task 7 實作
	}

	return s.createFillLevel(userID, tier, locale)
}

func (s *learnService) createFillLevel(userID uint, tier int, locale string) (*LevelView, error) {
	words, err := s.repo.RandomWordsByTier(tier, fillWordCount)
	if err != nil {
		return nil, err
	}

	if len(words) == 0 {
		return nil, fmt.Errorf("no words available for tier %d", tier)
	}

	rng := newLevelRand()
	lv := &learnLevel{
		ID:        uuid.NewString(),
		UserID:    userID,
		Mode:      ModeFill,
		Tier:      tier,
		CreatedAt: time.Now().UTC(),
	}

	for _, w := range words {
		lv.WordIDs = append(lv.WordIDs, w.ID)
		lv.Words = append(lv.Words, w.Word)
		lv.Phonetics = append(lv.Phonetics, w.Phonetic)
		lv.Defs = append(lv.Defs, definitionFor(w, locale))
		lv.Masks = append(lv.Masks, maskPositions(rng, len(w.Word), tier))
		lv.Solved = append(lv.Solved, false)
	}

	if err := s.saveLevel(lv); err != nil {
		return nil, err
	}

	return levelView(lv), nil
}

// Guess 作答一格
func (s *learnService) Guess(
	userID uint, levelID string, req *LearnGuessRequest,
) (*GuessOutcome, error) {
	lv, err := s.loadLevel(levelID)
	if err != nil {
		return nil, err
	}

	if lv == nil || lv.UserID != userID {
		return nil, ErrLearnLevelNotFound
	}

	slot := req.Slot
	if slot < 0 || slot >= len(lv.Words) {
		return nil, ErrLearnLevelNotFound
	}

	if lv.Solved[slot] {
		return nil, ErrLearnSlotSolved
	}

	guess := strings.ToLower(strings.TrimSpace(req.Word))
	correct := guess == lv.Words[slot]

	if err := s.repo.UpsertWordRecord(userID, lv.WordIDs[slot], correct); err != nil {
		return nil, err
	}

	out := &GuessOutcome{Correct: correct, Slot: slot}

	if !correct {
		return out, nil
	}

	lv.Solved[slot] = true
	out.Word = lv.Words[slot]
	out.Phonetic = lv.Phonetics[slot]
	out.Definition = lv.Defs[slot]
	out.XPAwarded = wordXP(lv.Words[slot], lv.Tier, lv.Mode)
	lv.XP += out.XPAwarded
	out.Completed = allSolved(lv.Solved)

	if out.Completed {
		out.TotalXP = lv.XP

		if err := s.onLevelCompleted(userID, lv); err != nil {
			return nil, err
		}
	}

	if err := s.saveLevel(lv); err != nil {
		return nil, err
	}

	return out, nil
}

// onLevelCompleted 完成關卡：更新 XP 與 streak（daily 計分於 Task 9 加入）
func (s *learnService) onLevelCompleted(userID uint, lv *learnLevel) error {
	stats, err := s.repo.GetOrCreateStats(userID)
	if err != nil {
		return err
	}

	today := time.Now().UTC().Format("2006-01-02")
	stats.Streak = nextStreak(stats.LastPlayedDate, today, stats.Streak)
	stats.LastPlayedDate = today
	stats.XP += lv.XP

	return s.repo.SaveStats(stats)
}

// Stats 個人統計
func (s *learnService) Stats(userID uint) (*LearnStatsView, error) {
	stats, err := s.repo.GetOrCreateStats(userID)
	if err != nil {
		return nil, err
	}

	return &LearnStatsView{
		XP:           stats.XP,
		Streak:       stats.Streak,
		WordsLearned: 0,                  // Task 5 wiring 後由 handler 顯示；v1 先由 recent 記錄推導
		RecentWords:  []RecentWordView{}, // ponytail: 先回空陣列，見 Task 6 前端顯示條件
	}, nil
}

// --- 純函式（可測性核心）---

func newLevelRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // 遊戲用途，非加密
}

// maskPositions 依 tier 決定遮蔽位置：tier1=40% … tier5=80%；至少遮 1、至少露 1
func maskPositions(rng *rand.Rand, wordLen, tier int) []int {
	ratio := 0.3 + float64(tier)*0.1 // tier1=0.4 … tier5=0.8
	n := int(float64(wordLen) * ratio)

	if n < 1 {
		n = 1
	}

	if n >= wordLen {
		n = wordLen - 1
	}

	positions := rng.Perm(wordLen)[:n]

	return positions
}

// nextStreak spec §5：昨天有玩 +1；同日不變；斷檔歸 1
func nextStreak(last, today string, cur int) int {
	if last == today {
		return cur
	}

	t, err := time.Parse("2006-01-02", today)
	if err != nil {
		return 1
	}

	if last == t.AddDate(0, 0, -1).Format("2006-01-02") {
		return cur + 1
	}

	return 1
}

// wordXP spec §5：字長×tier；fill 模式 ×1.5
func wordXP(word string, tier int, mode string) int {
	xp := len(word) * tier
	if mode == ModeFill {
		xp = xp * 3 / 2
	}

	return xp
}

// definitionFor 依 ui_locale 挑釋義；ja 缺字 fallback en（spec §3）
func definitionFor(w *model.Word, locale string) string {
	switch locale {
	case "zh":
		return w.DefinitionZH
	case langKeyZHTW:
		return w.DefinitionZHTW
	case "ja":
		if w.DefinitionJA != "" {
			return w.DefinitionJA
		}

		return w.DefinitionEN
	default:
		return w.DefinitionEN
	}
}

func allSolved(solved []bool) bool {
	for _, s := range solved {
		if !s {
			return false
		}
	}

	return true
}

// levelView 轉成下發 client 的謎面（未解格絕不含答案）
func levelView(lv *learnLevel) *LevelView {
	v := &LevelView{LevelID: lv.ID, Mode: lv.Mode, Tier: lv.Tier, Letters: lv.Letters}

	for i, word := range lv.Words {
		slot := SlotView{Length: len(word), Solved: lv.Solved[i]}

		if lv.Mode == ModeFill {
			slot.Definition = lv.Defs[i]
			slot.Masked = maskedWord(word, lv.Masks[i])
		} else {
			slot.Masked = strings.Repeat("_", len(word))
		}

		if lv.Solved[i] {
			slot.Word = word
			slot.Masked = word
			slot.Definition = lv.Defs[i]
		}

		v.Slots = append(v.Slots, slot)
	}

	return v
}

func maskedWord(word string, masks []int) string {
	b := []byte(word)
	for _, p := range masks {
		b[p] = '_'
	}

	return string(b)
}

// --- LevelStore helpers ---

func (s *learnService) saveLevel(lv *learnLevel) error {
	b, err := json.Marshal(lv)
	if err != nil {
		return err
	}

	return s.store.Set(fmt.Sprintf(levelKeyFmt, lv.ID), b, levelTTL)
}

func (s *learnService) loadLevel(id string) (*learnLevel, error) {
	b, err := s.store.Get(fmt.Sprintf(levelKeyFmt, id))
	if err != nil {
		return nil, err
	}

	if b == nil {
		return nil, ErrLearnLevelNotFound
	}

	var lv learnLevel
	if err := json.Unmarshal(b, &lv); err != nil {
		return nil, err
	}

	return &lv, nil
}
