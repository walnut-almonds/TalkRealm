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
        crossword: null,    // CrosswordView：{ level_id, tier, campaign?, rows, cols, letters, words[] }
        campaign: null,      // { total, furthest, levels: [{level_no, done, score?}] }
        campaignBoard: null, // LeaderboardView（關卡榜）
        weeklyBoard: null,   // LeaderboardView（週榜）
        boardScope: 'global', // 'global' | 'friends'
        srsOverview: null,   // { due_count, new_available }
        srs: null,           // SRSSessionView：{ session_id, total, new_count, cards[] }
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
        async startLevel(mode, tier, count = 0) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            this.crossword = null // 清掉另一模式的殘留狀態，避免畫面判斷式比對到舊資料
            try {
                this.level = await api.post(EP.LEARN_LEVELS, {
                    mode, tier, count, locale: getLocale(),
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
        async startCrossword(tier, count = 0) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            this.level = null // 清掉另一模式的殘留狀態，避免畫面判斷式比對到舊資料
            try {
                this.crossword = await api.post(EP.LEARN_CROSSWORD, { tier, count, locale: getLocale() })
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
                if (out.completed) {
                    this.loadStats()
                    if (this.crossword?.campaign) this.loadCampaign() // 首通進度/解鎖狀態刷新
                }
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
                if (out.completed) {
                    this.loadStats()
                    if (this.crossword?.campaign) this.loadCampaign() // 首通進度/解鎖狀態刷新
                }
                return out
            } catch (e) {
                if (String(e.message).includes('expired')) this.crossword = null
                this.error = e.message
                return null
            }
        },
        async loadCampaign() {
            try {
                this.campaign = await api.get(EP.LEARN_CAMPAIGN)
            } catch (e) { this.error = e.message }
        },
        async startCampaign(no) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            this.level = null // 清掉另一模式的殘留狀態，避免畫面判斷式比對到舊資料
            try {
                this.crossword = await api.post(EP.LEARN_CAMPAIGN_START(no), { locale: getLocale() })
            } catch (e) {
                this.error = e.message
            } finally {
                this.loading = false
            }
        },
        async loadBoards() {
            const q = this.boardScope === 'friends' ? '?scope=friends' : ''
            try {
                // 兩個榜一起刷新（hub 同畫面顯示，分開 catch 沒有意義）
                const [cb, wb] = await Promise.all([
                    api.get(EP.LEARN_CAMPAIGN_LB + q),
                    api.get(EP.LEARN_WEEKLY_LB + q),
                ])
                this.campaignBoard = cb
                this.weeklyBoard = wb
            } catch { /* 榜非關鍵，靜默失敗 */ }
        },
        setBoardScope(scope) {
            this.boardScope = scope
            this.loadBoards()
        },
        async loadSRSOverview() {
            try {
                this.srsOverview = await api.get(EP.LEARN_SRS_OVERVIEW)
            } catch { /* 非關鍵，靜默失敗 */ }
        },
        async startSRS(count) {
            this.loading = true
            this.error = ''
            this.lastOutcome = null
            // 清掉其他模式殘留，避免 playing 畫面判斷式比對到舊資料
            this.level = null
            this.crossword = null
            try {
                this.srs = await api.post(EP.LEARN_SRS_START, { count, locale: getLocale() })
            } catch (e) {
                this.error = e.message
            } finally {
                this.loading = false
            }
            return this.srs
        },
        async answerSRS(index, guess) {
            if (!this.srs) return null
            try {
                const out = await api.post(EP.LEARN_SRS_ANSWER(this.srs.session_id), { index, guess })
                if (out.completed) {
                    this.loadStats()
                    this.loadSRSOverview() // 到期數已變，刷新 hub 概況
                }
                return out
            } catch (e) {
                if (String(e.message).includes('expired')) this.srs = null
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
