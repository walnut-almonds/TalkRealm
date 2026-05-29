import { defineStore } from 'pinia'
import { ref, computed, reactive } from 'vue'
import { api, STORAGE_KEYS, guildLastChannel } from '@/api/index.js'
import { useVoiceStore } from './useVoiceStore.js'

export const useAppStore = defineStore('app', () => {
    // ── State ──────────────────────────────────────────────────
    const user = ref(null)
    const guilds = ref([])
    const currentGuild = ref(null)
    const channels = ref([])
    const currentChannel = ref(null)
    const members = ref([])
    const messages = ref([])
    const isLoading = ref(false)
    const notification = ref({ message: '', type: 'info', visible: false })
    const pendingFileIds = ref([])
    const lightboxUrl = ref(null)

    // Online user IDs (global, populated by presence_update WS events)
    const onlineUserIds = reactive(new Set())

    // Unread guild IDs: guilds that received messages while we weren't viewing them
    const unreadGuildIds = reactive(new Set())

    // channelId → guildId map (populated when guild channels are loaded)
    const channelGuildMap = new Map()

    // Translation cache: messageId → { original_lang, translations: {zh, ja, en} }
    const translationCache = reactive(new Map())
    // messageIds whose translation is in-flight (show loading indicator)
    const translationLoadingSet = reactive(new Set())

    // Typing: { [channelId]: { [userId]: { username, ts } } }
    const typingUsers = ref({})

    // Image URL cache: fileId → url
    const imageUrlCache = new Map()
    const imageUrlInFlight = new Set()

    // User cache for message authors
    const userCache = new Map()

    // ── Computed ────────────────────────────────────────────────
    const textChannels = computed(() => channels.value.filter(c => c.type === 'text'))
    const voiceChannels = computed(() => channels.value.filter(c => c.type === 'voice'))
    const isAuthenticated = computed(() => !!user.value)
    const currentUserRole = computed(() => {
        if (!user.value || !members.value.length) return null
        return members.value.find(m => m.user_id === user.value.id)?.role ?? null
    })

    // Typing list for current channel (exclude self, expire after 5s)
    const typingList = computed(() => {
        if (!currentChannel.value) return []
        const ch = typingUsers.value[currentChannel.value.id] || {}
        const now = Date.now()
        return Object.values(ch).filter(t => now - t.ts < 5000)
    })

    // ── Notifications ───────────────────────────────────────────
    let notifTimer = null
    function showNotification(message, type = 'info') {
        notification.value = { message, type, visible: true }
        clearTimeout(notifTimer)
        notifTimer = setTimeout(() => { notification.value.visible = false }, 3000)
    }

    // ── Loading ─────────────────────────────────────────────────
    function setLoading(val) { isLoading.value = val }

    // ── User Cache ──────────────────────────────────────────────
    function cacheUser(u) { if (u?.id) userCache.set(u.id, u) }

    function resolveMessageUser(msg) {
        if (msg.user?.id) return msg.user
        const uid = msg.user_id
        if (userCache.has(uid)) return userCache.get(uid)
        const m = members.value.find(m => m.user_id === uid)
        if (m?.user) { cacheUser(m.user); return m.user }
        if (user.value?.id === uid) { cacheUser(user.value); return user.value }
        fetchPublicUser(uid)
        return null
    }

    async function fetchPublicUser(userId) {
        if (userCache.has(`pending:${userId}`)) return
        userCache.set(`pending:${userId}`, true)
        try {
            const u = await api.getPublicUser(userId)
            cacheUser(u)
        } catch {
            cacheUser({ id: userId, username: 'Unknown', nickname: 'Unknown', avatar: null })
        } finally {
            userCache.delete(`pending:${userId}`)
        }
    }

    // ── Image URL Cache ─────────────────────────────────────────
    async function getImageUrl(fileId, force = false) {
        if (!force) {
            const cached = imageUrlCache.get(fileId)
            if (cached) return cached
        }
        if (imageUrlInFlight.has(fileId)) return null
        imageUrlInFlight.add(fileId)
        try {
            const resp = await api.getFileDownloadUrl(fileId)
            if (resp?.url) {
                imageUrlCache.set(fileId, resp.url)
                return resp.url
            }
        } catch (e) {
            console.warn('Failed to get image URL:', fileId, e)
        } finally {
            imageUrlInFlight.delete(fileId)
        }
        return null
    }

    // ── Auth ────────────────────────────────────────────────────
    async function checkAuth() {
        // OAuth callback
        const rawSearch = window.location.search || window.location.hash.replace(/^#/, '?')
        const params = new URLSearchParams(rawSearch)
        const oauthToken = params.get('access_token')
        const oauthRefresh = params.get('refresh_token')
        if (oauthToken) {
            api.setToken(oauthToken)
            if (oauthRefresh) api.setRefreshToken(oauthRefresh)
            window.history.replaceState({}, '', window.location.pathname)
            await loadUserData()
            return
        }
        const token = localStorage.getItem(STORAGE_KEYS.TOKEN)
        if (token) {
            api.setToken(token)
            await loadUserData()
        }
    }

    async function loadUserData() {
        try {
            setLoading(true)
            const res = await api.getCurrentUser()
            user.value = res.user
            cacheUser(res.user)
            await loadGuilds()

            const lastGuildId = localStorage.getItem(STORAGE_KEYS.LAST_GUILD)
            if (lastGuildId) {
                const g = guilds.value.find(g => g.id === parseInt(lastGuildId))
                if (g) {
                    // selectGuild 內部已自動恢復最後停留的頻道
                    await selectGuild(g.id)
                }
            }
        } catch (e) {
            console.error('loadUserData failed', e)
            handleLogout()
        } finally {
            setLoading(false)
        }
    }

    async function handleLogout() {
        await api.logout().catch(() => { })
        user.value = null
        guilds.value = []
        currentGuild.value = null
        channels.value = []
        currentChannel.value = null
        members.value = []
        messages.value = []
    }

    // ── Guild ────────────────────────────────────────────────────
    async function loadGuilds() {
        const res = await api.getMyGuilds()
        guilds.value = Array.isArray(res) ? res : (res.guilds || [])
    }

    async function selectGuild(guildId) {
        try {
            setLoading(true)
            const guild = await api.getGuild(guildId)
            currentGuild.value = guild

            const [chRes, membersRes] = await Promise.all([
                api.getGuildChannels(guildId),
                api.getGuildMembers(guildId),
            ])
            channels.value = Array.isArray(chRes) ? chRes : (chRes.channels || [])
            members.value = Array.isArray(membersRes) ? membersRes : (membersRes.members || [])

            // Populate channelId→guildId map for unread tracking
            channels.value.forEach(c => { if (c.guild_id) channelGuildMap.set(c.id, c.guild_id) })
            // Clear unread for this guild
            unreadGuildIds.delete(guildId)

            userCache.clear()
            members.value.forEach(m => { if (m.user) cacheUser(m.user) })
            if (user.value) cacheUser(user.value)

            localStorage.setItem(STORAGE_KEYS.LAST_GUILD, guildId)

            // 切換群組時先清空頻道狀態，避免殘留上一個群組的頻道畫面
            currentChannel.value = null
            messages.value = []

            // 恢復此群組最後停留的文字頻道，fallback 到第一個文字頻道
            const allTextChannels = channels.value.filter(c => c.type === 'text')
            const lastChId = guildLastChannel.get(guildId)
            const targetChannel = (lastChId && allTextChannels.find(c => c.id === lastChId))
                || allTextChannels[0]
            if (targetChannel) {
                await selectChannel(targetChannel.id)
            }

            // 載入各語音頻道目前的成員（進入前可預覽誰在裡面）
            const voiceStore = useVoiceStore()
            const voiceChannels = channels.value.filter(c => c.type === 'voice')
            await Promise.all(voiceChannels.map(async ch => {
                try {
                    const res = await api.getVoiceParticipants(ch.id)
                    const participants = res.participants || []
                    // Clear & repopulate
                    voiceStore.voiceParticipants[ch.id] = []
                    participants.forEach(p => {
                        voiceStore.upsertParticipant(ch.id, p.user_id, p.username)
                        voiceStore.upsertParticipantState(ch.id, p.user_id, {
                            user_id: p.user_id,
                            username: p.username,
                            mic_enabled: true,
                            deafened: false,
                        })
                    })
                } catch { /* 靜默失敗，非致命 */ }
            }))
        } catch (e) {
            console.error('selectGuild failed', e)
            showNotification('載入社群失敗', 'error')
        } finally {
            setLoading(false)
        }
    }

    // ── Channel ──────────────────────────────────────────────────
    async function selectChannel(channelId) {
        try {
            setLoading(true)
            const channel = await api.getChannel(channelId)
            currentChannel.value = channel
            messages.value = []
            await loadMessages(channelId)
            // 記錄此頻道為該 guild 最後停留的頻道
            if (currentGuild.value) {
                guildLastChannel.set(currentGuild.value.id, channelId)
            }
        } catch (e) {
            console.error('selectChannel failed', e)
            showNotification('載入頻道失敗', 'error')
        } finally {
            setLoading(false)
        }
    }

    // ── Messages ─────────────────────────────────────────────────
    async function loadMessages(channelId, before = null) {
        try {
            const res = await api.getChannelMessages(channelId, 50, before)
            const msgs = res.messages || []
            if (before) {
                messages.value = [...msgs, ...messages.value]
            } else {
                messages.value = msgs
            }
            return msgs
        } catch (e) {
            console.error('loadMessages failed', e)
            showNotification('載入訊息失敗', 'error')
            return []
        }
    }

    function handleNewMessage(msg) {
        if (!currentChannel.value || msg.channel_id !== currentChannel.value.id) {
            // Track unread by channel→guild map
            const gid = channelGuildMap.get(msg.channel_id)
            if (gid && (!currentGuild.value || currentGuild.value.id !== gid)) {
                unreadGuildIds.add(gid)
            }
            if (!currentChannel.value || msg.channel_id !== currentChannel.value.id) return
        }
        if (msg.nonce) {
            const idx = messages.value.findIndex(m => m.nonce === msg.nonce && m._pending)
            if (idx !== -1) {
                // splice 觸發完整的响應式更新
                messages.value.splice(idx, 1, msg)
                return
            }
            // 已存在相同 nonce 的真實訊息，不重複新增
            if (messages.value.some(m => m.nonce === msg.nonce && !m._pending)) return
        }
        messages.value.push(msg)
    }

    function handleTranslationReady(data) {
        if (!data?.message_id) return
        translationCache.set(data.message_id, {
            original_lang: data.original_lang,
            translations: data.translations,
        })
        translationLoadingSet.delete(data.message_id)
    }

    function handleMessageUpdate(msg) {
        const idx = messages.value.findIndex(m => m.id === msg.id)
        if (idx !== -1) messages.value[idx] = msg
    }

    function handleMessageDelete(data) {
        const idx = messages.value.findIndex(m => m.id === data.message_id)
        if (idx !== -1) messages.value.splice(idx, 1)
    }

    // ── Typing ───────────────────────────────────────────────────
    function handleTyping(data) {
        const { channel_id, user_id, username } = data
        if (user.value && user_id === user.value.id) return
        if (!typingUsers.value[channel_id]) typingUsers.value[channel_id] = {}
        typingUsers.value[channel_id][user_id] = { username, ts: Date.now() }
        // auto-expire
        setTimeout(() => {
            if (typingUsers.value[channel_id]?.[user_id]) {
                delete typingUsers.value[channel_id][user_id]
            }
        }, 5500)
    }

    // ── Presence ─────────────────────────────────────────────────
    function handleUserStatus(data) {
        const m = members.value.find(m => m.user_id === data.user_id)
        if (m?.user) m.user.status = data.status
        // Track global online set for HomeView galaxy
        if (data.status === 'online') {
            onlineUserIds.add(data.user_id)
        } else {
            onlineUserIds.delete(data.user_id)
        }
    }

    // ── Guild events ─────────────────────────────────────────────
    function handleGuildUpdate(guild) {
        const idx = guilds.value.findIndex(g => g.id === guild.id)
        if (idx !== -1) guilds.value[idx] = guild
        if (currentGuild.value?.id === guild.id) currentGuild.value = guild
    }

    function handleGuildDelete(data) {
        guilds.value = guilds.value.filter(g => g.id !== data.guild_id)
        if (currentGuild.value?.id === data.guild_id) {
            currentGuild.value = null; currentChannel.value = null
            channels.value = []; members.value = []; messages.value = []
            showNotification('所在社群已被刪除', 'info')
        }
    }

    // ── Channel events ────────────────────────────────────────────
    function handleChannelCreate(ch) {
        if (currentGuild.value && ch.guild_id === currentGuild.value.id) channels.value.push(ch)
    }

    function handleChannelUpdate(ch) {
        const idx = channels.value.findIndex(c => c.id === ch.id)
        if (idx !== -1) channels.value[idx] = ch
        if (currentChannel.value?.id === ch.id) currentChannel.value = ch
    }

    function handleChannelDelete(data) {
        channels.value = channels.value.filter(c => c.id !== data.channel_id)
        if (currentChannel.value?.id === data.channel_id) {
            currentChannel.value = null; messages.value = []
        }
    }

    // ── Member events ─────────────────────────────────────────────
    function handleMemberAdd(data) {
        if (currentGuild.value?.id === data.guild_id && !members.value.some(m => m.user_id === data.user_id)) {
            members.value.push(data)
        }
    }

    function handleMemberRemove(data) {
        if (currentGuild.value?.id === data.guild_id) {
            members.value = members.value.filter(m => m.user_id !== data.user_id)
            if (user.value?.id === data.user_id) {
                guilds.value = guilds.value.filter(g => g.id !== data.guild_id)
                currentGuild.value = null; currentChannel.value = null
                channels.value = []; members.value = []; messages.value = []
                showNotification('您已被移出該社群', 'info')
            }
        }
    }

    function handleMemberUpdate(data) {
        if (currentGuild.value?.id === data.guild_id) {
            const idx = members.value.findIndex(m => m.user_id === data.user_id)
            if (idx !== -1) members.value[idx] = { ...members.value[idx], ...data }
        }
    }

    return {
        // state
        user, guilds, currentGuild, channels, currentChannel,
        members, messages, isLoading, notification, pendingFileIds,
        lightboxUrl, typingUsers, typingList,
        translationCache, translationLoadingSet, onlineUserIds, unreadGuildIds,
        // computed
        textChannels, voiceChannels, isAuthenticated, currentUserRole,
        // methods
        showNotification, setLoading, cacheUser, resolveMessageUser, getImageUrl,
        checkAuth, loadUserData, handleLogout, loadGuilds, selectGuild,
        selectChannel, loadMessages,
        handleNewMessage, handleMessageUpdate, handleMessageDelete,
        handleTyping, handleUserStatus,
        handleTranslationReady,
        handleGuildUpdate, handleGuildDelete,
        handleChannelCreate, handleChannelUpdate, handleChannelDelete,
        handleMemberAdd, handleMemberRemove, handleMemberUpdate,
    }
})
