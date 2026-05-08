// API 配置
const _origin = window.location.origin; // e.g. http://140.99.243.135:8080
const _wsOrigin = _origin.replace(/^http/, 'ws');

const API_CONFIG = {
    BASE_URL: _origin,
    WS_URL: _wsOrigin,
    ENDPOINTS: {
        // 認證
        REGISTER: '/api/v1/auth/register',
        LOGIN: '/api/v1/auth/login',
        REFRESH: '/api/v1/auth/refresh',
        LOGOUT: '/api/v1/auth/logout',

        // 使用者
        ME: '/api/v1/users/me',
        UPDATE_ME: '/api/v1/users/me',

        // 社群
        GUILDS: '/api/v1/guilds',
        GUILD: (id) => `/api/v1/guilds/${id}`,
        MY_GUILDS: '/api/v1/guilds/me',

        // 社群成員
        GUILD_MEMBERS: (guildId) => `/api/v1/guilds/${guildId}/members`,
        JOIN_GUILD: (guildId) => `/api/v1/guilds/${guildId}/join`,
        LEAVE_GUILD: (guildId) => `/api/v1/guilds/${guildId}/leave`,

        // 頻道
        GUILD_CHANNELS: (guildId) => `/api/v1/guilds/${guildId}/channels`,
        CHANNEL: (channelId) => `/api/v1/channels/${channelId}`,

        // 訊息
        CHANNEL_MESSAGES: (channelId) => `/api/v1/channels/${channelId}/messages`,
        MESSAGE: (messageId) => `/api/v1/messages/${messageId}`,

        // 邀請碼
        CREATE_INVITE: (guildId) => `/api/v1/guilds/${guildId}/invites`,
        GET_INVITE: (code) => `/api/v1/invites/${code}`,
        JOIN_BY_INVITE: '/api/v1/guilds/join-by-invite',

        // 成員管理
        KICK_MEMBER: (guildId, userId) => `/api/v1/guilds/${guildId}/members/${userId}`,
        UPDATE_MEMBER_ROLE: (guildId, userId) => `/api/v1/guilds/${guildId}/members/${userId}/role`,

        // 使用者公開資料
        PUBLIC_USER: (id) => `/api/v1/users/${id}`,

        // 檔案
        FILE_PRESIGN: '/api/v1/files/presign',
        FILE_CONFIRM: (id) => `/api/v1/files/${id}/confirm`,
        FILE_META: (id) => `/api/v1/files/${id}`,
        FILE_URL: (id) => `/api/v1/files/${id}/url`,
        FILE_DELETE: (id) => `/api/v1/files/${id}`,

        // 語音
        VOICE_TOKEN: (channelId) => `/api/v1/channels/${channelId}/voice/token`,

        // WebSocket
        WS: '/api/v1/ws'
    }
};

// 本地儲存鍵
const STORAGE_KEYS = {
    TOKEN: 'talkrealm_token',
    REFRESH_TOKEN: 'talkrealm_refresh_token',
    USER: 'talkrealm_user',
    LAST_GUILD: 'talkrealm_last_guild',
    LAST_CHANNEL: 'talkrealm_last_channel'
};
