import { ref, onUnmounted } from 'vue'

// ponytail: 瀏覽器原生 SpeechRecognition（Chrome/Edge 走 webkit 前綴），免依賴免後端。
// Safari/Firefox 支援不齊，supported=false 時前端隱藏麥克風鈕退回打字。
const Ctor = typeof window !== 'undefined'
    ? (window.SpeechRecognition || window.webkitSpeechRecognition)
    : null

export function useSpeechInput(lang = 'en-US') {
    const supported = !!Ctor
    const listening = ref(false)
    let rec = null

    // start(onResult)：辨識到一句就回呼 transcript，然後自動停止（單次聽寫）
    function start(onResult) {
        if (!supported || listening.value) return

        rec = new Ctor()
        rec.lang = lang
        rec.interimResults = false
        rec.maxAlternatives = 1

        rec.onresult = (e) => {
            const text = e.results?.[0]?.[0]?.transcript?.trim()
            if (text) onResult(text)
        }
        rec.onend = () => { listening.value = false }
        rec.onerror = () => { listening.value = false }

        listening.value = true
        try {
            rec.start()
        } catch {
            listening.value = false // 連續 start 會 throw，靜默忽略
        }
    }

    function stop() {
        if (rec && listening.value) rec.stop()
    }

    onUnmounted(stop)

    return { supported, listening, start, stop }
}
