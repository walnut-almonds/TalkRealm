package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
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

	wheelMaxAnswers = 8
	wheelMinAnswers = 3

	dateFmt       = "2006-01-02"
	dailyTier     = 3
	dailyKeyFmt   = "learn:daily:%s"
	dailyBonusCap = 300

	// ModeFill / ModeWheel / ModeCrossword 關卡模式
	ModeFill      = "fill"
	ModeWheel     = "wheel"
	ModeCrossword = "crossword"
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

// DailyView 每日挑戰狀態
type DailyView struct {
	Date   string     `json:"date"`
	Played bool       `json:"played"`
	Score  int        `json:"score,omitempty"` // played 時
	Level  *LevelView `json:"level,omitempty"` // 未 played 時
}

// LeaderboardEntry 排行榜單列
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Score    int    `json:"score"`
}

// LeaderboardView 當日排行榜
type LeaderboardView struct {
	Date string             `json:"date"`
	Top  []LeaderboardEntry `json:"top"`
	Me   *LeaderboardEntry  `json:"me,omitempty"` // 未完成時 nil
}

// dailyTemplate 每日題目模板（全站共用）
type dailyTemplate struct {
	WordIDs []uint  `json:"word_ids"`
	Masks   [][]int `json:"masks"`
}

// levelEnvelope 包住不同模式的關卡 payload，讓 saveLevel/loadLevel 依 Mode 分流解析。
// fill/wheel 用 learnLevel，crossword 用 crosswordLevel（各自獨立，互不干擾）。
type levelEnvelope struct {
	Mode string          `json:"mode"`
	Data json.RawMessage `json:"data"`
}

// saveEnvelope 把任意關卡 payload 包信封存進 LevelStore
func saveEnvelope(store LevelStore, id, mode string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	env := levelEnvelope{Mode: mode, Data: payload}

	b, err := json.Marshal(env)
	if err != nil {
		return err
	}

	return store.Set(fmt.Sprintf(levelKeyFmt, id), b, levelTTL)
}

// loadEnvelope 從 LevelStore 讀信封（尚未解析出實際型別）
func loadEnvelope(store LevelStore, id string) (*levelEnvelope, error) {
	b, err := store.Get(fmt.Sprintf(levelKeyFmt, id))
	if err != nil {
		return nil, err
	}

	if b == nil {
		return nil, ErrLearnLevelNotFound
	}

	var env levelEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

// LearnService 單字學習服務介面
type LearnService interface {
	CreateLevel(userID uint, mode string, tier int, locale string) (*LevelView, error)
	CreateCrosswordLevel(userID uint, tier int, locale string) (*CrosswordView, error)
	Guess(userID uint, levelID string, req *LearnGuessRequest) (*GuessOutcome, error)
	Stats(userID uint) (*LearnStatsView, error)
	DailyLevel(userID uint, locale string) (*DailyView, error)
	DailyLeaderboard(userID uint) (*LeaderboardView, error)
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

	idxOnce sync.Once
	idx     *anagramIndex
	idxErr  error
}

func (s *learnService) anagram() (*anagramIndex, error) {
	// ponytail: lazy + sync.Once，首個 wheel 請求付一次建索引成本（<1s），之後常駐
	s.idxOnce.Do(func() {
		words, err := s.repo.AllWordsForIndex()
		if err != nil {
			s.idxErr = err

			return
		}

		s.idx = buildAnagramIndex(words)
	})

	return s.idx, s.idxErr
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
		return s.createWheelLevel(userID, tier, locale)
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

func (s *learnService) createWheelLevel(userID uint, tier int, locale string) (*LevelView, error) {
	idx, err := s.anagram()
	if err != nil {
		return nil, err
	}

	candidates, err := s.repo.RandomWordsByTier(tier, 20)
	if err != nil {
		return nil, err
	}

	base, ids, ok := findWheelBase(candidates, idx)
	if !ok {
		return nil, fmt.Errorf("no wheel puzzle available for tier %d", tier)
	}

	return s.buildWheelLevel(userID, tier, locale, base, ids, idx)
}

// findWheelBase 依既有規則找底字候選與其子字 ID 清單：優先 5–7 字母且答案數
// >= wheelMinAnswers；找不到則放寬到 >=4 字母、答案數 >= 2。
// wheel 與 crossword 共用同一套底字挑選規則。
func findWheelBase(candidates []*model.Word, idx *anagramIndex) (*model.Word, []uint, bool) {
	for _, base := range candidates {
		if len(base.Word) < 5 || len(base.Word) > 7 {
			continue
		}

		ids := idx.subWordIDs(base.Word)
		if len(ids) >= wheelMinAnswers {
			return base, ids, true
		}
	}

	for _, base := range candidates {
		if len(base.Word) < 4 {
			continue
		}

		ids := idx.subWordIDs(base.Word)
		if len(ids) >= 2 {
			return base, ids, true
		}
	}

	return nil, nil, false
}

func (s *learnService) buildWheelLevel(
	userID uint, tier int, locale string,
	base *model.Word, ids []uint, idx *anagramIndex,
) (*LevelView, error) {
	answers := make([]*model.Word, 0, len(ids))
	for _, id := range ids {
		answers = append(answers, idx.words[id])
	}

	// 短字在前、常用字優先；上限 8 個，底字必收
	sort.Slice(answers, func(i, j int) bool {
		if len(answers[i].Word) != len(answers[j].Word) {
			return len(answers[i].Word) < len(answers[j].Word)
		}

		return answers[i].Frequency < answers[j].Frequency
	})

	picked := []*model.Word{}

	for _, a := range answers {
		if a.ID == base.ID {
			continue
		}

		if len(picked) < wheelMaxAnswers-1 {
			picked = append(picked, a)
		}
	}

	picked = append(picked, base) // 底字放最後（最長）

	rng := newLevelRand()
	letters := []byte(base.Word)
	rng.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })

	lv := &learnLevel{
		ID:        uuid.NewString(),
		UserID:    userID,
		Mode:      ModeWheel,
		Tier:      tier,
		Letters:   string(letters),
		CreatedAt: time.Now().UTC(),
	}

	for _, w := range picked {
		lv.WordIDs = append(lv.WordIDs, w.ID)
		lv.Words = append(lv.Words, w.Word)
		lv.Phonetics = append(lv.Phonetics, w.Phonetic)
		lv.Defs = append(lv.Defs, definitionFor(w, locale))
		lv.Masks = append(lv.Masks, nil)
		lv.Solved = append(lv.Solved, false)
	}

	if err := s.saveLevel(lv); err != nil {
		return nil, err
	}

	return levelView(lv), nil
}

