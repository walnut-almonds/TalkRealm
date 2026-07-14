package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/walnut-almonds/talkrealm/internal/model"
)

// crosswordPlacement 單一字在網格中的座標與方向
type crosswordPlacement struct {
	Row, Col int
	Dir      string // "h" | "v"；未排進網格時 Row=-1,Col=-1,Dir=""
}

// cwCell 網格單一格子的佔用狀態：字母 + 該方向是否已被使用
// （同一格可以同時被一個水平字與一個垂直字佔用，這就是交叉點；
// 不能有兩個水平字或兩個垂直字重疊在同一格）
type cwCell struct {
	letter      byte
	horiz, vert bool
}

const cwStepBudget = 20000

// layoutCrossword 回溯搜尋，找出能讓最多字互相交叉的排版。
// ponytail: branch-and-bound + 步數上限保底，不保證數學最佳解，
// 但 N<=8 個短字的實務場景幾乎都能在上限內找到真正最佳排法。
func layoutCrossword(words []string) ([]crosswordPlacement, int, int) {
	n := len(words)

	best := make([]crosswordPlacement, n)
	for i := range best {
		best[i] = crosswordPlacement{Row: -1, Col: -1}
	}

	bestCount := -1
	steps := 0

	cur := make([]crosswordPlacement, n)
	for i := range cur {
		cur[i] = crosswordPlacement{Row: -1, Col: -1}
	}

	all := make([]int, n)
	for i := range all {
		all[i] = i
	}

	var search func(remaining []int, grid map[[2]int]cwCell, placedCount int)

	search = func(remaining []int, grid map[[2]int]cwCell, placedCount int) {
		steps++
		if steps > cwStepBudget {
			return
		}

		// branch-and-bound：樂觀上界都贏不了目前最佳解就剪掉
		if placedCount+len(remaining) <= bestCount {
			return
		}

		if placedCount > bestCount {
			bestCount = placedCount

			copy(best, cur)
		}

		for ri, idx := range remaining {
			candidates := candidatePlacements(words[idx], grid, len(grid) == 0)

			nextRemaining := make([]int, 0, len(remaining)-1)
			nextRemaining = append(nextRemaining, remaining[:ri]...)
			nextRemaining = append(nextRemaining, remaining[ri+1:]...)

			for _, cand := range candidates {
				newGrid := cloneGrid(grid)
				applyPlacement(newGrid, words[idx], cand)

				cur[idx] = cand

				search(nextRemaining, newGrid, placedCount+1)

				cur[idx] = crosswordPlacement{Row: -1, Col: -1}
			}
		}
	}

	search(all, map[[2]int]cwCell{}, 0)

	return normalizeCrossword(best, words)
}

// candidatePlacements 找出 word 在目前網格中所有合法的交叉位置。
// gridEmpty 時 word 是骨幹，固定放在 (0,0) 水平。
func candidatePlacements(word string, grid map[[2]int]cwCell, gridEmpty bool) []crosswordPlacement {
	if gridEmpty {
		return []crosswordPlacement{{Row: 0, Col: 0, Dir: "h"}}
	}

	seen := map[string]bool{}

	var out []crosswordPlacement

	tryAdd := func(row, col int, dir string) {
		key := fmt.Sprintf("%d,%d,%s", row, col, dir)
		if seen[key] {
			return
		}

		seen[key] = true

		if validPlacement(word, row, col, dir, grid) {
			out = append(out, crosswordPlacement{Row: row, Col: col, Dir: dir})
		}
	}

	for i := 0; i < len(word); i++ {
		ch := word[i]

		for coord, cell := range grid {
			if cell.letter != ch {
				continue
			}

			if cell.horiz && !cell.vert {
				tryAdd(coord[0]-i, coord[1], "v")
			}

			if cell.vert && !cell.horiz {
				tryAdd(coord[0], coord[1]-i, "h")
			}
		}
	}

	return out
}

