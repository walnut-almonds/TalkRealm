import { ref } from 'vue'
import { WS_URL } from '@/api/index.js'

// Singleton WS manager
let instance = null

export function useWebSocket() {
    if (instance) return instance
    instance = createWebSocketManager()
    return instance
}

function createWebSocketManager() {
    const isConnected = ref(false)
    const handlers = []

    let ws = null
    let token = null
    let heartbeatTimer = null
    let reconnectAttempts = 0
    const MAX_RECONNECT = 5
    const subscribedChannels = new Set()

    function connect(t) {
        if (ws?.readyState === WebSocket.OPEN) return
        token = t
        try {
            ws = new WebSocket(WS_URL)
            ws.onopen = () => { /* wait for hello */ }
            ws.onmessage = (e) => {
                try { handleMessage(JSON.parse(e.data)) } catch { }
            }
            ws.onerror = (e) => console.error('[WS] error', e)
            ws.onclose = () => {
                isConnected.value = false
                stopHeartbeat()
                scheduleReconnect()
            }
        } catch (e) {
            console.error('[WS] connect failed', e)
        }
    }

    function disconnect() {
        reconnectAttempts = MAX_RECONNECT
        ws?.close()
        ws = null
        stopHeartbeat()
        isConnected.value = false
    }

    function scheduleReconnect() {
        if (reconnectAttempts >= MAX_RECONNECT) {
            notify('reconnect_failed', {})
            return
        }
        reconnectAttempts++
        setTimeout(() => connect(token), 1000 * reconnectAttempts)
    }

    function send(msg) {
        if (ws?.readyState !== WebSocket.OPEN) return false
        try { ws.send(JSON.stringify(msg)); return true } catch { return false }
    }

    function sendIdentify(channels = []) {
        return send({ op: 'identify', d: { token, channels: [...channels] } })
    }

    function subscribeToChannel(channelId) {
        subscribedChannels.add(channelId)
        return send({ op: 'subscribe', d: { channel_id: channelId } })
    }

    function unsubscribeFromChannel(channelId) {
        subscribedChannels.delete(channelId)
        return send({ op: 'unsubscribe', d: { channel_id: channelId } })
    }

    function sendTyping(channelId) {
        return send({ op: 'typing_start', d: { channel_id: channelId } })
    }

    function sendMessage(channelId, content, type = 'text', nonce = '', fileIds = []) {
        return send({ op: 'send_message', d: { channel_id: channelId, content, type, nonce, file_ids: fileIds } })
    }

    function sendVoiceStateUpdate(channelId, action) {
        return send({ op: 'voice_state_update', d: { channel_id: channelId, action } })
    }

    function startHeartbeat(interval = 30000) {
        stopHeartbeat()
        heartbeatTimer = setInterval(() => {
            if (isConnected.value) send({ op: 'heartbeat' })
        }, interval)
    }

    function stopHeartbeat() {
        clearInterval(heartbeatTimer)
        heartbeatTimer = null
    }

    function handleMessage(msg) {
        switch (msg.op) {
            case 'hello':
                startHeartbeat(msg.d?.heartbeat_interval || 30000)
                sendIdentify(subscribedChannels)
                break
            case 'ready':
                isConnected.value = true
                reconnectAttempts = 0
                notify('ready', msg.d)
                break
            case 'heartbeat_ack': break
            case 'message_create': notify('message', msg.d); break
            case 'message_update': notify('message_update', msg.d); break
            case 'message_delete': notify('message_delete', msg.d); break
            case 'typing_start': notify('typing', msg.d); break
            case 'presence_update': notify('user_status', msg.d); break
            case 'channel_create': notify('channel_create', msg.d); break
            case 'channel_update': notify('channel_update', msg.d); break
            case 'channel_delete': notify('channel_delete', msg.d); break
            case 'guild_update': notify('guild_update', msg.d); break
            case 'guild_delete': notify('guild_delete', msg.d); break
            case 'guild_member_add': notify('member_add', msg.d); break
            case 'guild_member_remove': notify('member_remove', msg.d); break
            case 'guild_member_update': notify('member_update', msg.d); break
            case 'voice_state_update': notify('voice_state_update', msg.d); break
            case 'error':
                console.error('[WS] server error', msg.d)
                notify('error', msg.d)
                break
        }
    }

    function notify(type, data) {
        handlers.forEach(h => { try { h(type, data) } catch (e) { console.error('[WS] handler error', e) } })
    }

    function onMessage(handler) { handlers.push(handler) }
    function offMessage(handler) {
        const i = handlers.indexOf(handler); if (i > -1) handlers.splice(i, 1)
    }

    return {
        isConnected,
        connect, disconnect,
        send, sendIdentify, subscribeToChannel, unsubscribeFromChannel,
        sendTyping, sendMessage, sendVoiceStateUpdate,
        onMessage, offMessage,
    }
}
