// ─── 端點設定 ───────────────────────────────────────────────
const origin = window.location.origin
const wsOrigin = origin.replace(/^http/, 'ws')

export const API_BASE = origin
export const WS_URL = `${wsOrigin}/api/v1/ws`

export const EP = {
    REGISTER: '/api/v1/auth/register',
    LOGIN: '/api/v1/auth/login',
    REFRESH: '/api/v1/auth/refresh',
    LOGOUT: '/api/v1/auth/logout',
    ME: '/api/v1/users/me',
    UPDATE_ME: '/api/v1/users/me',
    PUBLIC_USER: (id) => `/api/v1/users/${id}`,
    GUILDS: '/api/v1/guilds',
    GUILD: (id) => `/api/v1/guilds/${id}`,
    MY_GUILDS: '/api/v1/guilds/me',
    GUILD_MEMBERS: (guildId) => `/api/v1/guilds/${guildId}/members`,
    JOIN_GUILD: (guildId) => `/api/v1/guilds/${guildId}/join`,
    LEAVE_GUILD: (guildId) => `/api/v1/guilds/${guildId}/leave`,
    GUILD_CHANNELS: (guildId) => `/api/v1/guilds/${guildId}/channels`,
    CHANNEL: (channelId) => `/api/v1/channels/${channelId}`,
    CHANNEL_MESSAGES: (channelId) => `/api/v1/channels/${channelId}/messages`,
    MESSAGE: (messageId) => `/api/v1/messages/${messageId}`,
    CREATE_INVITE: (guildId) => `/api/v1/guilds/${guildId}/invites`,
    GET_INVITE: (code) => `/api/v1/invites/${code}`,
    JOIN_BY_INVITE: '/api/v1/guilds/join-by-invite',
    KICK_MEMBER: (guildId, userId) => `/api/v1/guilds/${guildId}/members/${userId}`,
    UPDATE_MEMBER_ROLE: (guildId, userId) => `/api/v1/guilds/${guildId}/members/${userId}/role`,
    FILE_PRESIGN: '/api/v1/files/presign',
    FILE_CONFIRM: (id) => `/api/v1/files/${id}/confirm`,
    FILE_URL: (id) => `/api/v1/files/${id}/url`,
    FILE_DELETE: (id) => `/api/v1/files/${id}`,
    VOICE_TOKEN: (channelId) => `/api/v1/channels/${channelId}/voice/token`,
    VOICE_PARTICIPANTS: (channelId) => `/api/v1/channels/${channelId}/voice/participants`,
    MESSAGE_TRANSLATION: (id) => `/api/v1/messages/${id}/translation`,
    MESSAGE_TRANSLATION_ENSURE: (id) => `/api/v1/messages/${id}/translation/ensure`,
    MESSAGE_GUESS: (id) => `/api/v1/messages/${id}/guess`,
    MESSAGE_GAME: (id) => `/api/v1/messages/${id}/game`,
    // DM
    DM_CHANNELS: '/api/v1/dm/channels',
    DM_CHANNEL: (id) => `/api/v1/dm/channels/${id}`,
    DM_MESSAGES: (id) => `/api/v1/dm/channels/${id}/messages`,
    DM_MESSAGE: (id) => `/api/v1/dm/messages/${id}`,
    DM_MESSAGE_TRANSLATION: (id) => `/api/v1/dm/messages/${id}/translation`,
    DM_MESSAGE_TRANSLATION_ENSURE: (id) => `/api/v1/dm/messages/${id}/translation/ensure`,
    // Interaction stats
    INTERACTION_STATS: (days) => `/api/v1/users/me/interaction-stats?days=${days || 30}`,
    // Friends
    SEARCH_USERS: (q) => `/api/v1/users/search?q=${encodeURIComponent(q)}`,
    FRIENDS: '/api/v1/friends',
    FRIENDS_REQUESTS_INCOMING: '/api/v1/friends/requests/incoming',
    FRIENDS_REQUESTS_OUTGOING: '/api/v1/friends/requests/outgoing',
    FRIEND_ACCEPT: (userId) => `/api/v1/friends/${userId}/accept`,
    FRIEND_REMOVE: (userId) => `/api/v1/friends/${userId}`,
}

export const STORAGE_KEYS = {
    TOKEN: 'talkrealm_token',
    REFRESH_TOKEN: 'talkrealm_refresh_token',
    LAST_GUILD: 'talkrealm_last_guild',
    LAST_CHANNEL: 'talkrealm_last_channel',
}

/** 取得/設定某個 guild 最後停留的頻道 ID */
export const guildLastChannel = {
    get: (guildId) => {
        const v = localStorage.getItem(`talkrealm_last_channel_${guildId}`)
        return v ? parseInt(v) : null
    },
    set: (guildId, channelId) => {
        localStorage.setItem(`talkrealm_last_channel_${guildId}`, channelId)
    },
}

// ─── API 用戶端 ──────────────────────────────────────────────
class ApiClient {
    constructor() {
        this.token = localStorage.getItem(STORAGE_KEYS.TOKEN)
    }

