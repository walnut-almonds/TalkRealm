// ─── 端點設定 ───────────────────────────────────────────────
const origin = window.location.origin
const wsOrigin = origin.replace(/^http/, 'ws')

export const API_BASE = origin
export const WS_URL = `${wsOrigin}/api/v1/ws`
const TENOR_BASE = 'https://tenor.googleapis.com/v2'

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
    MESSAGE_PERMALINK: (id) => `/api/v1/messages/${id}/permalink`,
    CHANNEL_POSTS: (channelId) => `/api/v1/channels/${channelId}/posts`,
    MESSAGE_COMMENTS: (id) => `/api/v1/messages/${id}/comments`,
    MESSAGE_LIKE: (id) => `/api/v1/messages/${id}/like`,
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
    // Learn
    LEARN_LEVELS: '/api/v1/learn/levels',
    LEARN_GUESS: (id) => `/api/v1/learn/levels/${id}/guess`,
    LEARN_STATS: '/api/v1/learn/stats',
    LEARN_DAILY: '/api/v1/learn/daily',
    LEARN_LEADERBOARD: '/api/v1/learn/daily/leaderboard',
    LEARN_CROSSWORD: '/api/v1/learn/levels/crossword',
    LEARN_HINT: (id) => `/api/v1/learn/levels/${id}/hint`,
    LEARN_REVEAL: (id) => `/api/v1/learn/levels/${id}/reveal`,
    LEARN_CAMPAIGN: '/api/v1/learn/campaign',
    LEARN_CAMPAIGN_START: (no) => `/api/v1/learn/campaign/${no}`,
    LEARN_CAMPAIGN_LB: '/api/v1/learn/campaign/leaderboard',
    LEARN_WEEKLY_LB: '/api/v1/learn/leaderboard/weekly',
    LEARN_SRS_OVERVIEW: '/api/v1/learn/srs/overview',
    LEARN_SRS_START: '/api/v1/learn/srs',
    LEARN_SRS_ANSWER: (id) => `/api/v1/learn/srs/${id}/answer`,
    // Feed (cross-community personal feed)
    FEED_TIMELINE: '/api/v1/feed/timeline',
    FEED_DISCOVER: '/api/v1/feed/discover',
    FEED_POSTS: '/api/v1/feed/posts',
    FEED_POST: (id) => `/api/v1/feed/posts/${id}`,
    FEED_POST_LIKE: (id) => `/api/v1/feed/posts/${id}/like`,
    FEED_POST_COMMENTS: (id) => `/api/v1/feed/posts/${id}/comments`,
    FEED_COMMENT: (id) => `/api/v1/feed/comments/${id}`,
    FEED_COMMENT_LIKE: (id) => `/api/v1/feed/comments/${id}/like`,
    FEED_SUGGESTIONS: '/api/v1/feed/suggestions',
    FEED_FOLLOW: (userId) => `/api/v1/feed/follows/${userId}`,
    FEED_USER_POSTS: (userId) => `/api/v1/feed/users/${userId}/posts`,
    FEED_FOLLOWING: (userId) => `/api/v1/feed/users/${userId}/following`,
    FEED_FOLLOWERS: (userId) => `/api/v1/feed/users/${userId}/followers`,
}

