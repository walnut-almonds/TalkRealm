// URL regex — matches http(s) URLs, stopped at whitespace or common trailing punctuation.
const URL_RE = /https?:\/\/[^\s<>"']+[^\s<>"'.,;:!?)\]]/g

/**
 * renderMarkdown converts a subset of Markdown to safe HTML.
 * Returns { html: string, urls: string[] } where urls is the
 * ordered list of unique http(s) URLs found in the message.
 */
export function renderMarkdown(rawText) {
    if (!rawText) return { html: '', urls: [] }

    // Split on fenced code blocks first, so we never process their internals.
    const parts = rawText.split(/(```[\s\S]*?```)/g)
    const urlSet = new Set()

    const html = parts.map((part, i) => {
        if (i % 2 === 1) {
            // Fenced code block
            let inner = part.slice(3, -3)
            const nl = inner.indexOf('\n')
            if (nl > 0 && /^\w+$/.test(inner.slice(0, nl).trim())) inner = inner.slice(nl + 1)
            return `<pre class="md-code-block"><code>${escapeHtml(inner)}</code></pre>`
        }

        // Process line-by-line so block-level elements (headings, blockquotes,
        // lists) are handled correctly, then apply inline rules.
        const lines = part.split('\n')
        let inList = false
        let inBlockquote = false
        let out = ''

        lines.forEach((rawLine, j) => {
            const isLast = j === lines.length - 1

            // --- Block-level: headings ---
            const headingMatch = rawLine.match(/^(#{1,3}) (.+)$/)
            if (headingMatch) {
                if (inList) { out += '</ul>'; inList = false }
                if (inBlockquote) { out += '</blockquote>'; inBlockquote = false }
                const level = headingMatch[1].length
                const text = applyInline(escapeHtml(headingMatch[2]), urlSet)
                out += `<h${level} class="md-h${level}">${text}</h${level}>`
                return
            }

            // --- Block-level: blockquote ---
            const bqMatch = rawLine.match(/^> ?(.*)$/)
            if (bqMatch) {
                if (inList) { out += '</ul>'; inList = false }
                if (!inBlockquote) { out += '<blockquote class="md-blockquote">'; inBlockquote = true }
                const text = applyInline(escapeHtml(bqMatch[1]), urlSet)
                out += text + (isLast ? '' : '<br>')
                return
            }

            if (inBlockquote) { out += '</blockquote>'; inBlockquote = false }

            // --- Block-level: unordered list ---
            if (/^- /.test(rawLine)) {
                if (!inList) { out += '<ul class="md-list">'; inList = true }
                out += `<li>${applyInline(escapeHtml(rawLine.slice(2)), urlSet)}</li>`
                return
            }

            if (inList) { out += '</ul>'; inList = false }

            // --- Plain paragraph line ---
            out += applyInline(escapeHtml(rawLine), urlSet) + (isLast ? '' : '<br>')
        })

        if (inList) out += '</ul>'
        if (inBlockquote) out += '</blockquote>'

        return out
    }).join('')

    return { html, urls: [...urlSet] }
}

/**
 * applyInline applies bold, italic, inline-code and URL link rules.
 * Detected URLs are added to the urlSet.
 */
function applyInline(escaped, urlSet) {
    // Bold before italic to avoid greedy overlap.
    escaped = escaped.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    escaped = escaped.replace(/\*(.+?)\*/g, '<em>$1</em>')
    escaped = escaped.replace(/`([^`\n]+)`/g, '<code class="md-inline-code">$1</code>')

    // URLs — convert to <a> links and record for OGP preview.
    // We need to work on the original (pre-escape) URL text, but since
    // escapeHtml only changes &, <, >, " in the text portions and URLs
    // shouldn't contain those, we can match directly on the escaped string.
    escaped = escaped.replace(URL_RE, (match) => {
        // Un-escape &amp; back to & in the href only.
        const href = match.replace(/&amp;/g, '&')
        urlSet.add(href)
        return `<a href="${escapeAttr(href)}" target="_blank" rel="noopener noreferrer" class="md-link">${match}</a>`
    })

    return escaped
}

function escapeHtml(text) {
    const d = document.createElement('div')
    d.textContent = text
    return d.innerHTML
}

function escapeAttr(str) {
    return str.replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}
