//nolint:testpackage // 白箱測試：需存取未匯出的 SRS 排程邏輯與 session 結構
package service

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
)

func TestSRSIntervalCurve(t *testing.T) {
	// 答對連續進位：1→1、2→2、3→4、4→7、5+→15 天（封頂）
	tests := []struct{ stage, days int }{
		{0, 1}, {1, 1}, {2, 2}, {3, 4}, {4, 7}, {5, 15}, {6, 15},
	}
	for _, tt := range tests {
		if got := srsIntervalDays(tt.stage); got != tt.days {
			t.Errorf("srsIntervalDays(%d) = %d want %d", tt.stage, got, tt.days)
		}
	}
}

func TestSRSNextStage(t *testing.T) {
	if got := srsNextStage(0, true); got != 1 { // 新字答對 → 進 stage1
		t.Errorf("new correct = %d want 1", got)
	}

	if got := srsNextStage(3, true); got != 4 { // 進位
		t.Errorf("advance = %d want 4", got)
	}

	if got := srsNextStage(srsMaxStage, true); got != srsMaxStage { // 封頂不再進
		t.Errorf("cap = %d want %d", got, srsMaxStage)
	}

	if got := srsNextStage(4, false); got != 0 { // 答錯重置
		t.Errorf("wrong reset = %d want 0", got)
	}
}

func TestPlanSRSSession(t *testing.T) {
	// 8 題：新字配額 ceil(8/4)=2，其餘 6 為到期舊字
	picks := planSRSSession(8, []uint{1, 2, 3, 4}, []uint{10, 11, 12, 13, 14, 15, 16})
	newN, dueN := countPicks(picks)

	if len(picks) != 8 || newN != 2 || dueN != 6 {
		t.Fatalf("balanced: total=%d new=%d due=%d, want 8/2/6", len(picks), newN, dueN)
	}

	// 到期不足 → 用新字回填
	picks = planSRSSession(8, []uint{1, 2, 3, 4, 5, 6, 7, 8}, []uint{10})
	newN, dueN = countPicks(picks)

	if len(picks) != 8 || dueN != 1 || newN != 7 {
		t.Fatalf("due-short: total=%d new=%d due=%d, want 8/7/1", len(picks), newN, dueN)
	}

	// 兩者皆不足 → session 較小，不 panic
	picks = planSRSSession(8, []uint{1}, []uint{10})
	if len(picks) != 2 {
		t.Fatalf("scarce: total=%d want 2", len(picks))
	}
}

func countPicks(picks []srsPick) (newN, dueN int) {
	for _, p := range picks {
		if p.isNew {
			newN++
		} else {
			dueN++
		}
	}

	return newN, dueN
}

// srsTestRepo 建一個帶例句的 fake repo
func srsTestRepo() *fakeLearnRepo {
	words := []*model.Word{
		{ID: 1, Word: "house", Tier: 1, Phonetic: "haʊs", DefinitionEN: "a building"},
		{ID: 2, Word: "water", Tier: 1, DefinitionEN: "liquid"},
		{ID: 3, Word: "music", Tier: 1, DefinitionEN: "sound art"},
		{ID: 4, Word: "happy", Tier: 1, DefinitionEN: "glad"},
	}
	sent := map[uint][]*model.LearnSentence{}

	for _, w := range words {
		sent[w.ID] = []*model.LearnSentence{{
			WordID: w.ID, Answer: w.Word,
			TextEN: "I like the {{}}.", TextZHTW: "翻譯:" + w.Word,
		}}
	}

	return &fakeLearnRepo{words: words, sentences: sent}
}

