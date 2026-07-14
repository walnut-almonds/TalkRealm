package service

import (
	"fmt"
	"math"
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
