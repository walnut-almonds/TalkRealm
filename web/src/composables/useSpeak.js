// ponytail: 瀏覽器原生 speechSynthesis，免依賴免後端；語音包安裝與否看裝置，
// 不支援時 speak() 靜默不做事（呼叫端無需另外判斷）
const supported = typeof window !== 'undefined' && 'speechSynthesis' in window

/**
 * useSpeak — 唸出單字（en-US）。呼叫端務必只對已解出的答案顯示發音按鈕：
 * 對未解字唸出等於繞過提示系統的三階梯，直接洩漏答案。
 */
export function useSpeak() {
    function speak(word) {
        if (!supported || !word) return

        window.speechSynthesis.cancel() // 連續點擊時取消前一句，不要排隊唸

        const u = new SpeechSynthesisUtterance(word)
        u.lang = 'en-US'
        window.speechSynthesis.speak(u)
    }

    return { supported, speak }
}
