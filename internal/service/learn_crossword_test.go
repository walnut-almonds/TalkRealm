//nolint:testpackage // 白箱測試：需存取未匯出的排版演算法
package service

import (
	"errors"
	"testing"

	"github.com/walnut-almonds/talkrealm/internal/model"
)

func TestLayoutCrosswordNoLetterConflicts(t *testing.T) {
	words := []string{"star", "art", "tar", "rat"}
	placements, rows, cols := layoutCrossword(words)

	grid := map[[2]int]byte{}
	placedCount := 0

	for i, p := range placements {
		if p.Row == -1 {
			continue
		}

		placedCount++
		w := words[i]

		for k := 0; k < len(w); k++ {
			r, c := p.Row, p.Col
			if p.Dir == "h" {
				c += k
			} else {
				r += k
			}

			if r < 0 || r >= rows || c < 0 || c >= cols {
				t.Fatalf("cell (%d,%d) out of bounds [0,%d)x[0,%d)", r, c, rows, cols)
			}

			if existing, ok := grid[[2]int{r, c}]; ok && existing != w[k] {
				t.Fatalf("conflict at (%d,%d): %q vs %q", r, c, string(existing), string(w[k]))
			}

			grid[[2]int{r, c}] = w[k]
		}
	}

	if placedCount == 0 {
		t.Fatal("expected at least one word placed")
	}
}

func TestLayoutCrosswordPerpendicularCrossings(t *testing.T) {
	// "cat" 與 "car" 共用字首 "ca"，應能交叉
	words := []string{"cat", "car"}
	placements, _, _ := layoutCrossword(words)

	if placements[0].Row == -1 || placements[1].Row == -1 {
		t.Fatalf("expected both words placed: %+v", placements)
	}

	if placements[0].Dir == placements[1].Dir {
		t.Errorf(
			"crossing words must have perpendicular directions, got both %q",
			placements[0].Dir,
		)
	}
}

func TestLayoutCrosswordUnrelatedWordBecomesBonus(t *testing.T) {
	// "cat" 與 "xyz" 沒有任何共同字母，不可能交叉
	words := []string{"cat", "xyz"}
	placements, _, _ := layoutCrossword(words)

	placedCount := 0

	for _, p := range placements {
		if p.Row != -1 {
			placedCount++
		}
	}

	if placedCount != 1 {
		t.Errorf("expected exactly 1 word placed (no shared letters), got %d", placedCount)
	}
}

func TestLayoutCrosswordNormalizedOrigin(t *testing.T) {
	words := []string{"star", "art", "rat", "tar"}
	placements, rows, cols := layoutCrossword(words)

	if rows <= 0 || cols <= 0 {
		t.Fatalf("rows=%d cols=%d must be positive", rows, cols)
	}

	touchesOrigin := false

	for _, p := range placements {
		if p.Row == -1 {
			continue
		}

		if p.Row < 0 || p.Col < 0 {
			t.Errorf("negative coordinate after normalization: %+v", p)
		}

		if p.Row == 0 || p.Col == 0 {
			touchesOrigin = true
		}
	}

	if !touchesOrigin {
		t.Error("bounding box not normalized to touch origin")
	}
}

func TestLayoutCrosswordStepBudgetTerminates(t *testing.T) {
	// 8 個彼此幾乎不相交的字，逼搜尋走較多分支；只驗證不 panic、結果合法
	words := []string{
		"aabbccdd", "bbccddee", "ccddeeff", "ddeeffgg",
		"eeffgghh", "ffgghhii", "gghhiijj", "hhiijjkk",
	}

	placements, rows, cols := layoutCrossword(words)
	if rows < 0 || cols < 0 {
		t.Fatalf("invalid dims: %d x %d", rows, cols)
	}

	if len(placements) != len(words) {
		t.Fatalf("expected %d placements, got %d", len(words), len(placements))
	}
}

func TestLayoutCrosswordSingleWord(t *testing.T) {
	placements, rows, cols := layoutCrossword([]string{"star"})

	if placements[0].Row != 0 || placements[0].Col != 0 || placements[0].Dir != "h" {
		t.Errorf("single word should anchor at (0,0,h): %+v", placements[0])
	}

	if rows != 1 || cols != 4 {
		t.Errorf("dims = %dx%d want 1x4", rows, cols)
	}
}