// Guess 作答一格；依 Redis 信封的 Mode 分流到 fill/wheel 或 crossword 邏輯
func (s *learnService) Guess(
	userID uint, levelID string, req *LearnGuessRequest,
) (*GuessOutcome, error) {
	env, err := loadEnvelope(s.store, levelID)
	if err != nil {
		return nil, err
	}

	if env.Mode == ModeCrossword {
		return s.guessCrossword(userID, env, req)
	}

	return s.guessFillWheel(userID, env, req)
}

// guessFillWheel 處理 fill/wheel 的作答（既有邏輯，未改變任何行為）
func (s *learnService) guessFillWheel(
	userID uint, env *levelEnvelope, req *LearnGuessRequest,
) (*GuessOutcome, error) {
	var lv learnLevel
	if err := json.Unmarshal(env.Data, &lv); err != nil {
		return nil, err
	}

	if lv.UserID != userID {
		return nil, ErrLearnLevelNotFound
	}

	guess := strings.ToLower(strings.TrimSpace(req.Word))

	if lv.Mode == ModeWheel {
		slot := -1

		for i, w := range lv.Words {
			if w == guess {
				slot = i

				break
			}
		}

		if slot == -1 {
			// wheel 猜錯無對應 word_id，不寫 word record
			return &GuessOutcome{Correct: false, Slot: -1}, nil
		}

		req.Slot = slot // 收斂到共用路徑
	}

	slot := req.Slot
	if slot < 0 || slot >= len(lv.Words) {
		return nil, ErrLearnLevelNotFound
	}

	if lv.Solved[slot] {
		return nil, ErrLearnSlotSolved
	}

	correct := guess == lv.Words[slot]

	if lv.Mode == ModeFill || correct {
		if err := s.repo.UpsertWordRecord(userID, lv.WordIDs[slot], correct); err != nil {
			return nil, err
		}
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

		if err := s.onLevelCompleted(userID, lv.XP, lv.Daily, lv.CreatedAt); err != nil {
			return nil, err
		}
	}

	if err := s.saveLevel(&lv); err != nil {
		return nil, err
	}

	return out, nil
}

