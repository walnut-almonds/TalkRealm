<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useDMStore } from '@/stores/useDMStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import { useAppStore } from '@/stores/useAppStore.js'
import { randomUUID } from '@/utils/format.js'

const dm = useDMStore()
const ws = useWebSocket()
const store = useAppStore()

const messageListEl = ref(null)
const inputEl = ref(null)
const content = ref('')

const partner = computed(() => {
    const ch = dm.currentDMChannel
    if (!ch) return null
    const currentUserId = store.user?.id
    if (ch.user1 && ch.user1.id !== currentUserId) return ch.user1
    if (ch.user2 && ch.user2.id !== currentUserId) return ch.user2
    return ch.user1 || ch.user2
})

function scrollToBottom() {
    nextTick(() => {
        if (messageListEl.value) {
            messageListEl.value.scrollTop = messageListEl.value.scrollHeight
        }
    })
}

watch(() => dm.dmMessages.length, () => scrollToBottom())
watch(() => dm.currentDMChannel?.id, () => scrollToBottom())

onMounted(() => {
    ws.onMessage(onDMMessage)
    scrollToBottom()
})

onUnmounted(() => {
    ws.offMessage(onDMMessage)
})

function onDMMessage(type, data) {
    if (type === 'dm_message') dm.pushIncomingDM(data)
}

async function send() {
    const text = content.value.trim()
    if (!text || !dm.currentDMChannel) return
    const nonce = randomUUID()
    content.value = ''
    autoResize()
    await dm.sendDM(text, nonce)
}

function autoResize() {
    if (!inputEl.value) return
    inputEl.value.style.height = 'auto'
    inputEl.value.style.height = Math.min(inputEl.value.scrollHeight, 200) + 'px'
}

function onKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        send()
    }
}

function formatTime(ts) {
    return new Date(ts).toLocaleTimeString('zh-TW', { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
    <div class="dm-chat-area" v-if="dm.currentDMChannel">
        <!-- Header -->
        <div class="dm-chat-header">
            <div class="dm-partner-info" v-if="partner">
                <div class="dm-partner-avatar">
                    <img v-if="partner.avatar" :src="partner.avatar" :alt="partner.username" />
                    <span v-else>{{ (partner.username || '?').charAt(0).toUpperCase() }}</span>
                </div>
                <span class="dm-partner-name">{{ partner.display_name || partner.username }}</span>
            </div>
            <span v-else class="dm-partner-name">私人訊息</span>
        </div>

        <!-- Messages -->
        <div class="dm-messages" ref="messageListEl">
            <div v-if="dm.hasMoreMessages" class="load-more">
                <button @click="dm.loadDMMessages(dm.currentDMChannel.id, dm.dmMessages[0]?.id)">載入更多</button>
            </div>
            <div v-if="dm.isLoadingMessages && dm.dmMessages.length === 0" class="dm-loading">
                載入中...
            </div>
            <div
                v-for="msg in dm.dmMessages"
                :key="msg.id || msg.nonce"
                class="dm-message"
            >
                <div class="dm-msg-avatar">
                    <img v-if="msg.sender?.avatar" :src="msg.sender.avatar" :alt="msg.sender.username" />
                    <span v-else>{{ (msg.sender?.username || '?').charAt(0).toUpperCase() }}</span>
                </div>
                <div class="dm-msg-body">
                    <div class="dm-msg-meta">
                        <span class="dm-msg-author">{{ msg.sender?.display_name || msg.sender?.username }}</span>
                        <span class="dm-msg-time">{{ formatTime(msg.created_at) }}</span>
                        <span v-if="msg.is_edited" class="dm-msg-edited">(已編輯)</span>
                    </div>
                    <div class="dm-msg-content">{{ msg.content }}</div>
                </div>
            </div>
            <div v-if="!dm.isLoadingMessages && dm.dmMessages.length === 0" class="dm-no-messages">
                <p>這是你與 {{ partner?.display_name || partner?.username }} 的對話開始</p>
            </div>
        </div>

        <!-- Input -->
        <div class="dm-input-area">
            <textarea
                ref="inputEl"
                v-model="content"
                :placeholder="`傳訊息給 ${partner?.display_name || partner?.username || '...'}`"
                rows="1"
                @input="autoResize"
                @keydown="onKeydown"
            ></textarea>
            <button class="dm-send-btn" @click="send" :disabled="!content.trim()">
                <i class="fas fa-paper-plane"></i>
            </button>
        </div>
    </div>
    <div class="dm-empty-state" v-else>
        <i class="fas fa-comment-dots"></i>
        <p>選擇一個對話開始聊天</p>
    </div>
</template>

<style scoped>
.dm-chat-area {
    display: flex;
    flex-direction: column;
    flex: 1;
    height: 100%;
    background: var(--bg-primary, #36393f);
}

.dm-empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: var(--text-muted, #8e9297);
    gap: 12px;
}

.dm-empty-state i {
    font-size: 48px;
    opacity: 0.4;
}

.dm-chat-header {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color, #40444b);
    display: flex;
    align-items: center;
    min-height: 48px;
    flex-shrink: 0;
}

.dm-partner-info {
    display: flex;
    align-items: center;
    gap: 10px;
}

.dm-partner-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--accent, #5865f2);
    color: #fff;
    font-weight: 600;
    font-size: 14px;
    flex-shrink: 0;
}

.dm-partner-avatar img {
    width: 100%; height: 100%; object-fit: cover;
}

.dm-partner-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary, #dcddde);
}

.dm-messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.load-more {
    display: flex;
    justify-content: center;
    margin-bottom: 8px;
}

.load-more button {
    background: var(--bg-secondary, #2f3136);
    color: var(--text-muted, #8e9297);
    border: none;
    padding: 6px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 13px;
}

.dm-loading, .dm-no-messages {
    text-align: center;
    color: var(--text-muted, #8e9297);
    font-size: 13px;
    padding: 16px;
}

.dm-message {
    display: flex;
    gap: 12px;
    padding: 4px 0;
    align-items: flex-start;
}

.dm-msg-avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--accent, #5865f2);
    color: #fff;
    font-weight: 600;
    font-size: 14px;
}

.dm-msg-avatar img {
    width: 100%; height: 100%; object-fit: cover;
}

.dm-msg-body {
    flex: 1;
}

.dm-msg-meta {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin-bottom: 2px;
}

.dm-msg-author {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #dcddde);
}

.dm-msg-time {
    font-size: 11px;
    color: var(--text-muted, #8e9297);
}

.dm-msg-edited {
    font-size: 11px;
    color: var(--text-muted, #8e9297);
    font-style: italic;
}

.dm-msg-content {
    font-size: 14px;
    color: var(--text-normal, #dcddde);
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
}

.dm-input-area {
    padding: 0 16px 16px;
    display: flex;
    gap: 8px;
    align-items: flex-end;
}

.dm-input-area textarea {
    flex: 1;
    background: var(--bg-input, #40444b);
    border: none;
    border-radius: 8px;
    color: var(--text-primary, #dcddde);
    font-size: 14px;
    padding: 10px 14px;
    resize: none;
    outline: none;
    min-height: 40px;
    max-height: 200px;
    line-height: 1.5;
    font-family: inherit;
}

.dm-input-area textarea::placeholder {
    color: var(--text-muted, #8e9297);
}

.dm-send-btn {
    background: var(--accent, #5865f2);
    border: none;
    border-radius: 8px;
    color: #fff;
    width: 40px;
    height: 40px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.1s;
}

.dm-send-btn:hover:not(:disabled) {
    background: var(--accent-hover, #4752c4);
}

.dm-send-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
}
</style>
