// API 配置
const API_CONFIG = {
    BASE_URL: 'http://localhost:8080',
    WS_URL: 'ws://localhost:8080',
    ENDPOINTS: {
        // 認證
        REGISTER: '/api/v1/auth/register',
        LOGIN: '/api/v1/auth/login',
        
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
        
        // WebSocket
        WS: '/ws'
    }
};

// 本地儲存鍵
const STORAGE_KEYS = {
    TOKEN: 'talkrealm_token',
    USER: 'talkrealm_user',
    LAST_GUILD: 'talkrealm_last_guild',
    LAST_CHANNEL: 'talkrealm_last_channel'
};
