package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/walnut-almonds/talkrealm/internal/model"
)

// 例句填空 SRS（間隔重複）複習模式：從「到期舊字」+「少量新字」組成一場 session，
// 每張卡片挖空一個目標單字讓使用者填答。答對進位間隔（1/2/4/7/15 天），答錯重置。
// 排程狀態存在 LearnWordRecord 的 srs_stage/next_review_at（只此模式維護）。
const (
	ModeSRS     = "srs"
	srsMaxCount = 50
	srsNewRatio = 4 // 新字約佔 1/4
	srsMaxStage = 5 // 畢業階段（間隔封頂）
)

var (
	ErrLearnCardGraded  = errors.New("card already answered")
	ErrLearnNoSentences = errors.New("no sentences due or available for review")
)

// srsNextStage 依作答結果算下一個階段：答錯歸零重置，答對進位到封頂
func srsNextStage(prevStage int, correct bool) int {
	if !correct {
		return 0
	}

	if prevStage >= srsMaxStage {
		return srsMaxStage
	}

	return prevStage + 1
}

// srsIntervalDays 依「結果階段」查下次複習間隔（天）：
// 0(新/答錯重置)=1、1=1、2=2、3=4、4=7、5+=15。對應使用者要的 1/2/4/7 遞增曲線。
func srsIntervalDays(stage int) int {
	switch stage {
	case 0, 1:
		return 1
	case 2:
		return 2
	case 3:
		return 4
	case 4:
		return 7
	default:
		return 15
	}
}

// srsSession 進行中的複習 session（含答案，只存 LevelStore）
type srsSession struct {
	ID        string    `json:"id"`
	UserID    uint      `json:"user_id"`
	Items     []srsItem `json:"items"`
	XP        int       `json:"xp"`
	CreatedAt time.Time `json:"created_at"`
}

type srsItem struct {
	WordID   uint   `json:"word_id"`
	Answer   string `json:"answer"`
	TextEN   string `json:"text_en"` // 內含 "{{}}" 挖空標記
	Trans    string `json:"trans"`
	Phonetic string `json:"phonetic"`
	Tier     int    `json:"tier"`
	IsNew    bool   `json:"is_new"`
	// 答對才「退場」（Retired）；退場前答錯過就記 EverWrong——
	// 退場時據此決定長期排程：整場乾淨才進位，中途錯過就重置（lapse）。
	Retired   bool `json:"retired"`
	EverWrong bool `json:"ever_wrong"`
}

// SRSCardView 下發 client 的卡片（不含答案）
type SRSCardView struct {
	Index  int    `json:"index"`
	TextEN string `json:"text_en"` // 含 "{{}}"，前端渲染成輸入框
	Trans  string `json:"trans"`
	Length int    `json:"length"` // 答案長度（輸入框寬度提示）
	IsNew  bool   `json:"is_new"`
}

// SRSSessionView 複習 session 謎面
type SRSSessionView struct {
	SessionID string        `json:"session_id"`
	Total     int           `json:"total"`
	NewCount  int           `json:"new_count"`
	Cards     []SRSCardView `json:"cards"`
}

// SRSOverviewView 複習概況（hub 顯示今日待複習/可學新字）
type SRSOverviewView struct {
	DueCount     int `json:"due_count"`
	NewAvailable int `json:"new_available"`
}

// SRSAnswerOutcome 單卡作答結果
type SRSAnswerOutcome struct {
	Correct   bool   `json:"correct"`
	Answer    string `json:"answer"`
	Phonetic  string `json:"phonetic,omitempty"`
	Trans     string `json:"trans,omitempty"`
	XPAwarded int    `json:"xp_awarded"`
	NextStage int    `json:"next_stage"`
	Completed bool   `json:"completed"`
	TotalXP   int    `json:"total_xp,omitempty"`
}

// SRSOverview 回傳今日到期與可學新字數
func (s *learnService) SRSOverview(userID uint) (*SRSOverviewView, error) {
	now := time.Now().UTC()

	due, err := s.repo.CountDueReviews(userID, now)
	if err != nil {
		return nil, err
	}

	fresh, err := s.repo.CountNewSentenceWords(userID)
	if err != nil {
		return nil, err
	}

	return &SRSOverviewView{DueCount: int(due), NewAvailable: int(fresh)}, nil
}

