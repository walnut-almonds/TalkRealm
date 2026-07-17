//nolint:testpackage // 白箱測試：需存取未匯出的純函式（maskPositions/nextStreak/wordXP）
package service

import (
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestMaskPositions(t *testing.T) {
	for tier := 1; tier <= 5; tier++ {
		for wordLen := 3; wordLen <= 8; wordLen++ {
			masks := maskPositions(newLevelRand(), wordLen, tier)

			if len(masks) == 0 {
				t.Errorf("tier %d len %d: no masked positions", tier, wordLen)
			}

			if len(masks) >= wordLen {
				t.Errorf("tier %d len %d: all %d positions masked, must reveal >=1",
					tier, wordLen, len(masks))
			}

			seen := map[int]bool{}

			for _, p := range masks {
				if p < 0 || p >= wordLen {
					t.Errorf("position %d out of range [0,%d)", p, wordLen)
				}

				if seen[p] {
					t.Errorf("duplicate position %d", p)
				}

				seen[p] = true
			}
		}
	}

	// tier 5 遮蔽數應 >= tier 1
	r := newLevelRand()
	if len(maskPositions(r, 8, 5)) < len(maskPositions(r, 8, 1)) {
		t.Error("tier 5 should mask at least as many positions as tier 1")
	}
}

func TestNextStreak(t *testing.T) {
	tests := []struct {
		name        string
		last, today string
		cur, want   int
	}{
		{"first ever", "", "2026-07-08", 0, 1},
		{"consecutive day", "2026-07-07", "2026-07-08", 3, 4},
		{"same day, unchanged", "2026-07-08", "2026-07-08", 3, 3},
		{"gap resets", "2026-07-05", "2026-07-08", 9, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextStreak(tt.last, tt.today, tt.cur); got != tt.want {
				t.Errorf("nextStreak(%q,%q,%d)=%d want %d",
					tt.last, tt.today, tt.cur, got, tt.want)
			}
		})
	}
}

func TestWordXP(t *testing.T) {
	// spec §5：答對一字得 字長×tier；fill 模式 ×1.5
	if got := wordXP("star", 3, "wheel"); got != 12 {
		t.Errorf("wheel xp = %d want 12", got)
	}

	if got := wordXP("star", 3, ModeFill); got != 18 {
		t.Errorf("fill xp = %d want 18", got)
	}
}

func TestDefinitionFor(t *testing.T) {
	w := &model.Word{
		DefinitionEN: "en", DefinitionZH: "zh",
		DefinitionZHTW: "zhtw", DefinitionJA: "",
	}

	if got := definitionFor(w, "zh-tw"); got != "zhtw" {
		t.Errorf("zh-tw got %q", got)
	}

	if got := definitionFor(w, "ja"); got != "en" { // ja 缺 → fallback en
		t.Errorf("ja fallback got %q", got)
	}

	if got := definitionFor(w, "zh"); got != "zh" {
		t.Errorf("zh got %q", got)
	}

	if got := definitionFor(w, ""); got != "en" {
		t.Errorf("default got %q", got)
	}
}

// --- fill 模式端對端（mock repo + memory store）---

type fakeLearnRepo struct {
	words    []*model.Word
	stats    map[uint]*model.LearnStat
	daily    map[string]*model.LearnDailyScore
	records  int
	campaign map[int]*model.LearnCampaignLevel
	progress map[string]*model.LearnCampaignProgress // "uid:no"
	weekly   map[string]*model.LearnWeeklyXP         // "uid:week"
}

func (f *fakeLearnRepo) RandomWordsByTier(tier, n int) ([]*model.Word, error) {
	if n > len(f.words) {
		n = len(f.words)
	}

	return f.words[:n], nil
}
func (f *fakeLearnRepo) WordsByIDs(ids []uint) ([]*model.Word, error) { return f.words, nil }

