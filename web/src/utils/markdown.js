export function renderMarkdown(rawText) {
    if (!rawText) return ''
    // split on fenced code blocks
    const parts = rawText.split(/(```[\s\S]*?```)/g)
    return parts.map((part, i) => {
        if (i % 2 === 1) {
            let inner = part.slice(3, -3)
            const nl = inner.indexOf('\n')
            if (nl > 0 && /^\w+$/.test(inner.slice(0, nl).trim())) inner = inner.slice(nl + 1)
            return `<pre class="md-code-block"><code>${escapeHtml(inner)}</code></pre>`
        }
        let html = escapeHtml(part)
        html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
        html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
        html = html.replace(/`([^`\n]+)`/g, '<code class="md-inline-code">$1</code>')
        const lines = html.split('\n')
        let inList = false, out = ''
        lines.forEach((line, j) => {
            if (/^- /.test(line)) {
                if (!inList) { out += '<ul class="md-list">'; inList = true }
                out += `<li>${line.slice(2)}</li>`
            } else {
                if (inList) { out += '</ul>'; inList = false }
                out += line + (j < lines.length - 1 ? '<br>' : '')
            }
        })
        if (inList) out += '</ul>'
        return out
    }).join('')
}

function escapeHtml(text) {
    const d = document.createElement('div')
    d.textContent = text
    return d.innerHTML
}
