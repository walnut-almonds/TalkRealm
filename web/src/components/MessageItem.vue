<script setup>
import { ref, computed } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { renderMarkdown } from '@/utils/markdown.js'
import { formatTimestamp, formatFullTimestamp, formatFileSize, IMAGE_TYPES } from '@/utils/format.js'
import { api } from '@/api/index.js'

const props = defineProps({
  message: Object,
  grouped: Boolean,
  isSpeaking: Boolean,
})

const store = useAppStore()
const voiceStore = useVoiceStore()

const user = computed(() => store.resolveMessageUser(props.message))
const nickname = computed(() => user.value?.nickname || user.value?.username || 'Unknown')
const avatar = computed(() => user.value?.avatar || null)
const timestamp = computed(() => formatTimestamp(props.message.created_at))
const fullTimestamp = computed(() => formatFullTimestamp(props.message.created_at))
const bodyHtml = computed(() => renderMarkdown(props.message.content))
const isCurrentUser = computed(() => store.user?.id === props.message.user_id)

// ── Inline edit ───────────────────────────────────────────────
const isEditing = ref(false)
const editContent = ref('')

function startEdit() {
  editContent.value = props.message.content
  isEditing.value = true
  setTimeout(() => document.getElementById(`edit-${props.message.id}`)?.focus(), 30)
}

function cancelEdit() {
  isEditing.value = false
  editContent.value = ''
}

async function saveEdit() {
  const text = editContent.value.trim()
  if (!text || text === props.message.content) { cancelEdit(); return }
  try {
    const updated = await api.updateMessage(props.message.id, text)
    store.handleMessageUpdate({ ...props.message, ...(updated || {}), content: text, is_edited: true })
    cancelEdit()
  } catch (e) {
    store.showNotification('編輯失敗', 'error')
  }
}

function onEditKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); saveEdit() }
  if (e.key === 'Escape') cancelEdit()
}

// ── Delete ────────────────────────────────────────────────────
async function deleteMessage() {
  if (!confirm('確定要刪除此訊息？')) return
  try {
    await api.deleteMessage(props.message.id)
    // Immediately update local state — don't wait for WS echo
    store.handleMessageDelete({ message_id: props.message.id })
  } catch (e) {
    store.showNotification('刪除失敗', 'error')
  }
}

// ── Image loading ─────────────────────────────────────────────
async function loadImage(fileId, imgEl) {
  const url = await store.getImageUrl(fileId)
  if (url && imgEl) {
    imgEl.src = url
    imgEl.onload = () => {
      imgEl.style.opacity = '1'
      imgEl.closest('.message-attachment--image')?.classList.add('loaded')
    }
    imgEl.onerror = async () => {
      if (imgEl.dataset.refreshed) return
      imgEl.dataset.refreshed = '1'
      const fresh = await store.getImageUrl(fileId, true)
      if (fresh) imgEl.src = fresh
    }
  }
}

async function openLightbox(fileId, imgEl) {
  const url = imgEl?.src && !imgEl.src.endsWith('/') ? imgEl.src : await store.getImageUrl(fileId)
  if (url) store.lightboxUrl = url
}

async function openAttachment(fileId) {
  const url = await store.getImageUrl(fileId)
  if (url) window.open(url, '_blank')
  else store.showNotification('無法取得檔案連結', 'error')
}
</script>

<template>
  <div :class="['message', { 'message--pending': message._pending, 'message-group-start': !grouped }]">
    <!-- Avatar -->
    <div v-if="!grouped" class="message-avatar" :class="{ 'avatar-speaking': isSpeaking }">
      <img v-if="avatar" :src="avatar" :alt="nickname" />
      <i v-else class="fas fa-user"></i>
    </div>
    <div v-else class="message-avatar-spacer" aria-hidden="true"></div>

    <!-- Content -->
    <div class="message-content">
      <div v-if="!grouped" class="message-header">
        <span class="message-author">{{ nickname }}</span>
        <span class="message-timestamp" :title="fullTimestamp">{{ timestamp }}</span>
        <span v-if="message.is_edited" class="message-edited">(已編輯)</span>
      </div>

      <!-- Normal / Edit mode -->
      <div v-if="!isEditing" class="message-text" v-html="bodyHtml"></div>
      <div v-else class="message-edit-box">
        <textarea
          :id="`edit-${message.id}`"
          v-model="editContent"
          class="message-edit-input"
          rows="3"
          @keydown="onEditKeydown"
        ></textarea>
        <div class="message-edit-hints">
          <span>Enter 儲存 · Esc 取消</span>
          <div style="display:flex;gap:6px">
            <button class="btn-secondary btn-sm" @click="cancelEdit">取消</button>
            <button class="btn-primary btn-sm" @click="saveEdit">儲存</button>
          </div>
        </div>
      </div>

      <!-- Attachments -->
      <div v-if="message.attachments?.length" class="message-attachments">
        <template v-for="att in message.attachments" :key="att.id">
          <div
            v-if="att.file && IMAGE_TYPES.has(att.file.content_type)"
            class="message-attachment--image"
            @click="openLightbox(att.file.id, $event.currentTarget.querySelector('img'))"
          >
            <img alt="" style="opacity:0" :ref="el => el && loadImage(att.file.id, el)" />
            <div class="img-overlay"><i class="fas fa-expand"></i></div>
          </div>
          <div v-else-if="att.file" class="message-attachment--file">
            <i class="fas fa-file" style="color:var(--text-muted)"></i>
            <div class="attachment-info">
              <span class="attachment-name">{{ att.file.filename }}</span>
              <span class="attachment-size">{{ formatFileSize(att.file.size) }}</span>
            </div>
            <button class="attachment-download" @click="openAttachment(att.file.id)" title="下載">
              <i class="fas fa-download"></i>
            </button>
          </div>
        </template>
      </div>
    </div>

    <!-- Hover action bar (right side) — only for own non-pending messages -->
    <div v-if="isCurrentUser && message.id && !isEditing" class="message-actions-bar">
      <button class="msg-action-btn" title="編輯" @click="startEdit">
        <i class="fas fa-pencil"></i>
      </button>
      <button class="msg-action-btn danger" title="刪除" @click="deleteMessage">
        <i class="fas fa-trash"></i>
      </button>
    </div>
  </div>
</template>
