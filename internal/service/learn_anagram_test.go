//nolint:testpackage // 白箱測試：需存取未匯出的 anagram 索引（sortLetters/buildAnagramIndex）
package service

import (
	"errors"
	"testing"

	"github.com/walnut-almonds/talkrealm/internal/model"
)

func TestSortLetters(t *testing.T) {
	if got := sortLetters("star"); got != "arst" {
		t.Errorf("sortLetters = %q", got)
	}
}

func TestSubWordIDs(t *testing.T) {
	idx := buildAnagramIndex([]*model.Word{
		{ID: 1, Word: "star"},
		{ID: 2, Word: "rat"},
		{ID: 3, Word: "art"},   // rat 的 anagram，同 signature
		{ID: 4, Word: "tar"},   // 同上
		{ID: 5, Word: "moon"},  // 拼不出（沒有 m/o/n）
		{ID: 6, Word: "stars"}, // 需要兩個 s，拼不出
	})

	got := idx.subWordIDs("star")

	want := map[uint]bool{1: true, 2: true, 3: true, 4: true}
	if len(got) != len(want) {
		t.Fatalf("got %v want ids of star/rat/art/tar", got)
	}

	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected id %d", id)
		}
	}
}

func TestWheelLevelFlow(t *testing.T) {
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

	lv, err := svc.CreateLevel(7, ModeWheel, 2, 0, "zh-tw")
	if err != nil {
		t.Fatalf("CreateLevel wheel: %v", err)
	}

	if lv.Letters == "" || len(lv.Slots) < 2 {
		t.Fatalf("bad wheel level: %+v", lv)
	}

	for _, s := range lv.Slots {
		if s.Word != "" || s.Definition != "" {
			t.Error("wheel slot leaked word/definition before solve")
		}
	}

	// 猜一個存在的字
	out, err := svc.Guess(7, lv.LevelID, &LearnGuessRequest{Word: rat})
	if err != nil {
		t.Fatalf("Guess: %v", err)
	}

	if !out.Correct || out.Word != rat || out.Definition == "" {
		t.Errorf("outcome: %+v", out)
	}

	// 同一字再猜 → ErrLearnSlotSolved
	if _, err := svc.Guess(7, lv.LevelID, &LearnGuessRequest{Word: rat}); !errors.Is(
		err,
		ErrLearnSlotSolved,
	) {
		t.Errorf("expected ErrLearnSlotSolved, got %v", err)
	}

	// 不存在的字 → correct=false
	out, _ = svc.Guess(7, lv.LevelID, &LearnGuessRequest{Word: "zzz"})
	if out == nil || out.Correct {
		t.Errorf("zzz should be wrong: %+v", out)
	}
}
