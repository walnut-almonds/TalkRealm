// scripts/buildwords/build.go
package main

import (
	"regexp"
	"strings"
)

var wordRe = regexp.MustCompile(`^[a-z]{3,8}$`)

// keepWord 過濾條件：純小寫字母、長度 3–8、有中文釋義
func keepWord(word, translation string) bool {
	return wordRe.MatchString(word) && strings.TrimSpace(translation) != ""
}

// tierOf 依 ECDICT tag 映射難度；無 tag 時以 COCA 詞頻補分級；回傳 0 = 捨棄
func tierOf(tag string, frq int) int {
	tags := strings.Fields(tag)
	// 多標籤取最低 tier（spec §1）
	best := 0

	for _, t := range tags {
		var tier int

		switch t {
		case "zk":
			tier = 1
		case "gk":
			tier = 2
		case "cet4", "cet6":
			tier = 3
		case "ky", "toefl", "ielts":
			tier = 4
		case "gre":
			tier = 5
		default:
			continue
		}

		if best == 0 || tier < best {
			best = tier
		}
	}

	if best != 0 {
		return best
	}

	// 無標籤者以詞頻補分級
	switch {
	case frq == 0:
		return 0
	case frq <= 2000:
		return 1
	case frq <= 5000:
		return 2
	case frq <= 10000:
		return 3
	case frq <= 20000:
		return 4
	default:
		return 0 // 太冷僻且無考試標籤，捨棄
	}
}
