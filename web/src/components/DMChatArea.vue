<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDMStore } from '@/stores/useDMStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import { useAppStore } from '@/stores/useAppStore.js'
import { useFileUpload } from '@/composables/useFileUpload.js'
import { randomUUID } from '@/utils/format.js'
import MessageItem from '@/components/MessageItem.vue'
import GifPicker from '@/components/GifPicker.vue'

const emit = defineEmits(['open-sidebar'])

const dm = useDMStore()
const ws = useWebSocket()
const store = useAppStore()
const { t } = useI18n()

const messageListEl = ref(null)
const inputEl = ref(null)
const fileInputEl = ref(null)
const content = ref('')
const showGifPicker = ref(false)

const partner = computed(() => {
    const ch = dm.currentDMChannel
    if (!ch) return null
    const currentUserId = store.user?.id
    if (ch.participants) {
        const p = ch.participants.find(p => p.user_id !== currentUserId)
        return p?.user || null
    }
    return null
})

// ── File upload (DM context) ──────────────────────────────────
const { pendingChips, uploadFile, removeChip, clearChips } = useFileUpload({
    checkReady: () => !!dm.currentDMChannel,
    onNotReady: () => store.showNotification(t('dm.selectConversationFirst'), 'error'),
    addFileId: (id) => dm.dmPendingFileIds.push(id),
    removeFileId: (id) => {
        const idx = dm.dmPendingFileIds.indexOf(id)
        if (idx !== -1) dm.dmPendingFileIds.splice(idx, 1)
    },
    showNotification: (msg, type) => store.showNotification(msg, type),
})

function onFileInput(e) {
    const files = Array.from(e.target.files || [])
    files.forEach(f => uploadFile(f))
    e.target.value = ''
}

function onDrop(e) {
    e.preventDefault()
    const files = Array.from(e.dataTransfer.files || [])
    files.forEach(f => uploadFile(f))
}

function onDragover(e) { e.preventDefault() }

// ── Scroll ────────────────────────────────────────────────────
function scrollToBottom() {
    nextTick(() => {
        if (messageListEl.value) {
            messageListEl.value.scrollTop = messageListEl.value.scrollHeight
        }
    })
}

watch(() => dm.dmMessages.length, () => scrollToBottom())
watch(() => dm.currentDMChannel?.id, () => scrollToBottom())

// ── Message grouping ─────────────────────────────────────────
function isGrouped(index) {
    if (index === 0) return false
    const cur = dm.dmMessages[index]
    const prev = dm.dmMessages[index - 1]
    if (!cur || !prev) return false
    const sameAuthor = cur.user_id === prev.user_id
    const timeDiff = new Date(cur.created_at) - new Date(prev.created_at)
    return sameAuthor && timeDiff < 5 * 60 * 1000
}

// ── WS events ─────────────────────────────────────────────────
onMounted(() => {
    ws.onMessage(onWSMessage)
    scrollToBottom()
})

onUnmounted(() => {
    ws.offMessage(onWSMessage)
})

function onWSMessage(type, data) {
    const channelId = data?.channel_id || data?.dm_channel_id
    const isDMChannel = (id) => dm.dmChannels.some(c => c.id === id)
    if ((type === 'message_create' || type === 'dm_message' || type === 'dm_message_create') && isDMChannel(channelId)) dm.pushIncomingDM(data)
    else if ((type === 'message_update' || type === 'dm_message_update') && isDMChannel(channelId)) dm.handleDMMessageUpdate(data)
    else if ((type === 'message_delete' || type === 'dm_message_delete') && channelId && isDMChannel(channelId)) dm.handleDMMessageDelete(data)
    else if (type === 'message_reaction' && isDMChannel(channelId)) dm.handleDMMessageReaction(data)
    else if ((type === 'translation_ready' || type === 'dm_translation_ready') && data.message_id) dm.handleDMTranslationReady(data)
}

// ── Send ──────────────────────────────────────────────────────
async function send() {
    const text = content.value.trim()
    const fileIds = [...dm.dmPendingFileIds]
    if (!text && fileIds.length === 0) return
    if (!dm.currentDMChannel) return
    const nonce = randomUUID()
    content.value = ''
    dm.dmPendingFileIds.splice(0)
    clearChips()
    autoResize()
    await dm.sendDM(text, nonce, fileIds)
}

function toggleGifPicker() {
    showGifPicker.value = !showGifPicker.value
}