export const STORAGE_KEYS = {
    TOKEN: 'talkrealm_token',
    REFRESH_TOKEN: 'talkrealm_refresh_token',
    LAST_GUILD: 'talkrealm_last_guild',
    LAST_CHANNEL: 'talkrealm_last_channel',
    GIF_PROVIDER: 'talkrealm_gif_provider',
    GIF_API_KEY: 'talkrealm_gif_api_key',
    GIF_CLIENT_KEY: 'talkrealm_gif_client_key',
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
    getChannelMessages(channelId, limit = 50, before = null, around = null) {
        let url = `${EP.CHANNEL_MESSAGES(channelId)}?limit=${limit}`
        if (before) url += `&before=${before}`
        if (around) url += `&around=${around}`
        return this.get(url)
    }
    resolvePermalink(id) { return this.get(EP.MESSAGE_PERMALINK(id)) }
    sendMessage(channelId, content, type = 'text', nonce = null, fileIds = []) {
        const body = { content, type }
        if (nonce) body.nonce = nonce
        if (fileIds?.length) body.file_ids = fileIds
        return this.post(EP.CHANNEL_MESSAGES(channelId), body)
    }
    updateMessage(messageId, content) { return this.patch(EP.MESSAGE(messageId), { content }) }
    deleteMessage(messageId) { return this.del(EP.MESSAGE(messageId)) }

    // ── Feed / Wall ──
    getPosts(channelId, limit = 20, before = null, around = null) {
        const q = new URLSearchParams({ limit })
        if (before) q.set('before', before)
        if (around) q.set('around', around)
        return this.get(`${EP.CHANNEL_POSTS(channelId)}?${q}`)
    }
    createPost(channelId, content, fileIds = []) {
        return this.post(EP.CHANNEL_MESSAGES(channelId), { content, file_ids: fileIds })
    }
    getComments(postId, limit = 50, before = null) {
        const q = new URLSearchParams({ limit })
        if (before) q.set('before', before)
        return this.get(`${EP.MESSAGE_COMMENTS(postId)}?${q}`)
    }
    createComment(channelId, postId, content) {
        return this.post(EP.CHANNEL_MESSAGES(channelId), { content, parent_id: postId })
    }
    likePost(id) { return this.put(EP.MESSAGE_LIKE(id), {}) }
    unlikePost(id) { return this.del(EP.MESSAGE_LIKE(id)) }

    // ── Feed (cross-community personal feed) ──
    getTimeline(limit = 20, before = null) {
        const q = new URLSearchParams({ limit })
        if (before) q.set('before', before)
        return this.get(`${EP.FEED_TIMELINE}?${q}`)
    }
    getDiscover(offset = 0, limit = 20) {
        const q = new URLSearchParams({ offset, limit })
        return this.get(`${EP.FEED_DISCOVER}?${q}`)
    }
    createFeedPost(content, fileIds = []) { return this.post(EP.FEED_POSTS, { content, file_ids: fileIds }) }
    updateFeedPost(id, content) { return this.put(EP.FEED_POST(id), { content }) }
    deleteFeedPost(id) { return this.del(EP.FEED_POST(id)) }
    likeFeedPost(id) { return this.put(EP.FEED_POST_LIKE(id), {}) }
    unlikeFeedPost(id) { return this.del(EP.FEED_POST_LIKE(id)) }
    likeFeedComment(id) { return this.put(EP.FEED_COMMENT_LIKE(id), {}) }
    unlikeFeedComment(id) { return this.del(EP.FEED_COMMENT_LIKE(id)) }
    getFeedComments(id, limit = 50, before = null) {
        const q = new URLSearchParams({ limit })
        if (before) q.set('before', before)
        return this.get(`${EP.FEED_POST_COMMENTS(id)}?${q}`)
    }
    addFeedComment(id, content) { return this.post(EP.FEED_POST_COMMENTS(id), { content }) }
    getUserPosts(userId, limit = 20, before = null) {
        const q = new URLSearchParams({ limit })
        if (before) q.set('before', before)
        return this.get(`${EP.FEED_USER_POSTS(userId)}?${q}`)
    }
    getFollowSuggestions() { return this.get(EP.FEED_SUGGESTIONS) }
    follow(userId) { return this.post(EP.FEED_FOLLOW(userId), {}) }
    unfollow(userId) { return this.del(EP.FEED_FOLLOW(userId)) }
    getFollowing(userId) { return this.get(EP.FEED_FOLLOWING(userId)) }
    getFollowers(userId) { return this.get(EP.FEED_FOLLOWERS(userId)) }

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

    // ── GIF Settings ──
    getGIFConfig() {
        const provider = localStorage.getItem(STORAGE_KEYS.GIF_PROVIDER) || 'auto'
        const apiKey = localStorage.getItem(STORAGE_KEYS.GIF_API_KEY) || ''
        const clientKey = localStorage.getItem(STORAGE_KEYS.GIF_CLIENT_KEY) || 'talkrealm-web'
        return { provider, apiKey, clientKey }
    }

    setGIFConfig({ provider = 'auto', apiKey = '', clientKey = 'talkrealm-web' } = {}) {
        const normalizedProvider = ['auto', 'tenor-v2', 'tenor-v1'].includes(provider) ? provider : 'auto'
        const normalizedAPIKey = (apiKey || '').trim()
        const normalizedClientKey = (clientKey || 'talkrealm-web').trim() || 'talkrealm-web'

        localStorage.setItem(STORAGE_KEYS.GIF_PROVIDER, normalizedProvider)
        if (normalizedAPIKey) localStorage.setItem(STORAGE_KEYS.GIF_API_KEY, normalizedAPIKey)
        else localStorage.removeItem(STORAGE_KEYS.GIF_API_KEY)
        localStorage.setItem(STORAGE_KEYS.GIF_CLIENT_KEY, normalizedClientKey)
    }

    // ── GIF ──
    // Tenor's free/keyless v1 API (g.tenor.com/v1) was permanently discontinued
    // by Google, so v2 + a real registered API key is the only working path now.
    async searchGIFs(query = '', limit = 18, cursor = '') {
        const cappedLimit = Math.min(Math.max(Number(limit) || 18, 1), 30)
        const gifCfg = this.getGIFConfig()

        const v2Key = gifCfg.apiKey || import.meta.env.VITE_TENOR_API_KEY || ''
        const clientKey = gifCfg.clientKey || import.meta.env.VITE_TENOR_CLIENT_KEY || 'talkrealm-web'

        if (!v2Key) {
            throw new Error('請先在使用者設定中填入 Tenor API Key')
        }

        const endpoint = query ? 'search' : 'featured'
        const params = new URLSearchParams({
            key: v2Key,
            client_key: clientKey,
            media_filter: 'gif,tinygif',
            contentfilter: 'medium',
            locale: 'zh_TW',
            limit: String(cappedLimit),
        })
        if (query) params.set('q', query)
        if (cursor) params.set('pos', cursor)

        const res = await fetch(`${TENOR_BASE}/${endpoint}?${params.toString()}`)
        const data = await res.json().catch(() => ({}))
        if (!res.ok) {
            throw new Error(data?.error?.message || data?.error || 'GIF 服務暫時不可用')
        }
        return {
            items: (data.results || []).map((item) => {
                const gif = item?.media_formats?.gif?.url || ''
                const tiny = item?.media_formats?.tinygif?.url || gif
                return {
                    id: item.id,
                    title: item.content_description || 'GIF',
                    url: gif,
                    previewUrl: tiny,
                    source: 'tenor-v2',
                }
            }).filter(item => item.url),
            next: data.next || '',
        }
    }

    // ── Unread ──
    getAllUnread() { return this.get('/api/v1/users/me/unread') }
    getChannelUnread(channelId) { return this.get(`/api/v1/channels/${channelId}/unread`) }
    ackChannel(channelId, lastMessageId) { return this.post(`/api/v1/channels/${channelId}/ack`, { last_message_id: lastMessageId }) }
}

export const api = new ApiClient()
