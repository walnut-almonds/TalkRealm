package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/walnut-almonds/talkrealm/internal/model"
)

// 固定關卡（campaign）：開機冪等生成 1..campaignLevelCount 關（目前全為 easy tier），
// 存 DB 後不可變——所有玩家玩同一份題目，首通分數才有可比性。
// 遊玩沿用 crossword 的 envelope/Guess/Hint/Reveal 全套流程，只是謎題來源不同。
const (
	campaignLevelCount = 50
	campaignTier       = 1
	campaignGenTries   = 30
)

var ErrLearnCampaignLocked = errors.New("campaign level locked: clear the previous level first")

// campaignWordCount 關卡編號 → 答案數的難度曲線（1-10 關 3 字，每 10 關 +1，41-50 關 7 字）
func campaignWordCount(no int) int {
	return 3 + (no-1)/10
}

// campaignBaseLen 關卡編號 → 底字長度（= 字母輪的字母數）難度曲線。
// 只影響關卡生成，不影響計分；避免第 1 關就出現 7 字母底字。
func campaignBaseLen(no int) int {
	switch {
	case no <= 5:
		return 3
	case no <= 10:
		return 4
	case no <= 20:
		return 5
	case no <= 35:
		return 6
	default:
		return 7
	}
}

// campaignFindBase 挑指定長度的底字：優先答案數 ≥ want，退而求其次 ≥2
func campaignFindBase(
	candidates []*model.Word, idx *anagramIndex, baseLen, want int,
) (*model.Word, []uint, bool) {
	var (
		fbBase *model.Word
		fbIDs  []uint
	)

	for _, c := range candidates {
		if len(c.Word) != baseLen {
			continue
		}

		ids := idx.subWordIDs(c.Word)
		if len(ids) >= want {
			return c, ids, true
		}

		if fbBase == nil && len(ids) >= 2 {
			fbBase, fbIDs = c, ids
		}
	}

	if fbBase != nil {
		return fbBase, fbIDs, true
	}

	return nil, nil, false
}

// campaignPuzzle 固定關卡謎題（存 DB 的 JSON；只含排進網格的字）
type campaignPuzzle struct {
	Letters string   `json:"letters"`
	WordIDs []uint   `json:"word_ids"`
	Words   []string `json:"words"`
	Rows    int      `json:"rows"`
	Cols    int      `json:"cols"`
	Row     []int    `json:"row"`
	Col     []int    `json:"col"`
	Dir     []string `json:"dir"`
}

// CampaignLevelView 關卡選擇清單的單關狀態
type CampaignLevelView struct {
	LevelNo int  `json:"level_no"`
	Done    bool `json:"done"`
	Score   int  `json:"score,omitempty"`
}

// CampaignOverviewView 關卡總覽（含個人進度）
type CampaignOverviewView struct {
	Total    int                 `json:"total"`
	Furthest int                 `json:"furthest"` // 已通關的最大關卡編號；0 = 尚未通關任何關
	Levels   []CampaignLevelView `json:"levels"`
}

