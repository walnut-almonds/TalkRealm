// API 請求處理
class API {
    constructor() {
        this.baseURL = API_CONFIG.BASE_URL;
        this.token = localStorage.getItem(STORAGE_KEYS.TOKEN);
    }

    // 設定 token
    setToken(token) {
        this.token = token;
        if (token) {
            localStorage.setItem(STORAGE_KEYS.TOKEN, token);
        } else {
            localStorage.removeItem(STORAGE_KEYS.TOKEN);
        }
    }

    // 獲取 headers
    getHeaders(includeAuth = true) {
        const headers = {
            'Content-Type': 'application/json'
        };
        
        if (includeAuth && this.token) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }
        
        return headers;
    }

    // 通用請求方法
    async request(url, options = {}) {
        const config = {
            ...options,
            headers: this.getHeaders(options.auth !== false)
        };

        try {
            const response = await fetch(`${this.baseURL}${url}`, config);
            const data = await response.json().catch(() => ({}));

            if (!response.ok) {
                throw new Error(data.error || data.message || '請求失敗');
            }

            return data;
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }

    // GET 請求
    async get(url, auth = true) {
        return this.request(url, { method: 'GET', auth });
    }

    // POST 請求
    async post(url, data, auth = true) {
        return this.request(url, {
            method: 'POST',
            body: JSON.stringify(data),
            auth
        });
    }

    // PATCH 請求
    async patch(url, data, auth = true) {
        return this.request(url, {
            method: 'PATCH',
            body: JSON.stringify(data),
            auth
        });
    }

    // DELETE 請求
    async delete(url, auth = true) {
        return this.request(url, { method: 'DELETE', auth });
    }

    // 認證 API
    async register(username, email, password, nickname) {
        const data = await this.post(API_CONFIG.ENDPOINTS.REGISTER, {
            username,
            email,
            password,
            nickname: nickname || username
        }, false);
        return data;
    }

    async login(email, password) {
        const data = await this.post(API_CONFIG.ENDPOINTS.LOGIN, {
            email,
            password
        }, false);
        
        if (data.token) {
            this.setToken(data.token);
        }
        
        return data;
    }

    // 使用者 API
    async getCurrentUser() {
        return this.get(API_CONFIG.ENDPOINTS.ME);
    }

    async updateCurrentUser(updates) {
        return this.patch(API_CONFIG.ENDPOINTS.UPDATE_ME, updates);
    }

    // 社群 API
    async getMyGuilds() {
        return this.get(API_CONFIG.ENDPOINTS.MY_GUILDS);
    }

    async getGuild(guildId) {
        return this.get(API_CONFIG.ENDPOINTS.GUILD(guildId));
    }

    async createGuild(name, description) {
        return this.post(API_CONFIG.ENDPOINTS.GUILDS, {
            name,
            description
        });
    }

    async updateGuild(guildId, updates) {
        return this.patch(API_CONFIG.ENDPOINTS.GUILD(guildId), updates);
    }

    async deleteGuild(guildId) {
        return this.delete(API_CONFIG.ENDPOINTS.GUILD(guildId));
    }

    // 社群成員 API
    async getGuildMembers(guildId) {
        return this.get(API_CONFIG.ENDPOINTS.GUILD_MEMBERS(guildId));
    }

    async joinGuild(guildId) {
        return this.post(API_CONFIG.ENDPOINTS.JOIN_GUILD(guildId), {});
    }

    async leaveGuild(guildId) {
        return this.post(API_CONFIG.ENDPOINTS.LEAVE_GUILD(guildId), {});
    }

    // 頻道 API
    async getGuildChannels(guildId) {
        return this.get(API_CONFIG.ENDPOINTS.GUILD_CHANNELS(guildId));
    }

    async getChannel(channelId) {
        return this.get(API_CONFIG.ENDPOINTS.CHANNEL(channelId));
    }

    async createChannel(guildId, name, type, description) {
        return this.post(API_CONFIG.ENDPOINTS.GUILD_CHANNELS(guildId), {
            name,
            type,
            description
        });
    }

    async updateChannel(channelId, updates) {
        return this.patch(API_CONFIG.ENDPOINTS.CHANNEL(channelId), updates);
    }

    async deleteChannel(channelId) {
        return this.delete(API_CONFIG.ENDPOINTS.CHANNEL(channelId));
    }

    // 訊息 API
    async getChannelMessages(channelId, limit = 50, before = null) {
        let url = `${API_CONFIG.ENDPOINTS.CHANNEL_MESSAGES(channelId)}?limit=${limit}`;
        if (before) {
            url += `&before=${before}`;
        }
        return this.get(url);
    }

    async sendMessage(channelId, content, messageType = 'text') {
        return this.post(API_CONFIG.ENDPOINTS.CHANNEL_MESSAGES(channelId), {
            content,
            message_type: messageType
        });
    }

    async updateMessage(messageId, content) {
        return this.patch(API_CONFIG.ENDPOINTS.MESSAGE(messageId), {
            content
        });
    }

    async deleteMessage(messageId) {
        return this.delete(API_CONFIG.ENDPOINTS.MESSAGE(messageId));
    }
}

// 建立 API 實例
const api = new API();
