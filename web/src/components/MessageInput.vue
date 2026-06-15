<script setup>
import { ref, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import { useFileUpload } from '@/composables/useFileUpload.js'
import { randomUUID } from '@/utils/format.js'
import { api } from '@/api/index.js'
import GifPicker from '@/components/GifPicker.vue'

const store = useAppStore()
const ws = useWebSocket()
const { pendingChips, uploadFile, removeChip, clearChips } = useFileUpload(store)
const { t } = useI18n()

const input = ref(null)
const content = ref('')
const isDragging = ref(false)
const showGifPicker = ref(false)
let dragCounter = 0

// typing throttle
let typingTimeout = null

// ── @ mention autocomplete ────────────────────────────────
const mentionQuery = ref('')   // text after @
const mentionActive = ref(false)
let mentionStartIndex = -1     // caret position when @ was typed
const mentionSelectedIdx = ref(0)

function escapeRegExp(input) {
  return input.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

const SPECIAL_MENTIONS = [
  { id: '@here',     label: '@here',     desc: t('messageInput.mentionHereDesc') },
  { id: '@everyone', label: '@everyone', desc: t('messageInput.mentionEveryoneDesc') },
]

const mentionCandidates = computed(() => {
  const q = mentionQuery.value.toLowerCase()
  const special = SPECIAL_MENTIONS.filter(s => s.label.toLowerCase().includes(q))
  const memberList = store.members
    .filter(m => {
      if (m.user_id === store.user?.id) return false
      const name = (m.user?.nickname || m.user?.username || '').toLowerCase()
      return name.includes(q)
    })
    .slice(0, 8)
    .map(m => ({
      id: m.user_id,
      label: `@${m.user?.nickname || m.user?.username}`,
      desc: m.user?.username || '',
    }))
  return [...special, ...memberList].slice(0, 10)
})

function onInput() {
  autoResize()
  detectMention()
  if (!typingTimeout && store.currentChannel) {
    ws.sendTyping(store.currentChannel.id)
    typingTimeout = setTimeout(() => { typingTimeout = null }, 3000)
  }
}

function detectMention() {
  const el = input.value
  if (!el) return
  const text = content.value
  const caret = el.selectionStart
  // find last @ before caret
  const before = text.slice(0, caret)
  const atIdx = before.lastIndexOf('@')
  if (atIdx === -1) { mentionActive.value = false; return }
  // make sure no space between @ and caret
  const query = before.slice(atIdx + 1)
  if (/\s/.test(query)) { mentionActive.value = false; return }
  mentionStartIndex = atIdx
  mentionQuery.value = query
  mentionActive.value = true
  mentionSelectedIdx.value = 0
}

function pickMention(candidate) {
  const text = content.value
  const before = text.slice(0, mentionStartIndex)
  const after = text.slice(mentionStartIndex + 1 + mentionQuery.value.length)
  let insertion
  if (typeof candidate.id === 'number') {
    // Show friendly text while typing; convert to mention token right before send.
    insertion = `${candidate.label} `
  } else {
    // @here / @everyone
    insertion = `${candidate.id} `
  }
  content.value = before + insertion + after
  mentionActive.value = false
  nextTick(() => {
    const pos = before.length + insertion.length
    input.value?.setSelectionRange(pos, pos)
    input.value?.focus()
  })
}

function toWireMentions(text) {
  let output = text
  const aliasToId = new Map()

  store.members.forEach((m) => {
    if (m.user_id === store.user?.id) return
    const aliases = [m.user?.nickname, m.user?.username]
    aliases.forEach((alias) => {
      const name = (alias || '').trim()
      if (!name) return
      const lower = name.toLowerCase()
      if (lower === 'here' || lower === 'everyone') return
      if (!aliasToId.has(name)) aliasToId.set(name, m.user_id)
    })
  })

  // Prefer longer aliases first so @alexander is not partially matched by @alex.
  const sortedAliases = [...aliasToId.keys()].sort((a, b) => b.length - a.length)
  sortedAliases.forEach((alias) => {
    const uid = aliasToId.get(alias)
    const re = new RegExp(`(^|\\s)@${escapeRegExp(alias)}(?=$|\\s|[.,!?;:])`, 'g')
    output = output.replace(re, (_, prefix) => `${prefix}<@${uid}>`)
  })

  return output
}

function autoResize() {
  if (!input.value) return
  input.value.style.height = 'auto'
  input.value.style.height = Math.min(input.value.scrollHeight, 200) + 'px'
}

async function send() {
  const text = content.value.trim()
  const wireText = toWireMentions(text)
  const fileIds = [...store.pendingFileIds]
  if (!text && fileIds.length === 0) return
  if (!store.currentChannel) return

  const nonce = randomUUID()

  // Optimistic UI
  store.messages.push({
    id: null, nonce,
    channel_id: store.currentChannel.id,
    user_id: store.user.id,
    user: store.user,
    content: text,
    type: 'text',
    is_edited: false,
    attachments: [],
    created_at: new Date().toISOString(),
    _pending: true,
  })

  const savedContent = content.value
  content.value = ''
  await nextTick()
  autoResize()
  input.value?.focus()

  store.pendingFileIds = []
  clearChips()

  try {
    const ok = ws.sendMessage(store.currentChannel.id, wireText, 'text', nonce, fileIds)
    if (!ok) {
      // WS 未連線，退回 REST
      await api.sendMessage(store.currentChannel.id, wireText, 'text', nonce, fileIds)
    }
  } catch (e) {
    console.error('Send failed', e)
    store.showNotification(t('messageInput.sendFailed'), 'error')
    const idx = store.messages.findIndex(m => m.nonce === nonce && m._pending)
    if (idx !== -1) store.messages.splice(idx, 1)
    content.value = savedContent
  }
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

function onKeydown(e) {
  if (mentionActive.value && mentionCandidates.value.length) {
    if (e.key === 'ArrowDown') { e.preventDefault(); mentionSelectedIdx.value = (mentionSelectedIdx.value + 1) % mentionCandidates.value.length; return }
    if (e.key === 'ArrowUp') { e.preventDefault(); mentionSelectedIdx.value = (mentionSelectedIdx.value - 1 + mentionCandidates.value.length) % mentionCandidates.value.length; return }
    if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); pickMention(mentionCandidates.value[mentionSelectedIdx.value]); return }
    if (e.key === 'Escape') { mentionActive.value = false; return }
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

// File via button
function triggerFileInput() {
  document.getElementById('file-input-hidden')?.click()
}

async function onFileSelected(e) {
  const file = e.target.files[0]
  if (!file) return
  e.target.value = ''
  await uploadFile(file)
}

// Drag & drop
const hasDraggedFiles = (e) => {
  const types = e.dataTransfer?.types
  if (!types) return false
  if (typeof types.contains === 'function') return types.contains('Files')
  return Array.from(types).includes('Files')
}

function onDragEnter(e) {
  if (!hasDraggedFiles(e)) return
  e.preventDefault()
  dragCounter++
  isDragging.value = true
}
function onDragLeave() {
  dragCounter--
  if (dragCounter <= 0) { dragCounter = 0; isDragging.value = false }
}
function onDragOver(e) {
  if (!hasDraggedFiles(e)) return
  e.preventDefault()
  e.dataTransfer.dropEffect = 'copy'
}
async function onDrop(e) {
  e.preventDefault()
  dragCounter = 0
  isDragging.value = false
  const files = Array.from(e.dataTransfer?.files ?? [])
  if (!files.length) return
  if (!store.currentChannel) { store.showNotification(t('messageInput.selectChannelFirst'), 'error'); return }
  for (const f of files) await uploadFile(f)
}
</script>

<template>
  <div
    class="message-input-container"
    @dragenter="onDragEnter"
    @dragleave="onDragLeave"
    @dragover="onDragOver"
    @drop="onDrop"
  >
    <!-- File preview chips -->
    <div v-if="pendingChips.length" class="file-preview-area">
      <div v-for="chip in pendingChips" :key="chip.id" class="file-preview-chip">
        <i :class="chip.done ? 'fas fa-paperclip' : 'fas fa-spinner fa-spin'"></i>
        <span class="chip-name">{{ chip.name }}</span>
        <span v-if="!chip.done" class="chip-progress">{{ chip.progress }}%</span>
        <button v-if="chip.done" class="chip-remove" @click="removeChip(chip.id, chip.fileId)">
          <i class="fas fa-times"></i>
        </button>
      </div>
    </div>

    <!-- @ Mention dropdown -->
    <div v-if="mentionActive && mentionCandidates.length" class="mention-dropdown">
      <div
        v-for="(c, i) in mentionCandidates"
        :key="c.id"
        :class="['mention-item', { 'mention-item-selected': i === mentionSelectedIdx }]"
        @mousedown.prevent="pickMention(c)"
      >
        <span class="mention-item-label">{{ c.label }}</span>
        <span class="mention-item-desc">{{ c.desc }}</span>
      </div>
    </div>

    <!-- Input row -->
    <div class="message-input-row">
      <button class="btn-icon" :title="t('messageInput.uploadFile')" @click="triggerFileInput">
        <i class="fas fa-plus-circle"></i>
      </button>
      <textarea
        ref="input"
        v-model="content"
        :placeholder="t('messageInput.placeholder')"
        rows="1"
        @keydown="onKeydown"
        @input="onInput"
      ></textarea>
      <button class="btn-icon" :title="t('messageInput.send')" @click="send">
        <i class="fas fa-paper-plane"></i>
      </button>
      <button class="btn-icon" :class="{ active: showGifPicker }" title="GIF" @click="toggleGifPicker">
        <i class="fas fa-film"></i>
      </button>
    </div>

    <GifPicker
      v-if="showGifPicker"
      class="message-gif-picker"
      @close="showGifPicker = false"
      @select="onSelectGif"
    />

    <!-- Hidden file input -->
    <input
      id="file-input-hidden"
      type="file"
      style="display:none"
      accept="image/*,.pdf,.zip,.mp4,.mp3,.txt,.doc,.docx,.xls,.xlsx,.ppt,.pptx"
      @change="onFileSelected"
    />

    <!-- Drop overlay -->
    <div :class="['drop-overlay', { active: isDragging }]">
      <div class="drop-overlay-inner">
        <i class="fas fa-cloud-upload-alt"></i>
        <p>放開以上傳檔案</p>
      </div>
    </div>
  </div>
</template>
