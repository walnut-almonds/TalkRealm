import { defineStore } from 'pinia'
import { api, EP } from '@/api/index.js'
import { getLocale } from '@/i18n/index.js'

const HARD_MODE_KEY = 'talkrealm_learn_hard'

export const useLearnStore = defineStore('learn', {
    state: () => ({
        level: null,        // LevelView：{ level_id, mode, tier, slots[] }
        stats: null,        // { xp, streak, words_learned }
        lastOutcome: null,  // 最近一次 GuessOutcome
        daily: null,        // { date, played, score?, level? }
        leaderboard: null,  // { date, top[], me? }
        crossword: null,    // CrosswordView：{ level_id, tier, rows, cols, letters, words[] }
        loading: false,
        error: '',
        // 純本機顯示偏好：隱藏底線數量（不影響計分）
        hardMode: localStorage.getItem(HARD_MODE_KEY) === '1',
    }),
    actions: {
        setHardMode(v) {
            this.hardMode = v
            localStorage.setItem(HARD_MODE_KEY, v ? '1' : '0')
        },
        async startLevel(mode, tier) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            this.crossword = null // 清掉另一模式的殘留狀態，避免畫面判斷式比對到舊資料
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
                if (out.completed) {
                    this.loadStats()
                    if (this.daily) {
                        this.loadDaily()
                        this.loadLeaderboard()
                    }
                }
                return out
            } catch (e) {
                // 410 = 關卡過期
                if (String(e.message).includes('expired')) this.level = null
                this.error = e.message
                return null
            }
        },
        async hint(slot) {
            if (!this.level) return null
            try {
                const out = await api.post(EP.LEARN_HINT(this.level.level_id), { slot })
                const s = this.level.slots[out.slot]
                if (s) {
                    s.hint_tier = out.tier
                    if (out.masked) s.masked = out.masked
                    if (out.definition) s.definition = out.definition
                }
                return out
            } catch (e) {
                this.error = e.message
                return null
            }
        },
        async reveal(slot) {
            if (!this.level) return null
            try {
                const out = await api.post(EP.LEARN_REVEAL(this.level.level_id), { slot })
                const s = this.level.slots[out.slot]
                if (s) {
                    s.solved = true
                    s.word = out.word
                    s.masked = out.word
                    s.definition = out.definition
                }
                if (out.completed) {
                    this.loadStats()
                    if (this.daily) {
                        this.loadDaily()
                        this.loadLeaderboard()
                    }
                }
                return out
            } catch (e) {
                if (String(e.message).includes('expired')) this.level = null
                this.error = e.message
                return null
            }
        },
        async startCrossword(tier) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            this.level = null // 清掉另一模式的殘留狀態，避免畫面判斷式比對到舊資料
            try {
                this.crossword = await api.post(EP.LEARN_CROSSWORD, { tier, locale: getLocale() })
            } catch (e) {
                this.error = e.message
            } finally {
                this.loading = false
            }
        },
        async guessCrossword(word) {
            if (!this.crossword) return null
            try {
                const out = await api.post(EP.LEARN_GUESS(this.crossword.level_id), { slot: -1, word })
                this.lastOutcome = out
                if (out.correct) {
                    const s = this.crossword.words[out.slot]
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
                if (String(e.message).includes('expired')) this.crossword = null
                this.error = e.message
                return null
            }
        },
        async hintCrossword(slot) {
            if (!this.crossword) return null
            try {
                const out = await api.post(EP.LEARN_HINT(this.crossword.level_id), { slot })
                const s = this.crossword.words[out.slot]
                if (s) {
                    s.hint_tier = out.tier
                    if (out.masked) s.masked = out.masked
                    if (out.definition) s.definition = out.definition
                }
                return out
            } catch (e) {
                this.error = e.message
                return null
            }
        },
        async revealCrossword(slot) {
            if (!this.crossword) return null
            try {
                const out = await api.post(EP.LEARN_REVEAL(this.crossword.level_id), { slot })
                const s = this.crossword.words[out.slot]
                if (s) {
                    s.solved = true
                    s.word = out.word
                    s.masked = out.word
                    s.definition = out.definition
                }
                if (out.completed) this.loadStats()
                return out
            } catch (e) {
                if (String(e.message).includes('expired')) this.crossword = null
                this.error = e.message
                return null
            }
        },
        async loadStats() {
            try {
                this.stats = await api.get(EP.LEARN_STATS)
            } catch { /* stats 非關鍵，靜默失敗 */ }
        },
        async loadDaily() {
            try {
                this.daily = await api.get(`${EP.LEARN_DAILY}?locale=${getLocale()}`)
            } catch (e) { this.error = e.message }
        },
        async startDaily() {
            await this.loadDaily()
            if (this.daily && !this.daily.played) {
                this.level = this.daily.level
                this.crossword = null // 清掉另一模式的殘留狀態，避免畫面判斷式比對到舊資料
                this.lastOutcome = null
                return true
            }
            return false
        },
        async loadLeaderboard() {
            try {
                this.leaderboard = await api.get(EP.LEARN_LEADERBOARD)
            } catch { /* 非關鍵 */ }
        },
    },
})
