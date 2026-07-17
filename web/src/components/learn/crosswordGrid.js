// 從後端回傳的 words[] 陣列（含已解字的完整拼寫、網格座標）推導出可渲染的 2D 格子狀態。
// 交叉格提前顯示：多個已解字寫入同一格時字母必然相同（後端排版保證無衝突），故不處理衝突。
export function buildCells(words, rows, cols) {
    const cells = Array.from({ length: rows }, () => Array.from({ length: cols }, () => null))

    for (const [i, w] of words.entries()) {
        if (!w.dir) continue // bonus 字，不在網格上

        for (let k = 0; k < w.length; k++) {
            const r = w.dir === 'h' ? w.row : w.row + k
            const c = w.dir === 'h' ? w.col + k : w.col
            if (!cells[r][c]) cells[r][c] = { letter: '', words: [] }
            cells[r][c].words.push(i) // 交叉格會屬於兩個字，hover 時兩條都高亮
        }
    }

    for (const w of words) {
        if (!w.dir || !w.masked) continue

        for (let k = 0; k < w.length; k++) {
            if (w.masked[k] === '_') continue

            const r = w.dir === 'h' ? w.row : w.row + k
            const c = w.dir === 'h' ? w.col + k : w.col
            cells[r][c].letter = w.masked[k]
        }
    }

    return cells
}