// validPlacement 檢查 word 放在 (row,col,dir) 是否與現有格子衝突
func validPlacement(word string, row, col int, dir string, grid map[[2]int]cwCell) bool {
	for i := 0; i < len(word); i++ {
		r, c := row, col
		if dir == "h" {
			c += i
		} else {
			r += i
		}

		existing, ok := grid[[2]int{r, c}]
		if !ok {
			continue
		}

		if existing.letter != word[i] {
			return false
		}

		if dir == "h" && existing.horiz {
			return false
		}

		if dir == "v" && existing.vert {
			return false
		}
	}

	return true
}

func applyPlacement(grid map[[2]int]cwCell, word string, p crosswordPlacement) {
	for i := 0; i < len(word); i++ {
		r, c := p.Row, p.Col
		if p.Dir == "h" {
			c += i
		} else {
			r += i
		}

		cell := grid[[2]int{r, c}]
		cell.letter = word[i]

		if p.Dir == "h" {
			cell.horiz = true
		} else {
			cell.vert = true
		}

		grid[[2]int{r, c}] = cell
	}
}

func cloneGrid(grid map[[2]int]cwCell) map[[2]int]cwCell {
	out := make(map[[2]int]cwCell, len(grid))
	for k, v := range grid {
		out[k] = v
	}

	return out
}

// normalizeCrossword 把邊界框歸零到 (0,0)，回傳網格尺寸
func normalizeCrossword(
	placements []crosswordPlacement,
	words []string,
) ([]crosswordPlacement, int, int) {
	minRow, minCol := math.MaxInt, math.MaxInt
	maxRow, maxCol := math.MinInt, math.MinInt

	any := false

	for i, p := range placements {
		if p.Row == -1 {
			continue
		}

		any = true
		endRow, endCol := p.Row, p.Col

		if p.Dir == "h" {
			endCol += len(words[i]) - 1
		} else {
			endRow += len(words[i]) - 1
		}

		if p.Row < minRow {
			minRow = p.Row
		}

		if p.Col < minCol {
			minCol = p.Col
		}

		if endRow > maxRow {
			maxRow = endRow
		}

		if endCol > maxCol {
			maxCol = endCol
		}
	}

	if !any {
		return placements, 0, 0
	}

	out := make([]crosswordPlacement, len(placements))

	for i, p := range placements {
		if p.Row == -1 {
			out[i] = crosswordPlacement{Row: -1, Col: -1}
			continue
		}

		out[i] = crosswordPlacement{Row: p.Row - minRow, Col: p.Col - minCol, Dir: p.Dir}
	}

	return out, maxRow - minRow + 1, maxCol - minCol + 1
}

// crosswordLevel 進行中的交叉字謎關卡（含答案，只存 LevelStore；
// 獨立於 learnLevel，不與 fill/wheel 共用結構）
type crosswordLevel struct {
	ID        string    `json:"id"`
	UserID    uint      `json:"user_id"`
	Tier      int       `json:"tier"`
	Letters   string    `json:"letters"`
	WordIDs   []uint    `json:"word_ids"`
	Words     []string  `json:"words"`
	Phonetics []string  `json:"phonetics"`
	Defs      []string  `json:"defs"`
	Row       []int     `json:"row"` // 平行陣列；-1 = bonus 字（未排進網格）
	Col       []int     `json:"col"`
	Dir       []string  `json:"dir"` // "h" / "v"；bonus 字為空字串
	Solved    []bool    `json:"solved"`
	HintTier  []int     `json:"hint_tier"` // 0=無, 1=揭字母, 2=看釋義
	HintPos   []int     `json:"hint_pos"`  // 已揭露的字母位置；-1=尚未揭露
	Rows      int       `json:"rows"`
	Cols      int       `json:"cols"`
	XP        int       `json:"xp"`
	CreatedAt time.Time `json:"created_at"`
}