func TestSRSSessionAllNewWhenNoDue(t *testing.T) {
	repo := srsTestRepo()
	svc := NewLearnService(repo, nil, nil, NewMemoryLevelStore())

	ov, err := svc.SRSOverview(7)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	if ov.DueCount != 0 || ov.NewAvailable != 4 {
		t.Fatalf("overview: %+v want due0 new4", ov)
	}

	view, err := svc.CreateSRSSession(7, 4, "zh-tw")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if view.Total != 4 || view.NewCount != 4 {
		t.Fatalf("session: total=%d new=%d want 4/4", view.Total, view.NewCount)
	}

	for _, c := range view.Cards {
		if c.TextEN == "" || c.Length == 0 {
			t.Errorf("card missing text/length: %+v", c)
		}
	}
}

func TestSRSAnswerAdvancesAndReschedules(t *testing.T) {
	repo := srsTestRepo()
	svc := NewLearnService(repo, nil, nil, NewMemoryLevelStore())

	view, err := svc.CreateSRSSession(7, 4, "en")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// 找出 index 0 對應的字，取正確答案作答
	sess := loadSRSSession(t, svc, repo, view.SessionID)
	ans0 := sess.Items[0].Answer

	out, err := svc.AnswerSRS(7, view.SessionID, 0, ans0)
	if err != nil {
		t.Fatalf("answer correct: %v", err)
	}

	if !out.Correct || out.NextStage != 1 || out.XPAwarded == 0 {
		t.Fatalf("correct outcome: %+v", out)
	}

	// SRS 記錄：stage 進 1、排下次複習
	rec := repo.wordRecs[recKey(7, sess.Items[0].WordID)]
	if rec == nil || rec.SRSStage != 1 || rec.NextReviewAt == nil {
		t.Fatalf("srs record after correct: %+v", rec)
	}

	if d := time.Until(*rec.NextReviewAt); d < 20*time.Hour || d > 26*time.Hour {
		t.Errorf("stage1 next review should be ~1 day, got %v", d)
	}

	// 答錯另一張：stage 重置 0
	out, err = svc.AnswerSRS(7, view.SessionID, 1, "definitely-wrong")
	if err != nil {
		t.Fatalf("answer wrong: %v", err)
	}

	if out.Correct || out.NextStage != 0 || out.XPAwarded != 0 {
		t.Fatalf("wrong outcome: %+v", out)
	}

	// 重複作答同一張 → 拒絕
	if _, err := svc.AnswerSRS(7, view.SessionID, 0, ans0); !errors.Is(err, ErrLearnCardGraded) {
		t.Errorf("double answer should be ErrLearnCardGraded, got %v", err)
	}
}

func TestSRSCompletionAwardsXPAndStreak(t *testing.T) {
	repo := srsTestRepo()
	svc := NewLearnService(repo, nil, nil, NewMemoryLevelStore())

	view, err := svc.CreateSRSSession(7, 4, "en")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	sess := loadSRSSession(t, svc, repo, view.SessionID)

	var last *SRSAnswerOutcome

	for i := range sess.Items {
		out, err := svc.AnswerSRS(7, view.SessionID, i, sess.Items[i].Answer)
		if err != nil {
			t.Fatalf("answer %d: %v", i, err)
		}

		last = out
	}

	if !last.Completed || last.TotalXP == 0 {
		t.Fatalf("completion: %+v", last)
	}

	// 完成一場 = 今日有玩：streak 更新、週榜入帳
	if repo.stats[7] == nil || repo.stats[7].Streak == 0 {
		t.Errorf("streak not updated: %+v", repo.stats[7])
	}

	week := isoWeek(time.Now().UTC())
	if row := repo.weekly[weeklyKey(7, week)]; row == nil || row.XP != last.TotalXP {
		t.Errorf("weekly xp not recorded: %+v", repo.weekly)
	}
}