// CreateSRSSession 組建一場複習：目標 count 題，新字約佔 1/4，其餘為到期舊字；
// 材料不足時回傳較少題，兩者皆空回 ErrLearnNoSentences。
func (s *learnService) CreateSRSSession(
	userID uint, count int, locale string,
) (*SRSSessionView, error) {
	if count < 1 {
		count = 10
	}

	if count > srsMaxCount {
		count = srsMaxCount
	}

	now := time.Now().UTC()

	// 先各抓最多 count 個 ID（只是輕量 id，之後再分配），due 與 new 互斥（stage>=1 vs stage=0）
	newIDs, err := s.repo.NewSentenceWordIDs(userID, count)
	if err != nil {
		return nil, err
	}

	dueIDs, err := s.repo.DueReviewWordIDs(userID, now, count)
	if err != nil {
		return nil, err
	}

	chosen := planSRSSession(count, newIDs, dueIDs)
	if len(chosen) == 0 {
		return nil, ErrLearnNoSentences
	}

	items, err := s.buildSRSItems(chosen, locale)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, ErrLearnNoSentences
	}

	rng := newLevelRand()
	rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })

	sess := &srsSession{
		ID: uuid.NewString(), UserID: userID, Items: items, CreatedAt: now,
	}

	if err := saveEnvelope(s.store, sess.ID, ModeSRS, sess); err != nil {
		return nil, err
	}

	return srsSessionView(sess), nil
}

// srsPick 待組卡的字：word id + 是否新字
type srsPick struct {
	wordID uint
	isNew  bool
}

// planSRSSession 依「新字約 1/4、其餘到期舊字」分配，材料不足時互相回填。
// newQuota 用 floor（非 ceil）：到期複習優先，小場次全給舊字；沒有舊字時再用新字回填。
func planSRSSession(count int, newIDs, dueIDs []uint) []srsPick {
	newQuota := count / srsNewRatio // floor(count/4)

	takeNew := min(newQuota, len(newIDs))
	takeDue := min(count-takeNew, len(dueIDs))

	// 舊字不夠 → 多補新字；新字不夠 → 多補舊字
	if short := count - takeNew - takeDue; short > 0 {
		if extra := min(short, len(newIDs)-takeNew); extra > 0 {
			takeNew += extra
			short -= extra
		}

		if extra := min(short, len(dueIDs)-takeDue); extra > 0 {
			takeDue += extra
		}
	}

	picks := make([]srsPick, 0, takeNew+takeDue)
	for _, id := range dueIDs[:takeDue] {
		picks = append(picks, srsPick{wordID: id, isNew: false})
	}

	for _, id := range newIDs[:takeNew] {
		picks = append(picks, srsPick{wordID: id, isNew: true})
	}

	return picks
}

// buildSRSItems 為每個字取一則例句 + 全欄位字料，組成 session item
func (s *learnService) buildSRSItems(picks []srsPick, locale string) ([]srsItem, error) {
	ids := make([]uint, len(picks))
	for i, p := range picks {
		ids[i] = p.wordID
	}

	words, err := s.repo.WordsByIDs(ids)
	if err != nil {
		return nil, err
	}

	byID := map[uint]*model.Word{}
	for _, w := range words {
		byID[w.ID] = w
	}

	items := make([]srsItem, 0, len(picks))

	for _, p := range picks {
		sentence, err := s.repo.RandomSentenceByWord(p.wordID)
		if err != nil {
			return nil, err
		}

		if sentence == nil {
			continue // 理論上 due/new 篩選已保證有例句，防禦性跳過
		}

		w := byID[p.wordID]
		phonetic, tier := "", 1

		if w != nil {
			phonetic = w.Phonetic
			tier = w.Tier
		}

		items = append(items, srsItem{
			WordID: p.wordID, Answer: sentence.Answer,
			TextEN: sentence.TextEN, Trans: sentenceTrans(sentence, locale),
			Phonetic: phonetic, Tier: tier, IsNew: p.isNew,
		})
	}

	return items, nil
}

