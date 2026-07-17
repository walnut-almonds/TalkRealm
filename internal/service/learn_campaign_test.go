//nolint:testpackage // 白箱測試：需存取未匯出的生成邏輯與常數
package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
)

type fakeFriendLookup struct{ ids []uint }

func (f *fakeFriendLookup) FriendIDs(uint) ([]uint, error) { return f.ids, nil }

func TestEnsureCampaignLevelsIdempotent(t *testing.T) {
	svc, repo := newTestService(wheelAnagramWords())

	created, err := svc.EnsureCampaignLevels()
	if err != nil {
		t.Fatalf("EnsureCampaignLevels: %v", err)
	}

	if created != campaignLevelCount || len(repo.campaign) != campaignLevelCount {
		t.Fatalf("created=%d stored=%d want %d", created, len(repo.campaign), campaignLevelCount)
	}

	for no, l := range repo.campaign {
		if l.Tier != campaignTier || l.Puzzle == "" {
			t.Errorf("level %d: tier=%d puzzle empty=%v", no, l.Tier, l.Puzzle == "")
		}
	}

	// 冪等：再跑一次不重生任何關卡
	created, err = svc.EnsureCampaignLevels()
	if err != nil {
		t.Fatalf("EnsureCampaignLevels 2nd: %v", err)
	}

	if created != 0 {
		t.Errorf("2nd run created %d, want 0", created)
	}
}

func TestStartCampaignLevelLockGate(t *testing.T) {
	svc, _ := newTestService(wheelAnagramWords())

	if _, err := svc.EnsureCampaignLevels(); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	// 前一關未通，第 2 關上鎖
	if _, err := svc.StartCampaignLevel(7, 2, "en"); !errors.Is(err, ErrLearnCampaignLocked) {
		t.Fatalf("level 2 should be locked, got %v", err)
	}

	// 不存在的關卡
	if _, err := svc.StartCampaignLevel(7, 999, "en"); !errors.Is(err, ErrLearnLevelNotFound) {
		t.Fatalf("level 999 should be not found, got %v", err)
	}

	view, err := svc.StartCampaignLevel(7, 1, "en")
	if err != nil {
		t.Fatalf("StartCampaignLevel 1: %v", err)
	}

	if view.Campaign != 1 || view.LevelID == "" || len(view.Words) < 2 {
		t.Fatalf("bad view: %+v", view)
	}

	for _, w := range view.Words {
		if w.Word != "" {
			t.Error("answer leaked before solve")
		}
	}
}

// clearLevel 用字表逐字猜完整關（不在題內的字只是猜錯，無副作用）
func clearLevel(t *testing.T, svc LearnService, uid uint, levelID string) *GuessOutcome {
	t.Helper()

	var last *GuessOutcome

	for _, w := range wheelAnagramWords() {
		out, err := svc.Guess(uid, levelID, &LearnGuessRequest{Slot: -1, Word: w.Word})
		if err != nil {
			t.Fatalf("guess %s: %v", w.Word, err)
		}

		if out.Correct {
			last = out
		}
	}

	if last == nil || !last.Completed {
		t.Fatalf("level not completed: %+v", last)
	}

	return last
}

