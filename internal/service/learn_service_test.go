//nolint:testpackage // 白箱測試：需存取未匯出的純函式（maskPositions/nextStreak/wordXP）
package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
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
	words   []*model.Word
	stats   map[uint]*model.LearnStat
	daily   map[string]*model.LearnDailyScore
	records int
}

func (f *fakeLearnRepo) RandomWordsByTier(tier, n int) ([]*model.Word, error) {
	if n > len(f.words) {
		n = len(f.words)
	}

	return f.words[:n], nil
}
func (f *fakeLearnRepo) WordsByIDs(ids []uint) ([]*model.Word, error) { return f.words, nil }
func (f *fakeLearnRepo) AllWordsForIndex() ([]*model.Word, error)     { return f.words, nil }
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

func newTestService(words []*model.Word) (LearnService, *fakeLearnRepo) {
	repo := &fakeLearnRepo{words: words}
	svc := NewLearnService(repo, nil, NewMemoryLevelStore())

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