func TestSRSDueWordsPickedUp(t *testing.T) {
	repo := srsTestRepo()
	service := mkSRSSvc(repo)

	// 預先把 word 1 排成「昨天到期」的 stage2 複習卡
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	repo.wordRecs = map[string]*model.LearnWordRecord{
		recKey(7, 1): {UserID: 7, WordID: 1, SRSStage: 2, NextReviewAt: &yesterday},
	}

	ov, err := service.SRSOverview(7)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	if ov.DueCount != 1 || ov.NewAvailable != 3 { // word1 進輪替，剩 3 新字
		t.Fatalf("overview: %+v want due1 new3", ov)
	}

	// count=1、floor(1/4)=0 新字配額 → 全額給到期字，挑到 word1
	view, err := service.CreateSRSSession(7, 1, "en")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	sess := loadSRSSession(t, service, repo, view.SessionID)
	if len(sess.Items) != 1 || sess.Items[0].WordID != 1 || sess.Items[0].IsNew {
		t.Fatalf("expected the due word1, got %+v", sess.Items)
	}

	// 答對到期字 → stage 2→3，下次複習 +4 天
	out, err := service.AnswerSRS(7, view.SessionID, 0, sess.Items[0].Answer)
	if err != nil {
		t.Fatalf("answer: %v", err)
	}

	if out.NextStage != 3 {
		t.Fatalf("due word next stage = %d want 3", out.NextStage)
	}

	rec := repo.wordRecs[recKey(7, 1)]
	if d := time.Until(*rec.NextReviewAt); d < 3*24*time.Hour || d > 5*24*time.Hour {
		t.Errorf("stage3 next review should be ~4 days, got %v", d)
	}
}

func mkSRSSvc(repo *fakeLearnRepo) LearnService {
	return NewLearnService(repo, nil, nil, NewMemoryLevelStore())
}

func TestSRSWrongRequeuesUntilCorrect(t *testing.T) {
	repo := srsTestRepo()
	svc := NewLearnService(repo, nil, nil, NewMemoryLevelStore())

	view, err := svc.CreateSRSSession(7, 4, "en")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	sess := loadSRSSession(t, svc, repo, view.SessionID)
	wid := sess.Items[0].WordID
	ans := sess.Items[0].Answer

	// 答錯：不退場、不寫 DB、不完成
	out, err := svc.AnswerSRS(7, view.SessionID, 0, "wrong-on-purpose")
	if err != nil {
		t.Fatalf("wrong: %v", err)
	}

	if out.Correct || out.XPAwarded != 0 || out.Completed {
		t.Fatalf("wrong outcome should not retire/complete: %+v", out)
	}

	if repo.wordRecs[recKey(7, wid)] != nil {
		t.Errorf("wrong answer must not write SRS record yet")
	}

	// 同一張可再作答（未退場）；這次答對 → 退場，但因中途錯過 → 重置為 stage0（+1 天）
	out, err = svc.AnswerSRS(7, view.SessionID, 0, ans)
	if err != nil {
		t.Fatalf("retry correct: %v", err)
	}

	if !out.Correct || out.NextStage != 0 {
		t.Fatalf("lapsed retire should reset to stage0: %+v", out)
	}

	rec := repo.wordRecs[recKey(7, wid)]
	if rec == nil || rec.SRSStage != 0 || rec.WrongCount != 1 {
		t.Fatalf("lapsed record: %+v", rec)
	}

	// 退場後再作答 → 拒絕
	if _, err := svc.AnswerSRS(7, view.SessionID, 0, ans); !errors.Is(err, ErrLearnCardGraded) {
		t.Errorf("re-answer retired card should reject, got %v", err)
	}
}

// loadSRSSession 反序列化 Redis 內的 session（測試需要知道答案）
func loadSRSSession(
	t *testing.T, s LearnService, repo *fakeLearnRepo, id string,
) *srsSession {
	t.Helper()

	ls, ok := s.(*learnService)
	if !ok {
		t.Fatal("not *learnService")
	}

	env, err := loadEnvelope(ls.store, id)
	if err != nil {
		t.Fatalf("load envelope: %v", err)
	}

	var sess srsSession
	if err := json.Unmarshal(env.Data, &sess); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	return &sess
}
