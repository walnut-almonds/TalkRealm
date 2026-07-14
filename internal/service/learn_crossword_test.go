//nolint:testpackage // 白箱測試：需存取未匯出的排版演算法
package service

import "testing"

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