func TestCampaignFirstClearProgressAndWeekly(t *testing.T) {
	svc, repo := newTestService(wheelAnagramWords())

	if _, err := svc.EnsureCampaignLevels(); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	view, err := svc.StartCampaignLevel(7, 1, "en")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	out := clearLevel(t, svc, 7, view.LevelID)
	if out.TotalXP <= 0 {
		t.Fatalf("expected positive total xp, got %d", out.TotalXP)
	}

	// 首通進度入帳
	ov, err := svc.CampaignOverview(7)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	if ov.Furthest != 1 || ov.Total != campaignLevelCount || !ov.Levels[0].Done {
		t.Fatalf("overview: %+v", ov)
	}

	firstScore := ov.Levels[0].Score
	if firstScore != out.TotalXP {
		t.Errorf("progress score %d != total xp %d", firstScore, out.TotalXP)
	}

	// 週榜 XP 入帳
	week := isoWeek(time.Now().UTC())
	if row := repo.weekly[weeklyKey(7, week)]; row == nil || row.XP != out.TotalXP {
		t.Fatalf("weekly xp not recorded: %+v", repo.weekly)
	}

	// 通關後第 2 關解鎖
	if _, err := svc.StartCampaignLevel(7, 2, "en"); err != nil {
		t.Fatalf("level 2 should unlock after clearing 1: %v", err)
	}

	// 重玩第 1 關：可玩，但首通分數不覆寫（重玩不刷榜）
	replay, err := svc.StartCampaignLevel(7, 1, "en")
	if err != nil {
		t.Fatalf("replay start: %v", err)
	}

	clearLevel(t, svc, 7, replay.LevelID)

	ov2, _ := svc.CampaignOverview(7)
	if ov2.Levels[0].Score != firstScore {
		t.Errorf("replay overwrote first-clear score: %d -> %d", firstScore, ov2.Levels[0].Score)
	}

	// 重玩 XP 仍計入週榜（週榜量投入）
	if row := repo.weekly[weeklyKey(7, week)]; row.XP <= firstScore {
		t.Errorf("replay xp not accumulated into weekly: %d", row.XP)
	}
}

func weeklyKey(uid uint, week string) string {
	return fmt.Sprintf("%d:%s", uid, week)
}

func TestCampaignAndWeeklyLeaderboards(t *testing.T) {
	repo := &fakeLearnRepo{words: wheelAnagramWords()}
	friends := &fakeFriendLookup{ids: []uint{}} // user 7 沒有好友
	svc := NewLearnService(repo, nil, friends, NewMemoryLevelStore())

	mustProgress := func(uid uint, no, score int) {
		if _, err := repo.CreateCampaignProgress(&model.LearnCampaignProgress{
			UserID: uid, LevelNo: no, Score: score,
		}); err != nil {
			t.Fatalf("progress: %v", err)
		}
	}

	mustProgress(7, 1, 10)
	mustProgress(9, 1, 8)
	mustProgress(9, 2, 12)

	// 全球榜：user 9 總分 20（最遠 2）> user 7 總分 10
	lb, err := svc.CampaignLeaderboard(7, false)
	if err != nil {
		t.Fatalf("campaign lb: %v", err)
	}

	if len(lb.Top) != 2 || lb.Top[0].UserID != 9 || lb.Top[0].Score != 20 || lb.Top[0].Level != 2 {
		t.Fatalf("global top: %+v", lb.Top)
	}

	if lb.Me == nil || lb.Me.Rank != 2 || lb.Me.Score != 10 || lb.Me.Level != 1 {
		t.Fatalf("global me: %+v", lb.Me)
	}

	// 好友榜（無好友 = 只有自己）：rank 1
	lb, err = svc.CampaignLeaderboard(7, true)
	if err != nil {
		t.Fatalf("campaign friends lb: %v", err)
	}

	if len(lb.Top) != 1 || lb.Top[0].UserID != 7 || lb.Me.Rank != 1 {
		t.Fatalf("friends board: top=%+v me=%+v", lb.Top, lb.Me)
	}

	// 週榜：累加 + 排序
	week := isoWeek(time.Now().UTC())

	for _, xp := range []int{20, 10} { // 兩次完關累加
		if err := repo.UpsertWeeklyXP(7, week, xp); err != nil {
			t.Fatalf("weekly upsert: %v", err)
		}
	}

	if err := repo.UpsertWeeklyXP(9, week, 50); err != nil {
		t.Fatalf("weekly upsert: %v", err)
	}

	wb, err := svc.WeeklyLeaderboard(7, false)
	if err != nil {
		t.Fatalf("weekly lb: %v", err)
	}

	if wb.Week != week || len(wb.Top) != 2 || wb.Top[0].UserID != 9 || wb.Top[0].Score != 50 {
		t.Fatalf("weekly top: week=%s %+v", wb.Week, wb.Top)
	}

	if wb.Me == nil || wb.Me.Rank != 2 || wb.Me.Score != 30 {
		t.Fatalf("weekly me: %+v", wb.Me)
	}
}
