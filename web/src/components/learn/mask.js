// masked 字串轉顯示格。
// 普通模式：一字一格（'_' 為空格子，格數 = 字長）。
// 困難模式：連續 '_' 收成單一 gap 格，不洩漏缺幾個字母。
export function maskSegments(masked, hard) {
    if (!hard) return [...(masked || '')].map(ch => ({ ch: ch === '_' ? '' : ch, gap: false }))

    const segs = []
    for (const ch of masked || '') {
        if (ch === '_') {
            if (!segs.length || !segs[segs.length - 1].gap) segs.push({ ch: '', gap: true })
        } else {
            segs.push({ ch, gap: false })
        }
    }
    return segs
}
