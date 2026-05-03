// WebSocket 連接管理
class WebSocketManager {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.heartbeatInterval = null;
        this.isConnected = false;
        this.identified = false;
        this.subscribedChannels = new Set();
        this.messageHandlers = [];
        this._token = null;
    }

    // 連接 WebSocket
    connect(token) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            console.log('WebSocket already connected');
            return;
        }

        this._token = token;
        // 不把 token 放 query string，由 identify op 傳遞
        const wsUrl = `${API_CONFIG.WS_URL}${API_CONFIG.ENDPOINTS.WS}`;

        try {
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected, waiting for hello...');
                // 不在 onopen 做任何事；等待 server 的 hello，再送 identify
            };

            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.isConnected = false;
                this.identified = false;
                this.stopHeartbeat();
                this.attemptReconnect(token);
            };
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
        }
    }

    // 斷開連接
    disconnect() {
        this.reconnectAttempts = this.maxReconnectAttempts; // 防止自動重連
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.stopHeartbeat();
        this.isConnected = false;
    }

    // 嘗試重新連接
    attemptReconnect(token) {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.log('Max reconnect attempts reached');
            showNotification('WebSocket 連接失敗，請重新整理頁面', 'error');
            return;
        }

        this.reconnectAttempts++;
        const delay = this.reconnectDelay * this.reconnectAttempts;

        console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);

        setTimeout(() => {
            this.connect(token);
        }, delay);
    }

    // 發送訊息
    send(message) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket is not connected');
            return false;
        }

        try {
            this.ws.send(JSON.stringify(message));
            return true;
        } catch (error) {
            console.error('Failed to send WebSocket message:', error);
            return false;
        }
    }

    // 送出 identify op（收到 hello 後呼叫）
    sendIdentify(channels = []) {
        return this.send({
            op: 'identify',
            d: {
                token: this._token,
                channels: channels
            }
        });
    }

    // 訂閱頻道
    subscribeToChannel(channelId) {
        this.subscribedChannels.add(channelId);
        return this.send({
            op: 'subscribe',
            d: { channel_id: channelId }
        });
    }

    // 取消訂閱頻道
    unsubscribeFromChannel(channelId) {
        this.subscribedChannels.delete(channelId);
        return this.send({
            op: 'unsubscribe',
            d: { channel_id: channelId }
        });
    }

    // 透過 WebSocket 發送訊息
    sendMessage(channelId, content, type = 'text', nonce = '') {
        return this.send({
            op: 'send_message',
            d: { channel_id: channelId, content, type, nonce }
        });
    }

    // 發送正在輸入狀態
    sendTyping(channelId) {
        return this.send({
            op: 'typing_start',
            d: { channel_id: channelId }
        });
    }

    // 心跳機制
    startHeartbeat(interval = 30000) {
        this.stopHeartbeat();
        this.heartbeatInterval = setInterval(() => {
            if (this.isConnected) {
                this.send({ op: 'heartbeat' });
            }
        }, interval);
    }

    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }

    // 處理接收到的訊息
    handleMessage(message) {
        console.log('WebSocket message received:', message);

        // 後端使用 op / d 欄位
        switch (message.op) {
            case 'hello': {
                // server 發送心跳間隔；回應 identify
                const interval = message.d && message.d.heartbeat_interval || 30000;
                this.startHeartbeat(interval);
                // identify 時一次帶入目前已知的訂閱頻道
                this.sendIdentify([...this.subscribedChannels]);
                break;
            }

            case 'ready':
                // identify 成功
                this.isConnected = true;
                this.identified = true;
                this.reconnectAttempts = 0;
                console.log('WebSocket identified, user:', message.d);
                this.notifyHandlers('ready', message.d);
                break;

            case 'heartbeat_ack':
                // 心跳回應
                break;

            case 'message_create':
                // 新訊息
                this.notifyHandlers('message', message.d);
                break;

            case 'message_update':
                // 訊息更新
                this.notifyHandlers('message_update', message.d);
                break;

            case 'message_delete':
                // 訊息刪除
                this.notifyHandlers('message_delete', message.d);
                break;

            case 'typing_start':
                // 使用者正在輸入
                this.notifyHandlers('typing', message.d);
                break;

            case 'presence_update':
                // 使用者狀態更新
                this.notifyHandlers('user_status', message.d);
                break;

            case 'channel_create':
                // 頻道建立
                this.notifyHandlers('channel_create', message.d);
                break;

            case 'channel_update':
                // 頻道更新
                this.notifyHandlers('channel_update', message.d);
                break;

            case 'channel_delete':
                // 頻道刪除
                this.notifyHandlers('channel_delete', message.d);
                break;

            case 'guild_update':
                // 社群資訊更新
                this.notifyHandlers('guild_update', message.d);
                break;

            case 'guild_delete':
                // 社群刪除
                this.notifyHandlers('guild_delete', message.d);
                break;

            case 'guild_member_add':
                // 成員加入社群
                this.notifyHandlers('guild_member_add', message.d);
                break;

            case 'guild_member_remove':
                // 成員離開／被踢出社群
                this.notifyHandlers('guild_member_remove', message.d);
                break;

            case 'guild_member_update':
                // 成員資訊更新（角色等）
                this.notifyHandlers('guild_member_update', message.d);
                break;

            case 'error':
                // 錯誤訊息
                console.error('WebSocket error:', message.d);
                showNotification((message.d && message.d.message) || '發生錯誤', 'error');
                break;

            default:
                console.log('Unknown message type:', message.type);
        }
    }

    // 註冊訊息處理器
    onMessage(handler) {
        this.messageHandlers.push(handler);
    }

    // 移除訊息處理器
    offMessage(handler) {
        const index = this.messageHandlers.indexOf(handler);
        if (index > -1) {
            this.messageHandlers.splice(index, 1);
        }
    }

    // 通知所有處理器
    notifyHandlers(type, data) {
        this.messageHandlers.forEach(handler => {
            try {
                handler(type, data);
            } catch (error) {
                console.error('Error in message handler:', error);
            }
        });
    }
}

// 建立 WebSocket 管理器實例
const wsManager = new WebSocketManager();