// AllWordsForIndex 忠實模擬正式環境的輕量 SELECT（只有 id/word/tier/frequency/length，
// 不含 phonetic/definition）：buildWheelLevel/buildCrosswordLevel 若誤用這份資料當最終答案，
// 音標/釋義就會是空字串，測試才抓得到。
func (f *fakeLearnRepo) AllWordsForIndex() ([]*model.Word, error) {
	lite := make([]*model.Word, len(f.words))
	for i, w := range f.words {
		lite[i] = &model.Word{
			ID:        w.ID,
			Word:      w.Word,
			Tier:      w.Tier,
			Frequency: w.Frequency,
			Length:    w.Length,
		}
	}

	return lite, nil
}

func (f *fakeLearnRepo) GetOrCreateStats(userID uint) (*model.LearnStat, error) {
	if f.stats == nil {
		f.stats = map[uint]*model.LearnStat{}
	}

	if _, ok := f.stats[userID]; !ok {
		f.stats[userID] = &model.LearnStat{UserID: userID}
	}

	return f.stats[userID], nil
}
func (f *fakeLearnRepo) SaveStats(s *model.LearnStat) error { f.stats[s.UserID] = s; return nil }
func (f *fakeLearnRepo) UpsertWordRecord(userID, wordID uint, correct bool) error {
	f.records++
	return nil
}

func (f *fakeLearnRepo) CreateDailyScore(
	s *model.LearnDailyScore,
) (bool, error) {
	if f.daily == nil {
		f.daily = map[string]*model.LearnDailyScore{}
	}

	key := fmt.Sprintf("%d:%s", s.UserID, s.Date)
	if _, ok := f.daily[key]; ok {
		return false, nil
	}

	f.daily[key] = s

	return true, nil
}

func (f *fakeLearnRepo) TopDailyScores(date string, limit int) ([]*model.LearnDailyScore, error) {
	return nil, nil
}

func (f *fakeLearnRepo) UserDailyRank(
	userID uint,
	date string,
) (*model.LearnDailyScore, int, error) {
	if sc, ok := f.daily[fmt.Sprintf("%d:%s", userID, date)]; ok {
		return sc, 1, nil
	}

	return nil, 0, nil
}

func (f *fakeLearnRepo) CampaignLevelNos() ([]int, error) {
	nos := make([]int, 0, len(f.campaign))
	for no := range f.campaign {
		nos = append(nos, no)
	}

	sort.Ints(nos)

	return nos, nil
}

func (f *fakeLearnRepo) CreateCampaignLevel(l *model.LearnCampaignLevel) error {
	if f.campaign == nil {
		f.campaign = map[int]*model.LearnCampaignLevel{}
	}

	f.campaign[l.LevelNo] = l

	return nil
}

//nolint:nilnil // 忠實模擬真實 repo：nil = 關卡不存在
func (f *fakeLearnRepo) CampaignLevelByNo(no int) (*model.LearnCampaignLevel, error) {
	if l, ok := f.campaign[no]; ok {
		return l, nil
	}

	return nil, nil
}

func (f *fakeLearnRepo) CampaignProgress(userID uint) ([]*model.LearnCampaignProgress, error) {
	var ps []*model.LearnCampaignProgress

	for _, p := range f.progress {
		if p.UserID == userID {
			ps = append(ps, p)
		}
	}

	sort.Slice(ps, func(i, j int) bool { return ps[i].LevelNo < ps[j].LevelNo })

	return ps, nil
}

func (f *fakeLearnRepo) CreateCampaignProgress(p *model.LearnCampaignProgress) (bool, error) {
	if f.progress == nil {
		f.progress = map[string]*model.LearnCampaignProgress{}
	}

	key := fmt.Sprintf("%d:%d", p.UserID, p.LevelNo)
	if _, ok := f.progress[key]; ok {
		return false, nil
	}

	f.progress[key] = p

	return true, nil
}

