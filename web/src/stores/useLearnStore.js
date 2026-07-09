import { defineStore } from 'pinia'
import { api, EP } from '@/api/index.js'
import { getLocale } from '@/i18n/index.js'

export const useLearnStore = defineStore('learn', {
    state: () => ({
        level: null,        // LevelView：{ level_id, mode, tier, slots[] }
        stats: null,        // { xp, streak, words_learned }
        lastOutcome: null,  // 最近一次 GuessOutcome
        loading: false,
        error: '',
    }),
    actions: {
        async startLevel(mode, tier) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            try {
                this.level = await api.post(EP.LEARN_LEVELS, {
                    mode, tier, locale: getLocale(),
                })
            } catch (e) {
                this.error = e.message
            } finally {
                this.loading = false
            }
        },
        async guess(slot, word) {
            if (!this.level) return null
            try {
                const out = await api.post(EP.LEARN_GUESS(this.level.level_id), { slot, word })
                this.lastOutcome = out
                if (out.correct) {
                    // wheel 模式送 slot=-1，後端在 out.slot 回傳實際命中的格
                    const s = this.level.slots[out.slot]
                    if (s) {
                        s.solved = true
                        s.word = out.word
                        s.masked = out.word
                        s.definition = out.definition || s.definition
                    }
                }
                if (out.completed) this.loadStats()
                return out
            } catch (e) {
                // 410 = 關卡過期
                if (String(e.message).includes('expired')) this.level = null
                this.error = e.message
                return null
            }
        },
        async loadStats() {
            try {
                this.stats = await api.get(EP.LEARN_STATS)
            } catch { /* stats 非關鍵，靜默失敗 */ }
        },
    },
})
