<script setup>
import { computed } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { renderMarkdown } from '@/utils/markdown.js'
import { formatTimestamp, formatFullTimestamp, formatFileSize, escapeHtml, IMAGE_TYPES } from '@/utils/format.js'
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
const isSpeakingInVoice = computed(() => props.isSpeaking)

// ── Image loading with cache ──────────────────────────────────
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

async function openAttachment(fileId) {
  const url = await store.getImageUrl(fileId)
  if (url) window.open(url, '_blank')
  else store.showNotification('無法取得檔案連結', 'error')
}

async function openLightbox(fileId, imgEl) {
  const url = imgEl?.src && !imgEl.src.endsWith('/') ? imgEl.src : await store.getImageUrl(fileId)
  if (url) store.lightboxUrl = url
}

async function deleteMessage() {
  if (!confirm('確定要刪除此訊息？')) return
  try {
    await api.deleteMessage(props.message.id)
  } catch (e) {
    store.showNotification('刪除失敗', 'error')
  }
}
</script>

<template>
  <div :class="['message', { 'message--pending': message._pending, 'message-group-start': !grouped }]">
    <!-- Avatar -->
    <div v-if="!grouped" class="message-avatar" :class="{ 'avatar-speaking': isSpeakingInVoice }">
      <img v-if="avatar" :src="avatar" :alt="nickname" />
      <i v-else class="fas fa-user"></i>
    </div>
    <div v-else class="message-avatar-spacer" aria-hidden="true"></div>

    <!-- Content -->
    <div class="message-content">
      <div v-if="!grouped" class="message-header">
        <span class="message-author">{{ nickname }}</span>
        <span class="message-timestamp" :title="fullTimestamp">{{ timestamp }}</span>
        <span v-if="message.is_edited" class="message-timestamp" style="font-size:11px">(已編輯)</span>
      </div>
      <div class="message-text" v-html="bodyHtml"></div>

      <!-- Attachments -->
      <div v-if="message.attachments?.length" class="message-attachments">
        <template v-for="att in message.attachments" :key="att.id">
          <div
            v-if="att.file && IMAGE_TYPES.has(att.file.content_type)"
            class="message-attachment--image"
            @click="openLightbox(att.file.id, $event.currentTarget.querySelector('img'))"
          >
            <img
              alt=""
              style="opacity:0"
              :ref="el => el && loadImage(att.file.id, el)"
            />
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

      <!-- Actions (shown on hover via CSS or fixed for own messages) -->
      <div v-if="isCurrentUser && message.id" class="message-actions" style="margin-top:2px;display:flex;gap:4px">
        <button class="btn-icon" style="font-size:12px;padding:2px 6px" @click="deleteMessage" title="刪除">
          <i class="fas fa-trash" style="color:var(--danger)"></i>
        </button>
      </div>
    </div>
  </div>
</template>
