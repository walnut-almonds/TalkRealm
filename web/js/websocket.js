// WebSocket 連接管理
class WebSocketManager {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.heartbeatInterval = null;
        this.isConnected = false;
        this.subscribedChannels = new Set();
        this.messageHandlers = [];
    }

    // 連接 WebSocket
    connect(token) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            console.log('WebSocket already connected');
            return;
        }

        const wsUrl = `${API_CONFIG.WS_URL}${API_CONFIG.ENDPOINTS.WS}?token=${token}`;
        
        try {
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.startHeartbeat();
                
                // 重新訂閱之前的頻道
                this.subscribedChannels.forEach(channelId => {
                    this.subscribeToChannel(channelId);
                });
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

    // 訂閱頻道
    subscribeToChannel(channelId) {
        this.subscribedChannels.add(channelId);
        return this.send({
            type: 'subscribe',
            channel_id: channelId,
            timestamp: Date.now()
        });
    }

    // 取消訂閱頻道
    unsubscribeFromChannel(channelId) {
        this.subscribedChannels.delete(channelId);
        return this.send({
            type: 'unsubscribe',
            channel_id: channelId,
            timestamp: Date.now()
        });
    }

    // 發送正在輸入狀態
    sendTyping(channelId) {
        return this.send({
            type: 'typing',
            channel_id: channelId,
            timestamp: Date.now()
        });
    }

    // 心跳機制
    startHeartbeat() {
        this.stopHeartbeat();
        this.heartbeatInterval = setInterval(() => {
            if (this.isConnected) {
                this.send({
                    type: 'ping',
                    timestamp: Date.now()
                });
            }
        }, 30000); // 每 30 秒發送一次心跳
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

        switch (message.type) {
            case 'pong':
                // 心跳回應
                break;
                
            case 'message':
                // 新訊息
                this.notifyHandlers('message', message.data);
                break;
                
            case 'message_update':
                // 訊息更新
                this.notifyHandlers('message_update', message.data);
                break;
                
            case 'message_delete':
                // 訊息刪除
                this.notifyHandlers('message_delete', message.data);
                break;
                
            case 'typing':
                // 使用者正在輸入
                this.notifyHandlers('typing', message.data);
                break;
                
            case 'user_status':
                // 使用者狀態更新
                this.notifyHandlers('user_status', message.data);
                break;
                
            case 'channel_create':
                // 頻道建立
                this.notifyHandlers('channel_create', message.data);
                break;
                
            case 'channel_update':
                // 頻道更新
                this.notifyHandlers('channel_update', message.data);
                break;
                
            case 'channel_delete':
                // 頻道刪除
                this.notifyHandlers('channel_delete', message.data);
                break;
                
            case 'error':
                // 錯誤訊息
                console.error('WebSocket error:', message.data);
                showNotification(message.data.message || '發生錯誤', 'error');
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
