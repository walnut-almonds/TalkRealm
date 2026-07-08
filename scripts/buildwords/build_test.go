// scripts/buildwords/build_test.go
package main

import "testing"

func TestTierOf(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		frq  int
		want int
	}{
		{"zk maps to 1", "zk gk cet4", 0, 1},
		{"gk maps to 2", "gk cet4", 0, 2},
		{"cet4 maps to 3", "cet4 cet6", 0, 3},
		{"toefl maps to 4", "toefl ielts", 0, 4},
		{"gre maps to 5", "gre", 0, 5},
		{"no tag, high freq -> 1", "", 1500, 1},
		{"no tag, mid freq -> 3", "", 8000, 3},
		{"no tag, low freq -> drop (0)", "", 99999, 0},
		{"no tag, no freq -> drop", "", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tierOf(tt.tag, tt.frq); got != tt.want {
				t.Errorf("tierOf(%q,%d)=%d want %d", tt.tag, tt.frq, got, tt.want)
			}
		})
	}
}

func TestKeepWord(t *testing.T) {
	tests := []struct {
		word, translation string
		want              bool
	}{
		{"apple", "n. 蘋果", true},
		{"ab", "太短", false},        // len < 3
		{"abcdefghi", "太長", false}, // len > 8
		{"o'clock", "有符號", false},  // 非純字母
		{"Apple", "大寫", false},     // 只收小寫
		{"apple", "", false},       // 無中文釋義
	}
	for _, tt := range tests {
		if got := keepWord(tt.word, tt.translation); got != tt.want {
			t.Errorf("keepWord(%q,%q)=%v want %v", tt.word, tt.translation, got, tt.want)
		}
	}
}