func (f *fakeLearnRepo) CampaignTotals(
	userIDs []uint,
	limit int,
) ([]*repository.CampaignTotal, error) {
	byUser := map[uint]*repository.CampaignTotal{}

	for _, p := range f.progress {
		if len(userIDs) > 0 && !containsUint(userIDs, p.UserID) {
			continue
		}

		tp := byUser[p.UserID]
		if tp == nil {
			tp = &repository.CampaignTotal{UserID: p.UserID}
			byUser[p.UserID] = tp
		}

		tp.Total += p.Score

		if p.LevelNo > tp.Furthest {
			tp.Furthest = p.LevelNo
		}
	}

	out := make([]*repository.CampaignTotal, 0, len(byUser))
	for _, tp := range byUser {
		out = append(out, tp)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}

		return out[i].Furthest > out[j].Furthest
	})

	if len(out) > limit {
		out = out[:limit]
	}

	return out, nil
}

func (f *fakeLearnRepo) CampaignRank(
	userID uint, userIDs []uint,
) (*repository.CampaignTotal, int, error) {
	all, err := f.CampaignTotals(userIDs, 1<<30)
	if err != nil {
		return nil, 0, err
	}

	for i, tp := range all {
		if tp.UserID == userID {
			return tp, i + 1, nil
		}
	}

	return nil, 0, nil
}

func (f *fakeLearnRepo) UpsertWeeklyXP(userID uint, week string, xp int) error {
	if f.weekly == nil {
		f.weekly = map[string]*model.LearnWeeklyXP{}
	}

	key := fmt.Sprintf("%d:%s", userID, week)
	if row, ok := f.weekly[key]; ok {
		row.XP += xp

		return nil
	}

	f.weekly[key] = &model.LearnWeeklyXP{UserID: userID, Week: week, XP: xp}

	return nil
}

func (f *fakeLearnRepo) TopWeeklyXP(
	week string, userIDs []uint, limit int,
) ([]*model.LearnWeeklyXP, error) {
	var rows []*model.LearnWeeklyXP

	for _, r := range f.weekly {
		if r.Week != week {
			continue
		}

		if len(userIDs) > 0 && !containsUint(userIDs, r.UserID) {
			continue
		}

		rows = append(rows, r)
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].XP > rows[j].XP })

	if len(rows) > limit {
		rows = rows[:limit]
	}

	return rows, nil
}

func (f *fakeLearnRepo) WeeklyRank(
	userID uint, week string, userIDs []uint,
) (*model.LearnWeeklyXP, int, error) {
	rows, err := f.TopWeeklyXP(week, userIDs, 1<<30)
	if err != nil {
		return nil, 0, err
	}

	for i, r := range rows {
		if r.UserID == userID {
			return r, i + 1, nil
		}
	}

	return nil, 0, nil
}

func containsUint(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}

	return false
}

func newTestService(words []*model.Word) (LearnService, *fakeLearnRepo) {
	repo := &fakeLearnRepo{words: words}
	svc := NewLearnService(repo, nil, nil, NewMemoryLevelStore())

	return svc, repo
}

func testWords() []*model.Word {
	return []*model.Word{
		{
			ID:             1,
			Word:           "star",
			Phonetic:       "stɑː",
			Tier:           2,
			DefinitionEN:   "a ball of gas",
			DefinitionZH:   "星",
			DefinitionZHTW: "星星",
		},
		{
			ID:             2,
			Word:           "moon",
			Phonetic:       "muːn",
			Tier:           2,
			DefinitionEN:   "natural satellite",
			DefinitionZH:   "月",
			DefinitionZHTW: "月亮",
		},
	}
}

// wheelAnagramWords 供 wheel 模式測試使用：彼此互為 anagram 子字，
// 與 testWords()（star/moon，無交集）不同，findWheelBase 才找得到謎面。
func wheelAnagramWords() []*model.Word {
	const rat = "rat"

	return []*model.Word{
		{
			ID:             1,
			Word:           "star",
			Tier:           2,
			Frequency:      100,
			DefinitionEN:   "gas ball",
			DefinitionZHTW: "星星",
		},
		{ID: 2, Word: rat, Tier: 2, Frequency: 200, DefinitionEN: "rodent", DefinitionZHTW: "老鼠"},
		{
			ID:             3,
			Word:           "art",
			Tier:           2,
			Frequency:      150,
			DefinitionEN:   "creative work",
			DefinitionZHTW: "藝術",
		},
		{
			ID:             4,
			Word:           "tar",
			Tier:           2,
			Frequency:      300,
			DefinitionEN:   "black goo",
			DefinitionZHTW: "焦油",
		},
	}
}

