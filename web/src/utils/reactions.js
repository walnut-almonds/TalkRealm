// 可用的表情回應。必須與後端 service.AllowedReactions 一致——
// 不在清單內的 emoji 後端會回 400。
export const REACTION_EMOJIS = ['👍', '🎉', '😂', '❤️', '😮', '😢']

/**
 * applyReaction 就地套用一則 message_reaction WS 事件。
 * 後端下發的是原始的 reaction 列（每列一個 user + emoji），前端不做聚合，
 * 所以這裡只需要增刪一列。兩個 store（guild / DM）共用同一份邏輯。
 */
export function applyReaction(msg, { user_id, emoji, action }) {
    if (!msg) return

    if (!Array.isArray(msg.reactions)) msg.reactions = []

    const idx = msg.reactions.findIndex(r => r.user_id === user_id && r.emoji === emoji)

    if (action === 'add') {
        if (idx === -1) msg.reactions.push({ user_id, emoji })
        return
    }

    if (idx !== -1) msg.reactions.splice(idx, 1)
}

/**
 * groupReactions 把原始 reaction 列聚成畫面要的 [{ emoji, count, mine }]，
 * 依 REACTION_EMOJIS 的順序排列，讓同一則訊息的表情順序永遠一致。
 */
export function groupReactions(reactions, myUserId) {
    if (!reactions?.length) return []

    const counts = new Map()

    for (const r of reactions) {
        const cur = counts.get(r.emoji) || { emoji: r.emoji, count: 0, mine: false }
        cur.count += 1
        if (r.user_id === myUserId) cur.mine = true
        counts.set(r.emoji, cur)
    }

    return REACTION_EMOJIS.filter(e => counts.has(e)).map(e => counts.get(e))
}