// EnsureCampaignLevels 開機冪等生成缺少的固定關卡；回傳本次新生成的關卡數。
// 已存在的關卡絕不重生（發布後不可變）。
func (s *learnService) EnsureCampaignLevels() (int, error) {
	nos, err := s.repo.CampaignLevelNos()
	if err != nil {
		return 0, err
	}

	have := map[int]bool{}
	for _, no := range nos {
		have[no] = true
	}

	if len(have) >= campaignLevelCount {
		return 0, nil
	}

	idx, err := s.anagram()
	if err != nil {
		return 0, err
	}

	created := 0
	usedBase := map[string]bool{} // key=底字排序字母簽名：同一輪生成盡量不重複 anagram 組，避免關卡雷同

	for no := 1; no <= campaignLevelCount; no++ {
		if have[no] {
			continue
		}

		pz, err := s.generateCampaignPuzzle(no, idx, usedBase)
		if err != nil {
			return created, fmt.Errorf("generate campaign level %d: %w", no, err)
		}

		b, err := json.Marshal(pz)
		if err != nil {
			return created, err
		}

		err = s.repo.CreateCampaignLevel(&model.LearnCampaignLevel{
			LevelNo: no, Tier: campaignTier, Puzzle: string(b), CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return created, err
		}

		created++
	}

	return created, nil
}

// generateCampaignPuzzle 兩階段抽字排版：
// 第一階段堅持指定底字長度，「湊滿目標字數且全排進」立即接受，字數不足的成品記為候選；
// 第一階段只要有任何成品（即使字數不足）就直接採用——長度優先於字數，
// 3 字母 2 個答案的簡單關，好過 5 字母 3 個答案的「假 easy」。
// 第一階段全空手（字池缺該長度）才進第二階段放寬長度保底。
func (s *learnService) generateCampaignPuzzle(
	no int, idx *anagramIndex, usedBase map[string]bool,
) (*campaignPuzzle, error) {
	want := campaignWordCount(no)
	baseLen := campaignBaseLen(no)

	var (
		best       *campaignPuzzle
		bestPlaced int
		bestSig    string
	)

	for _, relax := range []bool{false, true} {
		for try := 0; try < campaignGenTries; try++ {
			pz, placed, sig, err := s.tryCampaignDraw(idx, usedBase, baseLen, want, relax)
			if err != nil {
				return nil, err
			}

			if pz == nil {
				continue
			}

			// placed ≤ 挑中字數 ≤ want，等於 want 即代表湊滿且全排進
			if placed == want {
				usedBase[sig] = true

				return pz, nil
			}

			if placed > bestPlaced {
				best, bestPlaced, bestSig = pz, placed, sig
			}
		}

		if best != nil {
			break // 本階段已有成品，不進入放寬階段
		}
	}

	if best == nil {
		return nil, fmt.Errorf(
			"no valid puzzle after %d tries (word pool too small?)",
			campaignGenTries*2,
		)
	}

	usedBase[bestSig] = true

	return best, nil
}

// tryCampaignDraw 抽一批候選字並嘗試組出謎題；relax 時放寬底字長度限制。
// 這批組不出東西回傳 (nil, 0, "", nil)，由呼叫端重抽。
func (s *learnService) tryCampaignDraw(
	idx *anagramIndex, usedBase map[string]bool, baseLen, want int, relax bool,
) (*campaignPuzzle, int, string, error) {
	// 抽多一點（50）提高命中指定長度底字的機率
	candidates, err := s.repo.RandomWordsByTier(campaignTier, 50)
	if err != nil {
		return nil, 0, "", err
	}

	fresh := make([]*model.Word, 0, len(candidates))

	for _, c := range candidates {
		if !usedBase[sortLetters(c.Word)] {
			fresh = append(fresh, c)
		}
	}

	base, ids, ok := campaignFindBase(fresh, idx, baseLen, want)
	if !ok && relax {
		// 放寬階段才逐步讓步：先允許重複 anagram 組（字池小），再退回一般 wheel 規則（字池缺該長度）
		base, ids, ok = campaignFindBase(candidates, idx, baseLen, want)
		if !ok {
			base, ids, ok = findWheelBase(candidates, idx)
		}
	}

	if !ok {
		return nil, 0, "", nil
	}

	picked := pickWheelAnswers(base, ids, idx, want)

	words := make([]string, len(picked))
	for i, w := range picked {
		words[i] = w.Word
	}

	placements, rows, cols := layoutCrossword(words)
	pz := &campaignPuzzle{Rows: rows, Cols: cols}
	placed := 0

	for i, p := range placements {
		if p.Row == -1 {
			continue
		}

		placed++

		pz.WordIDs = append(pz.WordIDs, picked[i].ID)
		pz.Words = append(pz.Words, picked[i].Word)
		pz.Row = append(pz.Row, p.Row)
		pz.Col = append(pz.Col, p.Col)
		pz.Dir = append(pz.Dir, p.Dir)
	}

	if placed < 2 {
		return nil, 0, "", nil
	}

	rng := newLevelRand()
	letters := []byte(base.Word)
	rng.Shuffle(
		len(letters),
		func(i, j int) { letters[i], letters[j] = letters[j], letters[i] },
	)
	pz.Letters = string(letters)

	return pz, placed, sortLetters(base.Word), nil
}

// pickWheelAnswers 依 wheel 既有規則（短字在前、常用優先、底字必收）挑最多 limit 個答案；
// wheel/crossword/campaign 三種生成共用
func pickWheelAnswers(
	base *model.Word, ids []uint, idx *anagramIndex, limit int,
) []*model.Word {
	answers := make([]*model.Word, 0, len(ids))
	for _, id := range ids {
		answers = append(answers, idx.words[id])
	}

	sortWheelAnswers(answers)

	picked := []*model.Word{}

	for _, a := range answers {
		if a.ID == base.ID {
			continue
		}

		if len(picked) < limit-1 {
			picked = append(picked, a)
		}
	}

	return append(picked, base)
}

// StartCampaignLevel 開始固定關卡：前一關已首通（或第 1 關）才可玩
func (s *learnService) StartCampaignLevel(
	userID uint, levelNo int, locale string,
) (*CrosswordView, error) {
	rec, err := s.repo.CampaignLevelByNo(levelNo)
	if err != nil {
		return nil, err
	}

	if rec == nil {
		return nil, ErrLearnLevelNotFound
	}

	if levelNo > 1 {
		cleared, err := s.hasCampaignCleared(userID, levelNo-1)
		if err != nil {
			return nil, err
		}

		if !cleared {
			return nil, ErrLearnCampaignLocked
		}
	}

	var pz campaignPuzzle
	if err := json.Unmarshal([]byte(rec.Puzzle), &pz); err != nil {
		return nil, err
	}

	fullWords, err := s.repo.WordsByIDs(pz.WordIDs)
	if err != nil {
		return nil, err
	}

	byID := map[uint]*model.Word{}
	for _, w := range fullWords {
		byID[w.ID] = w
	}

	lv := &crosswordLevel{
		ID:        uuid.NewString(),
		UserID:    userID,
		Tier:      rec.Tier,
		Campaign:  levelNo,
		Letters:   pz.Letters,
		Rows:      pz.Rows,
		Cols:      pz.Cols,
		CreatedAt: time.Now().UTC(),
	}

	for i, id := range pz.WordIDs {
		// 拼字以 puzzle 存檔為準（字表異動不影響已發布關卡）；音標/釋義/計分 tier 由字表即時補
		phonetic, def := "", ""
		wordTier := rec.Tier

		if w := byID[id]; w != nil {
			phonetic = w.Phonetic
			def = definitionFor(w, locale)
			wordTier = w.Tier
		}

		lv.WordIDs = append(lv.WordIDs, id)
		lv.Words = append(lv.Words, pz.Words[i])
		lv.Phonetics = append(lv.Phonetics, phonetic)
		lv.Defs = append(lv.Defs, def)
		lv.Tiers = append(lv.Tiers, wordTier)
		lv.Row = append(lv.Row, pz.Row[i])
		lv.Col = append(lv.Col, pz.Col[i])
		lv.Dir = append(lv.Dir, pz.Dir[i])
		lv.Solved = append(lv.Solved, false)
		lv.HintTier = append(lv.HintTier, 0)
		lv.HintPos = append(lv.HintPos, -1)
	}

	if err := saveEnvelope(s.store, lv.ID, ModeCrossword, lv); err != nil {
		return nil, err
	}

	return crosswordView(lv), nil
}

func (s *learnService) hasCampaignCleared(userID uint, levelNo int) (bool, error) {
	ps, err := s.repo.CampaignProgress(userID)
	if err != nil {
		return false, err
	}

	for _, p := range ps {
		if p.LevelNo == levelNo {
			return true, nil
		}
	}

	return false, nil
}

// CampaignOverview 關卡清單 + 個人進度
func (s *learnService) CampaignOverview(userID uint) (*CampaignOverviewView, error) {
	nos, err := s.repo.CampaignLevelNos()
	if err != nil {
		return nil, err
	}

	ps, err := s.repo.CampaignProgress(userID)
	if err != nil {
		return nil, err
	}

	byNo := map[int]*model.LearnCampaignProgress{}
	for _, p := range ps {
		byNo[p.LevelNo] = p
	}

	view := &CampaignOverviewView{Total: len(nos)}

	for _, no := range nos {
		lv := CampaignLevelView{LevelNo: no}

		if p := byNo[no]; p != nil {
			lv.Done = true
			lv.Score = p.Score

			if no > view.Furthest {
				view.Furthest = no
			}
		}

		view.Levels = append(view.Levels, lv)
	}

	return view, nil
}

// CampaignLeaderboard 關卡榜（首通分加總排序，最遠關卡 tiebreak；friends 時限好友+自己）
func (s *learnService) CampaignLeaderboard(
	userID uint, friends bool,
) (*LeaderboardView, error) {
	ids, err := s.scopeIDs(userID, friends)
	if err != nil {
		return nil, err
	}

	totals, err := s.repo.CampaignTotals(ids, 10)
	if err != nil {
		return nil, err
	}

	view := &LeaderboardView{}

	for i, tp := range totals {
		e := s.boardEntry(i+1, tp.UserID, tp.Total)
		e.Level = tp.Furthest
		view.Top = append(view.Top, e)
	}

	me, rank, err := s.repo.CampaignRank(userID, ids)
	if err != nil {
		return nil, err
	}

	if me != nil {
		e := s.boardEntry(rank, me.UserID, me.Total)
		e.Level = me.Furthest
		view.Me = &e
	}

	return view, nil
}

// WeeklyLeaderboard 本週 XP 成長榜（friends 時限好友+自己）
func (s *learnService) WeeklyLeaderboard(
	userID uint, friends bool,
) (*LeaderboardView, error) {
	ids, err := s.scopeIDs(userID, friends)
	if err != nil {
		return nil, err
	}

	week := isoWeek(time.Now().UTC())

	top, err := s.repo.TopWeeklyXP(week, ids, 10)
	if err != nil {
		return nil, err
	}

	view := &LeaderboardView{Week: week}

	for i, row := range top {
		view.Top = append(view.Top, s.boardEntry(i+1, row.UserID, row.XP))
	}

	me, rank, err := s.repo.WeeklyRank(userID, week, ids)
	if err != nil {
		return nil, err
	}

	if me != nil {
		e := s.boardEntry(rank, me.UserID, me.XP)
		view.Me = &e
	}

	return view, nil
}

// scopeIDs 好友榜的範圍：好友 + 自己；全球榜回 nil（不限定）
func (s *learnService) scopeIDs(userID uint, friends bool) ([]uint, error) {
	if !friends {
		return nil, nil
	}

	if s.friends == nil {
		return []uint{userID}, nil
	}

	ids, err := s.friends.FriendIDs(userID)
	if err != nil {
		return nil, err
	}

	return append(ids, userID), nil
}

func isoWeek(t time.Time) string {
	y, w := t.ISOWeek()

	return fmt.Sprintf("%d-W%02d", y, w)
}