func TestFillLevelFlow(t *testing.T) {
	svc, repo := newTestService(testWords())

	lv, err := svc.CreateLevel(7, ModeFill, 2, "zh-tw")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	if lv.Mode != ModeFill || len(lv.Slots) != 2 {
		t.Fatalf("unexpected level view: %+v", lv)
	}

	for _, s := range lv.Slots {
		if s.Word != "" {
			t.Error("answer leaked in slot view")
		}

		if s.Definition == "" {
			t.Error("fill slot must include definition")
		}
	}

	// 答錯
	out, err := svc.Guess(7, lv.LevelID, &LearnGuessRequest{Slot: 0, Word: "wrong"})
	if err != nil {
		t.Fatalf("Guess wrong: %v", err)
	}

	if out.Correct || out.XPAwarded != 0 {
		t.Errorf("wrong guess outcome: %+v", out)
	}

	// 答對第 0 格（testWords 順序即 slot 順序）
	out, err = svc.Guess(7, lv.LevelID, &LearnGuessRequest{Slot: 0, Word: "STAR"})
	if err != nil {
		t.Fatalf("Guess correct: %v", err)
	}

	if !out.Correct || out.Word != "star" || out.Completed {
		t.Errorf("correct guess outcome: %+v", out)
	}

	if out.XPAwarded != wordXP("star", 2, ModeFill) {
		t.Errorf("xp = %d", out.XPAwarded)
	}

	// 重複作答已解格 → ErrLearnSlotSolved
	if _, err := svc.Guess(
		7,
		lv.LevelID,
		&LearnGuessRequest{Slot: 0, Word: "star"},
	); !errors.Is(
		err,
		ErrLearnSlotSolved,
	) {
		t.Errorf("expected ErrLearnSlotSolved, got %v", err)
	}

	// 答對最後一格 → completed + streak
	out, err = svc.Guess(7, lv.LevelID, &LearnGuessRequest{Slot: 1, Word: "moon"})
	if err != nil {
		t.Fatalf("Guess final: %v", err)
	}

	if !out.Completed || out.TotalXP == 0 {
		t.Errorf("final outcome: %+v", out)
	}

	if repo.stats[7].Streak != 1 || repo.stats[7].XP != out.TotalXP {
		t.Errorf("stats not updated: %+v", repo.stats[7])
	}

	if repo.records != 3 { // 已解格的重複作答被拒（409），不記錄；其餘 3 次都要記錄
		t.Errorf("records = %d want 3", repo.records)
	}
}

func TestGuessUnknownLevel(t *testing.T) {
	svc, _ := newTestService(testWords())

	if _, err := svc.Guess(
		7,
		"nope",
		&LearnGuessRequest{Word: "x"},
	); !errors.Is(
		err,
		ErrLearnLevelNotFound,
	) {
		t.Errorf("expected ErrLearnLevelNotFound, got %v", err)
	}
}

func TestCreateLevelValidation(t *testing.T) {
	svc, _ := newTestService(testWords())

	if _, err := svc.CreateLevel(7, "bogus", 2, "en"); !errors.Is(err, ErrLearnInvalidMode) {
		t.Errorf("mode: %v", err)
	}

	if _, err := svc.CreateLevel(7, ModeFill, 9, "en"); !errors.Is(err, ErrLearnInvalidTier) {
		t.Errorf("tier: %v", err)
	}
}

// --- 每日挑戰（Task 9）---

