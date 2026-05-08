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

    // 設定 refresh token
    setRefreshToken(token) {
        if (token) {
            localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, token);
        } else {
            localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
        }
    }

    getRefreshToken() {
        return localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
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
    async request(url, options = {}, _isRetry = false) {
        const config = {
            ...options,
            headers: this.getHeaders(options.auth !== false)
        };

        try {
            const response = await fetch(`${this.baseURL}${url}`, config);
            const data = await response.json().catch(() => ({}));

            // 嘗試自動換發 access token（一次）
            if (response.status === 401 && !_isRetry && options.auth !== false) {
                const refreshToken = this.getRefreshToken();
                if (refreshToken) {
                    try {
                        const refreshed = await this.refreshToken(refreshToken);
                        this.setToken(refreshed.access_token);
                        this.setRefreshToken(refreshed.refresh_token);
                        return this.request(url, options, true);
                    } catch (_) {
                        // refresh 失敗，清除 token 並拋出錯誤
                        this.setToken(null);
                        this.setRefreshToken(null);
                    }
                }
            }

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

        if (data.access_token) {
            this.setToken(data.access_token);
        }
        if (data.refresh_token) {
            this.setRefreshToken(data.refresh_token);
        }

        return data;
    }

    async refreshToken(refreshToken) {
        return this.post(API_CONFIG.ENDPOINTS.REFRESH, { refresh_token: refreshToken }, false);
    }

    async logout(refreshToken) {
        const rt = refreshToken || this.getRefreshToken();
        if (rt) {
            await this.post(API_CONFIG.ENDPOINTS.LOGOUT, { refresh_token: rt }).catch(() => { });
        }
        this.setToken(null);
        this.setRefreshToken(null);
    }

    // 使用者 API
    async getCurrentUser() {
        return this.get(API_CONFIG.ENDPOINTS.ME);
    }

    async updateCurrentUser(updates) {
        return this.patch(API_CONFIG.ENDPOINTS.UPDATE_ME, updates);
    }

    async getPublicUser(userId) {
        return this.get(API_CONFIG.ENDPOINTS.PUBLIC_USER(userId));
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

    async createChannel(guildId, name, type, topic) {
        return this.post(API_CONFIG.ENDPOINTS.GUILD_CHANNELS(guildId), {
            name,
            type,
            topic
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

    async sendMessage(channelId, content, messageType = 'text', nonce = null, fileIds = []) {
        const body = { content, type: messageType };
        if (nonce) body.nonce = nonce;
        if (fileIds && fileIds.length > 0) body.file_ids = fileIds;
        return this.post(API_CONFIG.ENDPOINTS.CHANNEL_MESSAGES(channelId), body);
    }

    async updateMessage(messageId, content) {
        return this.patch(API_CONFIG.ENDPOINTS.MESSAGE(messageId), {
            content
        });
    }

    async deleteMessage(messageId) {
        return this.delete(API_CONFIG.ENDPOINTS.MESSAGE(messageId));
    }

    // 成員管理 API
    async kickMember(guildId, userId) {
        return this.delete(API_CONFIG.ENDPOINTS.KICK_MEMBER(guildId, userId));
    }

    async updateMemberRole(guildId, userId, role) {
        return this.request(API_CONFIG.ENDPOINTS.UPDATE_MEMBER_ROLE(guildId, userId), {
            method: 'PUT',
            body: JSON.stringify({ role })
        });
    }

    // 邀請碼 API
    async createInvite(guildId) {
        return this.post(API_CONFIG.ENDPOINTS.CREATE_INVITE(guildId), {});
    }

    async getInvite(code) {
        return this.get(API_CONFIG.ENDPOINTS.GET_INVITE(code), false);
    }

    async joinByInvite(code) {
        return this.post(API_CONFIG.ENDPOINTS.JOIN_BY_INVITE, { code });
    }

    // 檔案 API
    // 1. 取得 pre-signed upload URL（建立 pending 記錄）
    async presignUpload(filename, contentType, size) {
        return this.post(API_CONFIG.ENDPOINTS.FILE_PRESIGN, {
            filename,
            content_type: contentType,
            size
        });
    }

    // 2. 直接 PUT 檔案至 Minio（不經過 API Server）
    async uploadToMinio(uploadUrl, file, onProgress) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            xhr.open('PUT', uploadUrl);
            xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream');

            if (onProgress) {
                xhr.upload.onprogress = (e) => {
                    if (e.lengthComputable) {
                        onProgress(Math.round((e.loaded / e.total) * 100));
                    }
                };
            }

            xhr.onload = () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    resolve();
                } else {
                    reject(new Error(`上傳失敗 (${xhr.status})`));
                }
            };
            xhr.onerror = () => reject(new Error('網路錯誤，上傳失敗'));
            xhr.send(file);
        });
    }

    // 3. 確認上傳完成
    async confirmUpload(fileId) {
        return this.post(API_CONFIG.ENDPOINTS.FILE_CONFIRM(fileId), {});
    }

    // 取得下載 URL
    async getFileDownloadUrl(fileId) {
        return this.get(API_CONFIG.ENDPOINTS.FILE_URL(fileId));
    }

    // 刪除檔案
    async deleteFile(fileId) {
        return this.request(API_CONFIG.ENDPOINTS.FILE_DELETE(fileId), {
            method: 'DELETE'
        });
    }
    // 語音 API
    async getVoiceToken(channelId) {
        return this.get(API_CONFIG.ENDPOINTS.VOICE_TOKEN(channelId));
    }
}

// 建立 API 實例
const api = new API();