// AnswerSRS 作答一張卡。答錯不退場（記 EverWrong，前端排回佇列稍後再考），
// 答對才退場並寫入長期排程：整場乾淨過關才進位，中途錯過即重置（lapse）。
// 只有已退場的卡再作答才拒絕（ErrLearnCardGraded）。
func (s *learnService) AnswerSRS(
	userID uint, sessionID string, index int, guess string,
) (*SRSAnswerOutcome, error) {
	env, err := loadEnvelope(s.store, sessionID)
	if err != nil {
		return nil, err
	}

	if env.Mode != ModeSRS {
		return nil, ErrLearnLevelNotFound
	}

	var sess srsSession
	if err := json.Unmarshal(env.Data, &sess); err != nil {
		return nil, err
	}

	if sess.UserID != userID {
		return nil, ErrLearnLevelNotFound
	}

	if index < 0 || index >= len(sess.Items) {
		return nil, ErrLearnLevelNotFound
	}

	item := &sess.Items[index]
	if item.Retired {
		return nil, ErrLearnCardGraded
	}

	correct := strings.EqualFold(strings.TrimSpace(guess), item.Answer)

	out := &SRSAnswerOutcome{
		Correct: correct, Answer: item.Answer,
		Phonetic: item.Phonetic, Trans: item.Trans,
	}

	if !correct {
		item.EverWrong = true
		out.NextStage = 0 // 已 lapse，退場時將重置

		if err := saveEnvelope(s.store, sess.ID, ModeSRS, &sess); err != nil {
			return nil, err
		}

		return out, nil // 不退場、不寫 DB：留在本場佇列，前端稍後再考
	}

	// 答對 → 退場並結算長期排程
	item.Retired = true
	clean := !item.EverWrong

	rec, err := s.repo.GetWordRecord(userID, item.WordID)
	if err != nil {
		return nil, err
	}

	prevStage := 0
	if rec != nil {
		prevStage = rec.SRSStage
	}

	finalStage := srsNextStage(prevStage, clean) // 乾淨進位；lapse 重置為 0
	nextReviewAt := time.Now().UTC().AddDate(0, 0, srsIntervalDays(finalStage))

	// counter 的 correct 傳 clean：乾淨複習計 correct_count、lapse 計 wrong_count
	if err := s.repo.SaveSRSResult(
		userID,
		item.WordID,
		finalStage,
		nextReviewAt,
		clean,
	); err != nil {
		return nil, err
	}

	out.NextStage = finalStage
	out.XPAwarded = wordXP(item.Answer, item.Tier, ModeFill) // 例句填空同 fill 檔次（全字回想 ×1.5）

	if !clean {
		out.XPAwarded /= 2 // lapse 折半：反覆猜到對不該與一次答對同分
	}

	sess.XP += out.XPAwarded
	out.Completed = allSRSRetired(sess.Items)

	if out.Completed {
		out.TotalXP = sess.XP

		if err := s.onLevelCompleted(userID, sess.XP, "", sess.CreatedAt); err != nil {
			return nil, err
		}
	}

	if err := saveEnvelope(s.store, sess.ID, ModeSRS, &sess); err != nil {
		return nil, err
	}

	return out, nil
}

func allSRSRetired(items []srsItem) bool {
	for i := range items {
		if !items[i].Retired {
			return false
		}
	}

	return true
}

func srsSessionView(sess *srsSession) *SRSSessionView {
	v := &SRSSessionView{SessionID: sess.ID, Total: len(sess.Items)}

	for i := range sess.Items {
		it := &sess.Items[i]
		if it.IsNew {
			v.NewCount++
		}

		v.Cards = append(v.Cards, SRSCardView{
			Index: i, TextEN: it.TextEN, Trans: it.Trans,
			Length: len(it.Answer), IsNew: it.IsNew,
		})
	}

	return v
}

// sentenceTrans 依 locale 挑例句翻譯（母語語意提示）。英文介面或缺翻譯時回空字串——
// 英文句子本身就是語境，不需要也不該把含 {{}} 的原句當「翻譯」顯示。
func sentenceTrans(s *model.LearnSentence, locale string) string {
	switch locale {
	case "zh":
		return s.TextZH
	case langKeyZHTW:
		return s.TextZHTW
	case "ja":
		return s.TextJA
	default:
		return ""
	}
}
