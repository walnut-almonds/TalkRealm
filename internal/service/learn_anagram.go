package service

import (
	"sort"
	"strings"

	"github.com/walnut-almonds/talkrealm/internal/model"
)

// anagramIndex 排序字母簽名 → word IDs（啟動後常駐記憶體，5 萬字約數 MB）
type anagramIndex struct {
	sig   map[string][]uint
	words map[uint]*model.Word
}

func sortLetters(w string) string {
	b := []byte(w)
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })

	return string(b)
}

func buildAnagramIndex(words []*model.Word) *anagramIndex {
	idx := &anagramIndex{
		sig:   make(map[string][]uint, len(words)),
		words: make(map[uint]*model.Word, len(words)),
	}

	for _, w := range words {
		s := sortLetters(w.Word)
		idx.sig[s] = append(idx.sig[s], w.ID)
		idx.words[w.ID] = w
	}

	return idx
}

// subWordIDs 回傳 base 的字母（multiset）能拼出的所有字 ID（含 base 自身；≥3 字母）
// 做法：枚舉 base 排序字母的所有子序列簽名（2^len ≤ 256），查索引。
func (idx *anagramIndex) subWordIDs(base string) []uint {
	sorted := sortLetters(strings.ToLower(base))
	n := len(sorted)
	seen := map[string]bool{}

	var ids []uint

	for mask := 0; mask < (1 << n); mask++ {
		var sb strings.Builder

		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				sb.WriteByte(sorted[i])
			}
		}

		s := sb.String()
		if len(s) < 3 || seen[s] {
			continue
		}

		seen[s] = true
		ids = append(ids, idx.sig[s]...)
	}

	return ids
}