func TestDailyLevelSameForAllUsers(t *testing.T) {
	svc, _ := newTestService(testWords())

	d1, err := svc.DailyLevel(7, "en")
	if err != nil {
		t.Fatalf("DailyLevel u7: %v", err)
	}

	d2, err := svc.DailyLevel(8, "en")
	if err != nil {
		t.Fatalf("DailyLevel u8: %v", err)
	}

	if d1.Played || d2.Played {
		t.Fatal("fresh users should not be played")
	}

	// 兩人拿到不同 level instance、但同一組題目（masked 相同）
	if d1.Level.LevelID == d2.Level.LevelID {
		t.Error("instances must be per-user")
	}

	for i := range d1.Level.Slots {
		if d1.Level.Slots[i].Masked != d2.Level.Slots[i].Masked {
			t.Error("daily puzzle must be identical for all users")
		}
	}
}

func TestDailyCompletionRecordsScore(t *testing.T) {
	svc, repo := newTestService(testWords())
	_ = repo

	d, err := svc.DailyLevel(7, "en")
	if err != nil {
		t.Fatalf("DailyLevel: %v", err)
	}

	// 全部答對
	for i, w := range []string{"star", "moon"} {
		if _, err := svc.Guess(
			7,
			d.Level.LevelID,
			&LearnGuessRequest{Slot: i, Word: w},
		); err != nil {
			t.Fatalf("guess %d: %v", i, err)
		}
	}

	// 再取 daily → played=true
	d2, err := svc.DailyLevel(7, "en")
	if err != nil {
		t.Fatalf("DailyLevel after: %v", err)
	}

	if !d2.Played || d2.Level != nil {
		t.Errorf("after completion: %+v", d2)
	}
}

func TestDailyTimeBonus(t *testing.T) {
	if got := dailyTimeBonus(10 * time.Second); got != 290 {
		t.Errorf("bonus(10s) = %d want 290", got)
	}

	if got := dailyTimeBonus(400 * time.Second); got != 0 {
		t.Errorf("bonus(400s) = %d want 0", got)
	}
}

func TestMaskWithHint(t *testing.T) {
	if got := maskWithHint("star", -1); got != "____" {
		t.Errorf("no hint: got %q want ____", got)
	}

	if got := maskWithHint("star", 0); got != "s___" {
		t.Errorf("hint pos 0: got %q want s___", got)
	}

	if got := maskWithHint("star", 3); got != "___r" {
		t.Errorf("hint pos 3: got %q want ___r", got)
	}
}

func TestHintDiscount(t *testing.T) {
	tests := []struct {
		xp, tier, want int
	}{
		{12, 0, 12},
		{12, 1, 9},
		{12, 2, 3},
		{12, 3, 12}, // 未定義階段當作無折扣
	}

	for _, tt := range tests {
		if got := hintDiscount(tt.xp, tt.tier); got != tt.want {
			t.Errorf("hintDiscount(%d,%d) = %d want %d", tt.xp, tt.tier, got, tt.want)
		}
	}
}