function onSelectGif(gif) {
    if (!gif?.url) return
    const draft = content.value.trim()
    content.value = draft ? `${draft}\n${gif.url}` : gif.url
    showGifPicker.value = false
    send()
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
</script>

<template>
    <div class="dm-chat-area" v-if="dm.currentDMChannel" @drop="onDrop" @dragover="onDragover">
        <!-- Header -->
        <div class="dm-chat-header">
            <button class="mobile-hamburger" @click="emit('open-sidebar')" :title="t('dm.list')">
                <i class="fas fa-bars"></i>
            </button>
            <div class="dm-partner-info" v-if="partner">
                <div class="dm-partner-avatar">
                    <img v-if="partner.avatar" :src="partner.avatar" :alt="partner.username" />
                    <span v-else>{{ (partner.username || '?').charAt(0).toUpperCase() }}</span>
                </div>
                <span class="dm-partner-name">{{ partner.display_name || partner.username }}</span>
            </div>
            <span v-else class="dm-partner-name">{{ t('dm.privateMessages') }}</span>
        </div>

        <!-- Messages -->
        <div class="dm-messages" ref="messageListEl">
            <div v-if="dm.hasMoreMessages" class="load-more">
                <button @click="dm.loadDMMessages(dm.currentDMChannel.id, dm.dmMessages[0]?.id)">{{ t('dm.loadMore') }}</button>
            </div>
            <div v-if="dm.isLoadingMessages && dm.dmMessages.length === 0" class="dm-loading">
                {{ t('dm.loading') }}
            </div>
            <MessageItem
                v-for="(msg, idx) in dm.dmMessages"
                :key="msg.id || msg.nonce"
                :message="msg"
                :grouped="isGrouped(idx)"
                :is-d-m="true"
            />
            <div v-if="!dm.isLoadingMessages && dm.dmMessages.length === 0" class="dm-no-messages">
                <p>{{ t('dm.conversationStart', { name: partner?.display_name || partner?.username }) }}</p>
            </div>
        </div>

        <!-- Pending file chips -->
        <div v-if="pendingChips.length > 0" class="dm-file-chips">
            <div v-for="chip in pendingChips" :key="chip.id" class="file-chip">
                <span class="file-chip-name">{{ chip.name }}</span>
                <span v-if="!chip.done" class="file-chip-progress">{{ chip.progress }}%</span>
                <button v-else class="file-chip-remove" @click="removeChip(chip.id, chip.fileId)">
                    <i class="fas fa-times"></i>
                </button>
            </div>
        </div>

        <!-- Input -->
        <div class="dm-input-area">
            <label class="dm-attach-btn" :title="t('dm.uploadFile')">
                <i class="fas fa-paperclip"></i>
                <input ref="fileInputEl" type="file" multiple style="display:none" @change="onFileInput" />
            </label>
            <textarea
                ref="inputEl"
                v-model="content"
                :placeholder="t('dm.messageTo', { name: partner?.display_name || partner?.username || '...' })"
                rows="1"
                @input="autoResize"
                @keydown="onKeydown"
            ></textarea>
            <button class="dm-send-btn" @click="send" :disabled="!content.trim() && dm.dmPendingFileIds.length === 0">
                <i class="fas fa-paper-plane"></i>
            </button>
            <button class="dm-gif-btn" :class="{ active: showGifPicker }" title="GIF" @click="toggleGifPicker">
                <i class="fas fa-film"></i>
            </button>
            <GifPicker
                v-if="showGifPicker"
                class="dm-gif-picker"
                @close="showGifPicker = false"
                @select="onSelectGif"
            />
        </div>
    </div>
    <div class="dm-empty-state" v-else>
        <i class="fas fa-comment-dots"></i>
        <p>{{ t('dm.selectToStart') }}</p>
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
    border-radius: var(--radius);
    cursor: pointer;
    font-size: 13px;
}

.dm-loading, .dm-no-messages {
    text-align: center;
    color: var(--text-muted, #8e9297);
    font-size: 13px;
    padding: 16px;
}

.dm-file-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 0 16px 8px;
}

.file-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-secondary, #2f3136);
    border-radius: var(--radius);
    padding: 4px 10px;
    font-size: 12px;
    color: var(--text-normal, #dcddde);
}

.file-chip-name {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.file-chip-progress {
    color: var(--text-muted, #8e9297);
}

.file-chip-remove {
    background: none;
    border: none;
    color: var(--text-muted, #8e9297);
    cursor: pointer;
    padding: 0;
    line-height: 1;
}

.file-chip-remove:hover { color: var(--text-primary, #dcddde); }

.dm-input-area {
    padding: 0 16px 16px;
    display: flex;
    gap: 8px;
    align-items: flex-end;
    position: relative;
}

.dm-attach-btn {
    background: none;
    border: none;
    color: var(--text-muted, #8e9297);
    cursor: pointer;
    font-size: 18px;
    padding: 0 4px 8px;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    transition: color 0.1s;
}

.dm-attach-btn:hover { color: var(--text-primary, #dcddde); }

.dm-input-area textarea {
    flex: 1;
    background: var(--bg-input, #40444b);
    border: none;
    border-radius: var(--radius);
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
    border-radius: var(--radius);
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

.dm-gif-btn {
    background: var(--bg-secondary, #2f3136);
    border: none;
    border-radius: var(--radius);
    color: var(--text-primary, #dcddde);
    width: 40px;
    height: 40px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.1s, color 0.1s;
}

.dm-gif-btn:hover {
    background: var(--bg-hover, #4f545c);
}

.dm-gif-btn.active {
    color: var(--accent, #5865f2);
}

.dm-gif-picker {
    position: absolute;
    right: 16px;
    bottom: 64px;
    z-index: 20;
}
</style>