    setToken(t) {
        this.token = t
        if (t) localStorage.setItem(STORAGE_KEYS.TOKEN, t)
        else localStorage.removeItem(STORAGE_KEYS.TOKEN)
    }

    setRefreshToken(t) {
        if (t) localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, t)
        else localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN)
    }

    getRefreshToken() {
        return localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN)
    }

    headers(auth = true) {
        const h = { 'Content-Type': 'application/json' }
        if (auth && this.token) h['Authorization'] = `Bearer ${this.token}`
        return h
    }

    async request(url, options = {}, isRetry = false) {
        const res = await fetch(`${API_BASE}${url}`, {
            ...options,
            headers: this.headers(options.auth !== false),
        })
        const data = await res.json().catch(() => ({}))

        if (res.status === 401 && !isRetry && options.auth !== false) {
            const rt = this.getRefreshToken()
            if (rt) {
                try {
                    const refreshed = await this.post(EP.REFRESH, { refresh_token: rt }, false)
                    this.setToken(refreshed.access_token)
                    this.setRefreshToken(refreshed.refresh_token)
                    return this.request(url, options, true)
                } catch {
                    this.setToken(null)
                    this.setRefreshToken(null)
                }
            }
        }

        if (!res.ok) throw new Error(data.error || data.message || '請求失敗')
        return data
    }

    get(url, auth = true) { return this.request(url, { method: 'GET', auth }) }
    post(url, body, auth = true) { return this.request(url, { method: 'POST', body: JSON.stringify(body), auth }) }
    patch(url, body, auth = true) { return this.request(url, { method: 'PATCH', body: JSON.stringify(body), auth }) }
    del(url, auth = true) { return this.request(url, { method: 'DELETE', auth }) }
    put(url, body, auth = true) { return this.request(url, { method: 'PUT', body: JSON.stringify(body), auth }) }

    // ── Auth ──
    register(username, email, password, nickname) {
        return this.post(EP.REGISTER, { username, email, password, nickname: nickname || username }, false)
    }
    async login(email, password) {
        const data = await this.post(EP.LOGIN, { email, password }, false)
        if (data.access_token) this.setToken(data.access_token)
        if (data.refresh_token) this.setRefreshToken(data.refresh_token)
        return data
    }
    async logout() {
        const rt = this.getRefreshToken()
        if (rt) await this.post(EP.LOGOUT, { refresh_token: rt }).catch(() => { })
        this.setToken(null)
        this.setRefreshToken(null)
    }

    // ── User ──
    getCurrentUser() { return this.get(EP.ME) }
    updateCurrentUser(updates) { return this.patch(EP.UPDATE_ME, updates) }
    getPublicUser(userId) { return this.get(EP.PUBLIC_USER(userId)) }

    // ── Guild ──
    getMyGuilds() { return this.get(EP.MY_GUILDS) }
    getGuild(id) { return this.get(EP.GUILD(id)) }
    createGuild(nameOrObj, description) {
        const body = typeof nameOrObj === 'object' ? nameOrObj : { name: nameOrObj, description }
        return this.post(EP.GUILDS, body)
    }
    updateGuild(id, updates) { return this.patch(EP.GUILD(id), updates) }
    deleteGuild(id) { return this.del(EP.GUILD(id)) }

    // ── Members ──
    getGuildMembers(guildId) { return this.get(EP.GUILD_MEMBERS(guildId)) }
    kickMember(guildId, userId) { return this.del(EP.KICK_MEMBER(guildId, userId)) }
    updateMemberRole(guildId, userId, role) { return this.put(EP.UPDATE_MEMBER_ROLE(guildId, userId), { role }) }

    // ── Channel ──
    getGuildChannels(guildId) { return this.get(EP.GUILD_CHANNELS(guildId)) }
    getChannel(channelId) { return this.get(EP.CHANNEL(channelId)) }
    createChannel(guildId, nameOrObj, type, topic) {
        const body = typeof nameOrObj === 'object' ? nameOrObj : { name: nameOrObj, type, topic }
        return this.post(EP.GUILD_CHANNELS(guildId), body)
    }
    updateChannel(channelId, updates) { return this.patch(EP.CHANNEL(channelId), updates) }
    deleteChannel(channelId) { return this.del(EP.CHANNEL(channelId)) }

    // ── Message ──
    getChannelMessages(channelId, limit = 50, before = null) {
        let url = `${EP.CHANNEL_MESSAGES(channelId)}?limit=${limit}`
        if (before) url += `&before=${before}`
        return this.get(url)
    }
    sendMessage(channelId, content, type = 'text', nonce = null, fileIds = []) {
        const body = { content, type }
        if (nonce) body.nonce = nonce
        if (fileIds?.length) body.file_ids = fileIds
        return this.post(EP.CHANNEL_MESSAGES(channelId), body)
    }
    updateMessage(messageId, content) { return this.patch(EP.MESSAGE(messageId), { content }) }
    deleteMessage(messageId) { return this.del(EP.MESSAGE(messageId)) }

    // ── Invite ──
    createInvite(guildId) { return this.post(EP.CREATE_INVITE(guildId), {}) }
    getInvite(code) { return this.get(EP.GET_INVITE(code), false) }
    joinByInvite(code) { return this.post(EP.JOIN_BY_INVITE, { code }) }

    // ── File ──
    presignUpload(filename, contentType, size) {
        return this.post(EP.FILE_PRESIGN, { filename, content_type: contentType, size })
    }
    uploadToMinio(uploadUrl, file, onProgress) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest()
            xhr.open('PUT', uploadUrl)
            xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream')
            if (onProgress) {
                xhr.upload.onprogress = (e) => {
                    if (e.lengthComputable) onProgress(Math.round((e.loaded / e.total) * 100))
                }
            }
            xhr.onload = () => (xhr.status >= 200 && xhr.status < 300) ? resolve() : reject(new Error(`上傳失敗 (${xhr.status})`))
            xhr.onerror = () => reject(new Error('網路錯誤，上傳失敗'))
            xhr.send(file)
        })
    }
    confirmUpload(fileId) { return this.post(EP.FILE_CONFIRM(fileId), {}) }
    getFileDownloadUrl(fileId) { return this.get(EP.FILE_URL(fileId)) }
    deleteFile(fileId) { return this.del(EP.FILE_DELETE(fileId)) }

    // ── Voice ──
    getVoiceToken(channelId) { return this.get(EP.VOICE_TOKEN(channelId)) }
    getVoiceParticipants(channelId) { return this.get(EP.VOICE_PARTICIPANTS(channelId)) }

    // ── DM ──
    openDMChannel(targetUserId) { return this.post(EP.DM_CHANNELS, { target_user_id: targetUserId }) }
    listDMChannels() { return this.get(EP.DM_CHANNELS) }
    getDMMessages(dmChannelId, limit = 50, before = null) {
        let url = `${EP.DM_MESSAGES(dmChannelId)}?limit=${limit}`
        if (before) url += `&before=${before}`
        return this.get(url)
    }
    sendDMMessage(dmChannelId, content, nonce = null, fileIds = []) {
        const body = { content }
        if (nonce) body.nonce = nonce
        if (fileIds?.length) body.file_ids = fileIds
        return this.post(EP.DM_MESSAGES(dmChannelId), body)
    }
    updateDMMessage(id, content) { return this.patch(EP.DM_MESSAGE(id), { content }) }
    deleteDMMessage(id) { return this.del(EP.DM_MESSAGE(id)) }
    getDMTranslation(id) { return this.get(EP.DM_MESSAGE_TRANSLATION(id)) }
    ensureDMTranslation(id, lang) {
        const q = lang ? `?lang=${encodeURIComponent(lang)}` : ''
        return this.get(EP.DM_MESSAGE_TRANSLATION_ENSURE(id) + q)
    }

    // ── Profile (aliases) ──
    updateProfile(updates) { return this.patch(EP.UPDATE_ME, updates) }
    changePassword(body) { return this.post('/api/v1/users/me/password', body) }

    // ── Translation ──
    getTranslation(messageId) { return this.get(EP.MESSAGE_TRANSLATION(messageId)) }
    requestTranslation(messageId, lang) {
        const q = lang ? `?lang=${encodeURIComponent(lang)}` : ''
        return this.post(EP.MESSAGE_TRANSLATION(messageId) + q, {})
    }
    ensureTranslation(messageId, lang) {
        const q = lang ? `?lang=${encodeURIComponent(lang)}` : ''
        return this.get(EP.MESSAGE_TRANSLATION_ENSURE(messageId) + q)
    }
    submitGuess(messageId, guessContent, hiddenLang) {
        return this.post(EP.MESSAGE_GUESS(messageId), { guess_content: guessContent, hidden_lang: hiddenLang })
    }
    getGameStatus(messageId, hiddenLang) {
        return this.get(`${EP.MESSAGE_GAME(messageId)}?hidden_lang=${hiddenLang}`)
    }

    // ── Interaction stats ──
    getInteractionStats(days = 30) { return this.get(EP.INTERACTION_STATS(days)) }

    // ── Friends ──
    searchUsers(q) { return this.get(EP.SEARCH_USERS(q)) }
    listFriends() { return this.get(EP.FRIENDS) }
    listIncomingRequests() { return this.get(EP.FRIENDS_REQUESTS_INCOMING) }
    listOutgoingRequests() { return this.get(EP.FRIENDS_REQUESTS_OUTGOING) }
    sendFriendRequest(username) { return this.post(EP.FRIENDS, { username }) }
    acceptFriendRequest(userId) { return this.put(EP.FRIEND_ACCEPT(userId), {}) }
    removeFriend(userId) { return this.del(EP.FRIEND_REMOVE(userId)) }

    // ── OG Preview ──
    getOGPreview(url) { return this.get(`/api/v1/og?url=${encodeURIComponent(url)}`) }
}

export const api = new ApiClient()