func TestHintWheelProgression(t *testing.T) {
	svc, _ := newTestService(wheelAnagramWords())

	lv, err := svc.CreateLevel(7, ModeWheel, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	out, err := svc.Hint(7, lv.LevelID, 0)
	if err != nil {
		t.Fatalf("Hint tier1: %v", err)
	}

	if out.Tier != 1 || out.Definition != "" {
		t.Errorf("tier1 outcome: %+v", out)
	}

	revealed := 0

	for _, ch := range out.Masked {
		if ch != '_' {
			revealed++
		}
	}

	if revealed != 1 {
		t.Errorf("tier1 masked %q should reveal exactly 1 letter", out.Masked)
	}

	out2, err := svc.Hint(7, lv.LevelID, 0)
	if err != nil {
		t.Fatalf("Hint tier2: %v", err)
	}

	if out2.Tier != 2 || out2.Definition == "" {
		t.Errorf("tier2 outcome: %+v", out2)
	}

	out3, err := svc.Hint(7, lv.LevelID, 0)
	if err != nil {
		t.Fatalf("Hint tier2 repeat: %v", err)
	}

	if out3.Tier != 2 {
		t.Errorf("hint beyond tier2 should stay at 2: %+v", out3)
	}
}

func TestHintRejectsSolvedSlot(t *testing.T) {
	const rat = "rat"

	svc, _ := newTestService(wheelAnagramWords())

	lv, err := svc.CreateLevel(7, ModeWheel, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	out, err := svc.Guess(7, lv.LevelID, &LearnGuessRequest{Word: rat})
	if err != nil {
		t.Fatalf("Guess: %v", err)
	}

	if _, err := svc.Hint(7, lv.LevelID, out.Slot); !errors.Is(err, ErrLearnSlotSolved) {
		t.Errorf("expected ErrLearnSlotSolved, got %v", err)
	}
}

func TestHintRejectsFillMode(t *testing.T) {
	svc, _ := newTestService(testWords())

	lv, err := svc.CreateLevel(7, ModeFill, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	if _, err := svc.Hint(7, lv.LevelID, 0); !errors.Is(err, ErrLearnHintNotSupported) {
		t.Errorf("expected ErrLearnHintNotSupported, got %v", err)
	}
}

// TestGuessXPDiscountedByHintTier 用 wheelAnagramWords()（非計劃原文的 testWords()）：
// ModeWheel 需要彼此有共同字母的字組，testWords()（star/moon）會讓 CreateLevel 在執行期失敗。
// wheelAnagramWords() 底字為 star，picked 順序固定為 [art, rat, tar, star]（按長度、頻率排序），
// 故 slot 0 恆為 "art"。
func TestGuessXPDiscountedByHintTier(t *testing.T) {
	const art = "art"

	svc, _ := newTestService(wheelAnagramWords())

	lv, err := svc.CreateLevel(7, ModeWheel, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	base := wordXP(art, 2, ModeWheel) // 3*2=6

	if _, err := svc.Hint(7, lv.LevelID, 0); err != nil { // tier1
		t.Fatalf("Hint tier1: %v", err)
	}

	out, err := svc.Guess(7, lv.LevelID, &LearnGuessRequest{Word: art})
	if err != nil {
		t.Fatalf("Guess after hint1: %v", err)
	}

	want := hintDiscount(base, 1)
	if !out.Correct || out.XPAwarded != want {
		t.Errorf("XPAwarded = %d want %d (base=%d)", out.XPAwarded, want, base)
	}
}

func TestRevealWheel(t *testing.T) {
	svc, repo := newTestService(wheelAnagramWords())

	lv, err := svc.CreateLevel(7, ModeWheel, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	out, err := svc.Reveal(7, lv.LevelID, 0)
	if err != nil {
		t.Fatalf("Reveal: %v", err)
	}

	if !out.Correct || out.XPAwarded != 0 || out.Word == "" {
		t.Errorf("reveal outcome: %+v", out)
	}

	if repo.records != 0 {
		t.Errorf("reveal must not write learn_word_records, got %d", repo.records)
	}

	if _, err := svc.Reveal(7, lv.LevelID, 0); !errors.Is(err, ErrLearnSlotSolved) {
		t.Errorf("expected ErrLearnSlotSolved on repeat reveal, got %v", err)
	}
}

// TestRevealCompletesLevel 依 lv.Slots 動態算出格數（wheelAnagramWords() 產生 4 個 slot，
// 不是計劃原文假設的 2 個 star/moon slot），逐格 Reveal 到全解為止。
func TestRevealCompletesLevel(t *testing.T) {
	svc, repo := newTestService(wheelAnagramWords())

	lv, err := svc.CreateLevel(7, ModeWheel, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	var out *GuessOutcome

	for slot := range lv.Slots {
		out, err = svc.Reveal(7, lv.LevelID, slot)
		if err != nil {
			t.Fatalf("Reveal slot %d: %v", slot, err)
		}
	}

	if !out.Completed {
		t.Errorf("expected completed after revealing all slots: %+v", out)
	}

	if repo.stats[7].XP != 0 {
		t.Errorf("all-reveal level should award 0 total XP, got %d", repo.stats[7].XP)
	}
}
