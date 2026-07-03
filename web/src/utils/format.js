export function formatTimestamp(timestamp) {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now - date
    const diffMins = Math.floor(diffMs / 60000)
    if (diffMins < 1) return '剛剛'
    if (diffMins < 60) return `${diffMins} 分鐘前`
    const diffHours = Math.floor(diffMins / 60)
    if (diffHours < 24) return `${diffHours} 小時前`
    const diffDays = Math.floor(diffHours / 24)
    if (diffDays < 7) return `${diffDays} 天前`
    return date.toLocaleDateString('zh-TW')
}

export function formatFullTimestamp(timestamp) {
    return new Date(timestamp).toLocaleString('zh-TW')
}

export function formatFileSize(bytes) {
    if (!bytes) return '0 B'
    const k = 1024, sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

export function escapeHtml(text) {
    const d = document.createElement('div')
    d.textContent = String(text ?? '')
    return d.innerHTML
}

export function getStatusText(t, status) {
    switch (status) {
        case 'online': return t('status.online')
        case 'idle': return t('status.idle')
        case 'dnd': return t('status.dnd')
        case 'busy': return t('status.busy')
        case 'away': return t('status.away')
        case 'invisible': return t('status.invisible') // 僅自己看得到（後端對他人一律回 offline）
        default: return t('status.offline')
    }
}

export function randomUUID() {
    if (typeof crypto.randomUUID === 'function') return crypto.randomUUID()
    return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, c =>
        (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16)
    )
}

export const IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml'])