func TestCreateCrosswordLevel(t *testing.T) {
	words := []*model.Word{
		{
			ID: 1, Word: "star", Tier: 2, Frequency: 100,
			DefinitionEN: "gas ball", DefinitionZHTW: "星星",
		},
		{ID: 2, Word: "rat", Tier: 2, Frequency: 200, DefinitionEN: "rodent", DefinitionZHTW: "老鼠"},
		{
			ID: 3, Word: "art", Tier: 2, Frequency: 150,
			DefinitionEN: "creative work", DefinitionZHTW: "藝術",
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
	svc, _ := newTestService(words)

	cw, err := svc.CreateCrosswordLevel(7, 2, "zh-tw")
	if err != nil {
		t.Fatalf("CreateCrosswordLevel: %v", err)
	}

	if cw.LevelID == "" || cw.Rows == 0 || cw.Cols == 0 || len(cw.Words) < 2 {
		t.Fatalf("bad crossword view: %+v", cw)
	}

	placedCount := 0

	for _, w := range cw.Words {
		if w.Word != "" {
			t.Error("answer leaked before solve")
		}

		if w.Dir != "" {
			placedCount++
		}
	}

	if placedCount == 0 {
		t.Error("expected at least one word placed on the grid")
	}
}

func TestGuessCrossword(t *testing.T) {
	const rat = "rat"

	words := []*model.Word{
		{
			ID: 1, Word: "star", Tier: 2, Frequency: 100,
			DefinitionEN: "gas ball", DefinitionZHTW: "星星",
		},
		{ID: 2, Word: rat, Tier: 2, Frequency: 200, DefinitionEN: "rodent", DefinitionZHTW: "老鼠"},
		{
			ID: 3, Word: "art", Tier: 2, Frequency: 150,
			DefinitionEN: "creative work", DefinitionZHTW: "藝術",
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
	svc, repo := newTestService(words)

	cw, err := svc.CreateCrosswordLevel(7, 2, "zh-tw")
	if err != nil {
		t.Fatalf("CreateCrosswordLevel: %v", err)
	}

	out, err := svc.Guess(7, cw.LevelID, &LearnGuessRequest{Slot: -1, Word: rat})
	if err != nil {
		t.Fatalf("Guess: %v", err)
	}

	if !out.Correct || out.Word != rat || out.Definition == "" {
		t.Errorf("outcome: %+v", out)
	}

	if _, err := svc.Guess(
		7, cw.LevelID, &LearnGuessRequest{Slot: -1, Word: rat},
	); !errors.Is(err, ErrLearnSlotSolved) {
		t.Errorf("expected ErrLearnSlotSolved, got %v", err)
	}

	out, _ = svc.Guess(7, cw.LevelID, &LearnGuessRequest{Slot: -1, Word: "zzz"})
	if out == nil || out.Correct {
		t.Errorf("zzz should be wrong: %+v", out)
	}

	if repo.records != 1 { // 只有猜對 rat 那次要記錄；猜錯的字沒有對應 word_id
		t.Errorf("records = %d want 1", repo.records)
	}
}

func TestGuessCrosswordCompletion(t *testing.T) {
	words := []*model.Word{
		{
			ID:             1,
			Word:           "cats",
			Tier:           1,
			Frequency:      50,
			DefinitionEN:   "felines",
			DefinitionZHTW: "貓咪",
		},
		{ID: 2, Word: "cat", Tier: 1, Frequency: 10, DefinitionEN: "feline", DefinitionZHTW: "貓"},
	}
	svc, repo := newTestService(words)

	cw, err := svc.CreateCrosswordLevel(9, 1, "zh-tw")
	if err != nil {
		t.Fatalf("CreateCrosswordLevel: %v", err)
	}

	if len(cw.Words) != 2 {
		t.Fatalf("expected 2 words in crossword, got %d: %+v", len(cw.Words), cw.Words)
	}

	if _, err := svc.Guess(9, cw.LevelID, &LearnGuessRequest{Slot: -1, Word: "cat"}); err != nil {
		t.Fatalf("guess cat: %v", err)
	}

	out, err := svc.Guess(9, cw.LevelID, &LearnGuessRequest{Slot: -1, Word: "cats"})
	if err != nil {
		t.Fatalf("guess cats: %v", err)
	}

	if !out.Completed || out.TotalXP == 0 {
		t.Errorf("completion outcome: %+v", out)
	}

	if repo.stats[9].XP != out.TotalXP {
		t.Errorf("stats not updated: %+v", repo.stats[9])
	}
}

func TestHintCrosswordProgression(t *testing.T) {
	const rat = "rat"

	words := []*model.Word{
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
	svc, _ := newTestService(words)

	cw, err := svc.CreateCrosswordLevel(7, 2, "en")
	if err != nil {
		t.Fatalf("CreateCrosswordLevel: %v", err)
	}

	out, err := svc.Hint(7, cw.LevelID, 0)
	if err != nil {
		t.Fatalf("Hint tier1: %v", err)
	}

	if out.Tier != 1 {
		t.Errorf("tier1 outcome: %+v", out)
	}

	out2, err := svc.Hint(7, cw.LevelID, 0)
	if err != nil {
		t.Fatalf("Hint tier2: %v", err)
	}

	if out2.Tier != 2 || out2.Definition == "" {
		t.Errorf("tier2 outcome: %+v", out2)
	}
}

func TestFillWheelUnaffectedByEnvelope(t *testing.T) {
	// 信封改動的回歸驗證：fill 既有流程必須完全不受影響
	svc, _ := newTestService(testWords())

	lv, err := svc.CreateLevel(7, ModeFill, 2, "en")
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}

	out, err := svc.Guess(7, lv.LevelID, &LearnGuessRequest{Slot: 0, Word: "star"})
	if err != nil || !out.Correct {
		t.Fatalf("guess: %v, out=%+v", err, out)
	}
}