// CrosswordSlot 單一答案在交叉字謎網格中的呈現（下發 client，不含未解出的答案）
type CrosswordSlot struct {
	Row        int    `json:"row"`
	Col        int    `json:"col"`
	Dir        string `json:"dir,omitempty"` // 空字串 = bonus 字，此時 row/col 無意義
	Length     int    `json:"length"`
	Masked     string `json:"masked"`
	Definition string `json:"definition,omitempty"`
	Solved     bool   `json:"solved"`
	Word       string `json:"word,omitempty"`
}

// CrosswordView 交叉字謎關卡謎面
type CrosswordView struct {
	LevelID string          `json:"level_id"`
	Tier    int             `json:"tier"`
	Rows    int             `json:"rows"`
	Cols    int             `json:"cols"`
	Letters string          `json:"letters"`
	Words   []CrosswordSlot `json:"words"`
}

// CreateCrosswordLevel 生成新的交叉字謎網格關卡
func (s *learnService) CreateCrosswordLevel(
	userID uint, tier int, locale string,
) (*CrosswordView, error) {
	if tier < 1 || tier > 5 {
		return nil, ErrLearnInvalidTier
	}

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
		return nil, fmt.Errorf("no crossword puzzle available for tier %d", tier)
	}

	return s.buildCrosswordLevel(userID, tier, locale, base, ids, idx)
}

func (s *learnService) buildCrosswordLevel(
	userID uint, tier int, locale string,
	base *model.Word, ids []uint, idx *anagramIndex,
) (*CrosswordView, error) {
	answers := make([]*model.Word, 0, len(ids))
	for _, id := range ids {
		answers = append(answers, idx.words[id])
	}

	// 短字在前、常用字優先；上限 8 個，底字必收（沿用 wheel 既有規則）
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

	picked = append(picked, base)

	words := make([]string, len(picked))
	for i, w := range picked {
		words[i] = w.Word
	}

	placements, rows, cols := layoutCrossword(words)

	rng := newLevelRand()
	letters := []byte(base.Word)
	rng.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })

	lv := &crosswordLevel{
		ID:        uuid.NewString(),
		UserID:    userID,
		Tier:      tier,
		Letters:   string(letters),
		Rows:      rows,
		Cols:      cols,
		CreatedAt: time.Now().UTC(),
	}

	for i, w := range picked {
		lv.WordIDs = append(lv.WordIDs, w.ID)
		lv.Words = append(lv.Words, w.Word)
		lv.Phonetics = append(lv.Phonetics, w.Phonetic)
		lv.Defs = append(lv.Defs, definitionFor(w, locale))
		lv.Row = append(lv.Row, placements[i].Row)
		lv.Col = append(lv.Col, placements[i].Col)
		lv.Dir = append(lv.Dir, placements[i].Dir)
		lv.Solved = append(lv.Solved, false)
		lv.HintTier = append(lv.HintTier, 0)
		lv.HintPos = append(lv.HintPos, -1)
	}

	if err := saveEnvelope(s.store, lv.ID, ModeCrossword, lv); err != nil {
		return nil, err
	}

	return crosswordView(lv), nil
}

// crosswordView 轉成下發 client 的謎面（未解字絕不含答案）
func crosswordView(lv *crosswordLevel) *CrosswordView {
	v := &CrosswordView{
		LevelID: lv.ID, Tier: lv.Tier, Rows: lv.Rows, Cols: lv.Cols, Letters: lv.Letters,
	}

	for i, word := range lv.Words {
		slot := CrosswordSlot{
			Length: len(word), Solved: lv.Solved[i],
			Masked: maskWithHint(word, lv.HintPos[i]),
		}

		if lv.Dir[i] != "" {
			slot.Row = lv.Row[i]
			slot.Col = lv.Col[i]
			slot.Dir = lv.Dir[i]
		}

		if lv.HintTier[i] >= 2 {
			slot.Definition = lv.Defs[i]
		}

		if lv.Solved[i] {
			slot.Word = word
			slot.Masked = word
			slot.Definition = lv.Defs[i]
		}

		v.Words = append(v.Words, slot)
	}

	return v
}