// onLevelCompleted 完成關卡：更新 XP 與 streak；daily 非空時另記當日分數。
// 簽名用原生型別（不吃 *learnLevel），fill/wheel/crossword 都能呼叫。
func (s *learnService) onLevelCompleted(
	userID uint,
	xp int,
	daily string,
	createdAt time.Time,
) error {
	stats, err := s.repo.GetOrCreateStats(userID)
	if err != nil {
		return err
	}

	today := time.Now().UTC().Format(dateFmt)
	stats.Streak = nextStreak(stats.LastPlayedDate, today, stats.Streak)
	stats.LastPlayedDate = today
	stats.XP += xp

	if err := s.repo.SaveStats(stats); err != nil {
		return err
	}

	if daily != "" {
		elapsed := time.Now().UTC().Sub(createdAt)

		// 唯一鍵擋重複；重玩不覆寫首次分數
		_, err := s.repo.CreateDailyScore(&model.LearnDailyScore{
			UserID:        userID,
			Date:          daily,
			Score:         xp + dailyTimeBonus(elapsed),
			CompletedInMs: elapsed.Milliseconds(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// dailyTimeBonus spec §5：300 秒內完成才有時間加成，逐秒遞減
func dailyTimeBonus(elapsed time.Duration) int {
	b := dailyBonusCap - int(elapsed.Seconds())
	if b < 0 {
		return 0
	}

	return b
}

// DailyLevel 取得今日挑戰：已完成回分數；未完成發個人關卡 instance
func (s *learnService) DailyLevel(userID uint, locale string) (*DailyView, error) {
	date := time.Now().UTC().Format(dateFmt)

	existing, _, err := s.repo.UserDailyRank(userID, date)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return &DailyView{Date: date, Played: true, Score: existing.Score}, nil
	}

	tpl, err := s.dailyTemplate(date)
	if err != nil {
		return nil, err
	}

	words, err := s.repo.WordsByIDs(tpl.WordIDs)
	if err != nil {
		return nil, err
	}

	// WordsByIDs 不保序，依模板順序重排
	byID := map[uint]*model.Word{}
	for _, w := range words {
		byID[w.ID] = w
	}

	lv := &learnLevel{
		ID:        uuid.NewString(),
		UserID:    userID,
		Mode:      ModeFill,
		Tier:      dailyTier,
		Daily:     date,
		CreatedAt: time.Now().UTC(),
	}

	for i, id := range tpl.WordIDs {
		w := byID[id]
		if w == nil {
			return nil, fmt.Errorf("daily word %d missing", id)
		}

		lv.WordIDs = append(lv.WordIDs, w.ID)
		lv.Words = append(lv.Words, w.Word)
		lv.Phonetics = append(lv.Phonetics, w.Phonetic)
		lv.Defs = append(lv.Defs, definitionFor(w, locale))
		lv.Masks = append(lv.Masks, tpl.Masks[i])
		lv.Solved = append(lv.Solved, false)
	}

	if err := s.saveLevel(lv); err != nil {
		return nil, err
	}

	return &DailyView{Date: date, Played: false, Level: levelView(lv)}, nil
}

// dailyTemplate 取得（或首次生成）當日題目模板；SetNX 保證全站同題
func (s *learnService) dailyTemplate(date string) (*dailyTemplate, error) {
	key := fmt.Sprintf(dailyKeyFmt, date)

	if b, err := s.store.Get(key); err != nil {
		return nil, err
	} else if b != nil {
		var tpl dailyTemplate
		if err := json.Unmarshal(b, &tpl); err != nil {
			return nil, err
		}

		return &tpl, nil
	}

	words, err := s.repo.RandomWordsByTier(dailyTier, fillWordCount)
	if err != nil {
		return nil, err
	}

	if len(words) == 0 {
		return nil, fmt.Errorf("no words for daily challenge")
	}

	rng := newLevelRand()
	tpl := &dailyTemplate{}

	for _, w := range words {
		tpl.WordIDs = append(tpl.WordIDs, w.ID)
		tpl.Masks = append(tpl.Masks, maskPositions(rng, len(w.Word), dailyTier))
	}

	b, err := json.Marshal(tpl)
	if err != nil {
		return nil, err
	}

	ok, err := s.store.SetNX(key, b, 26*time.Hour)
	if err != nil {
		return nil, err
	}

	if !ok { // 併發下別人先寫入：改用他的
		return s.dailyTemplate(date)
	}

	return tpl, nil
}

// DailyLeaderboard 當日排行榜（top 10 + 自己）
func (s *learnService) DailyLeaderboard(userID uint) (*LeaderboardView, error) {
	date := time.Now().UTC().Format(dateFmt)

	top, err := s.repo.TopDailyScores(date, 10)
	if err != nil {
		return nil, err
	}

	view := &LeaderboardView{Date: date}

	for i, sc := range top {
		view.Top = append(view.Top, s.leaderboardEntry(i+1, sc))
	}

	me, rank, err := s.repo.UserDailyRank(userID, date)
	if err != nil {
		return nil, err
	}

	if me != nil {
		e := s.leaderboardEntry(rank, me)
		view.Me = &e
	}

	return view, nil
}

func (s *learnService) leaderboardEntry(rank int, sc *model.LearnDailyScore) LeaderboardEntry {
	e := LeaderboardEntry{Rank: rank, UserID: sc.UserID, Score: sc.Score}

	// 模組邊界：顯示資訊走 UserLookup，查不到就只顯示 ID
	if s.users != nil {
		if u, err := s.users.GetByID(sc.UserID); err == nil {
			e.Username = displayName(u)
			e.Avatar = u.Avatar
		}
	}

	return e
}

func displayName(u *model.User) string {
	if u.Nickname != "" {
		return u.Nickname
	}

	return u.Username
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

	t, err := time.Parse(dateFmt, today)
	if err != nil {
		return 1
	}

	if last == t.AddDate(0, 0, -1).Format(dateFmt) {
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
	return saveEnvelope(s.store, lv.ID, lv.Mode, lv)
}