// hintCrossword 處理 crossword 模式的提示前進
func (s *learnService) hintCrossword(
	userID uint,
	env *levelEnvelope,
	slot int,
) (*HintOutcome, error) {
	var lv crosswordLevel
	if err := json.Unmarshal(env.Data, &lv); err != nil {
		return nil, err
	}

	if lv.UserID != userID {
		return nil, ErrLearnLevelNotFound
	}

	if slot < 0 || slot >= len(lv.Words) {
		return nil, ErrLearnLevelNotFound
	}

	if lv.Solved[slot] {
		return nil, ErrLearnSlotSolved
	}

	out := advanceHint(&lv.HintTier[slot], &lv.HintPos[slot], lv.Words[slot], lv.Defs[slot], slot)

	if err := saveEnvelope(s.store, lv.ID, ModeCrossword, &lv); err != nil {
		return nil, err
	}

	return out, nil
}

// revealCrossword 處理 crossword 模式的揭曉答案
func (s *learnService) revealCrossword(
	userID uint,
	env *levelEnvelope,
	slot int,
) (*GuessOutcome, error) {
	var lv crosswordLevel
	if err := json.Unmarshal(env.Data, &lv); err != nil {
		return nil, err
	}

	if lv.UserID != userID {
		return nil, ErrLearnLevelNotFound
	}

	if slot < 0 || slot >= len(lv.Words) {
		return nil, ErrLearnLevelNotFound
	}

	if lv.Solved[slot] {
		return nil, ErrLearnSlotSolved
	}

	lv.Solved[slot] = true

	out := &GuessOutcome{
		Correct: true, Slot: slot,
		Word: lv.Words[slot], Phonetic: lv.Phonetics[slot], Definition: lv.Defs[slot],
	}
	out.Completed = allSolved(lv.Solved)

	if out.Completed {
		out.TotalXP = lv.XP

		if err := s.onLevelCompleted(userID, lv.XP, "", lv.CreatedAt); err != nil {
			return nil, err
		}
	}

	if err := saveEnvelope(s.store, lv.ID, ModeCrossword, &lv); err != nil {
		return nil, err
	}

	return out, nil
}

// guessCrossword 處理 crossword 的作答：字母盤送字，後端找出命中哪個字
func (s *learnService) guessCrossword(
	userID uint, env *levelEnvelope, req *LearnGuessRequest,
) (*GuessOutcome, error) {
	var lv crosswordLevel
	if err := json.Unmarshal(env.Data, &lv); err != nil {
		return nil, err
	}

	if lv.UserID != userID {
		return nil, ErrLearnLevelNotFound
	}

	guess := strings.ToLower(strings.TrimSpace(req.Word))

	slot := -1

	for i, w := range lv.Words {
		if w == guess {
			slot = i

			break
		}
	}

	if slot == -1 {
		return &GuessOutcome{Correct: false, Slot: -1}, nil
	}

	if lv.Solved[slot] {
		return nil, ErrLearnSlotSolved
	}

	if err := s.repo.UpsertWordRecord(userID, lv.WordIDs[slot], true); err != nil {
		return nil, err
	}

	lv.Solved[slot] = true

	out := &GuessOutcome{
		Correct:    true,
		Slot:       slot,
		Word:       lv.Words[slot],
		Phonetic:   lv.Phonetics[slot],
		Definition: lv.Defs[slot],
		XPAwarded: hintDiscount(
			wordXP(lv.Words[slot], lv.Tier, ModeWheel), lv.HintTier[slot],
		), // 猜字機制同 wheel，套用同一套 XP 檔次
	}
	lv.XP += out.XPAwarded
	out.Completed = allSolved(lv.Solved)

	if out.Completed {
		out.TotalXP = lv.XP

		if err := s.onLevelCompleted(userID, lv.XP, "", lv.CreatedAt); err != nil {
			return nil, err
		}
	}

	if err := saveEnvelope(s.store, lv.ID, ModeCrossword, &lv); err != nil {
		return nil, err
	}

	return out, nil
}
