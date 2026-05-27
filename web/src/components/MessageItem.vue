<script setup>
import { ref, computed } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { useDMStore } from '@/stores/useDMStore.js'
import { renderMarkdown } from '@/utils/markdown.js'
import { formatTimestamp, formatFullTimestamp, formatFileSize, IMAGE_TYPES } from '@/utils/format.js'
import { api } from '@/api/index.js'

const props = defineProps({
  message: Object,
  grouped: Boolean,
  isSpeaking: Boolean,
  isDM: Boolean,
})

const store = useAppStore()
const voiceStore = useVoiceStore()
const dm = useDMStore()

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
    const updated = props.isDM
      ? await api.updateDMMessage(props.message.id, text)
      : await api.updateMessage(props.message.id, text)
    store.handleMessageUpdate({ ...props.message, ...(updated || {}), content: text, is_edited: true })
    dm.handleDMMessageUpdate({ ...props.message, ...(updated || {}), content: text, is_edited: true })
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
    if (props.isDM) {
      await api.deleteDMMessage(props.message.id)
    } else {
      await api.deleteMessage(props.message.id)
    }
    store.handleMessageDelete({ message_id: props.message.id })
    dm.handleDMMessageDelete({ id: props.message.id })
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

// ── Translation & Guess ───────────────────────────────────────
const LANG_LABELS = { zh: '中文（簡體）', 'zh-tw': '繁體中文', ja: '日本語', en: 'English' }

const isTextMessage = computed(() => !props.message.type || props.message.type === 'text')
const showTranslationSection = computed(() =>
  isTextMessage.value && props.message.id && !props.message._pending
)

const translation = computed(() => {
  return store.translationCache.get(props.message.id) ?? dm.dmTranslationCache.get(props.message.id) ?? null
})
const isTranslationLoading = computed(() => {
  return store.translationLoadingSet.has(props.message.id) || dm.dmTranslationLoadingSet.has(props.message.id)
})

const preferredLang = computed(() => store.user?.preferred_lang || 'zh')

// Which language to display (preferred unless message is already in that lang)
const displayLang = computed(() => {
  if (!translation.value) return preferredLang.value
  const orig = translation.value.original_lang
  if (orig === preferredLang.value) {
    return ['zh', 'zh-tw', 'ja', 'en'].find(l => l !== orig) || 'en'
  }
  return preferredLang.value
})

const translatedText = computed(() => {
  if (!translation.value) return null
  if (!translation.value?.translations) return null
  if (translation.value.original_lang === preferredLang.value) {
    const fallback = ['zh', 'zh-tw', 'ja', 'en'].find(l => l !== preferredLang.value)
    return fallback ? (translation.value.translations[fallback] || null) : null
  }
  return translation.value.translations[preferredLang.value] || null
})

const translationVisible = ref(false)
const translationDismissed = ref(false)
const guessMode = ref(false)
const guessInput = ref('')
const guessResult = ref(null)  // { is_correct, similarity_score, correct_content }
const isGuessing = ref(false)

async function fetchTranslation() {
  translationDismissed.value = false
  if (translation.value || isTranslationLoading.value) return
  store.translationLoadingSet.add(props.message.id)
  dm.dmTranslationLoadingSet.add(props.message.id)
  try {
    const result = props.isDM
      ? await api.ensureDMTranslation(props.message.id, preferredLang.value)
      : await api.ensureTranslation(props.message.id, preferredLang.value)
    if (result?.status === 'processing') return
    const payload = {
      message_id: props.message.id,
      original_lang: result.original_lang,
      translations: result.translations || { zh: result.content_zh, 'zh-tw': result.content_zh_tw, ja: result.content_ja, en: result.content_en },
    }
    store.handleTranslationReady(payload)
    dm.handleDMTranslationReady(payload)
  } catch {
    store.translationLoadingSet.delete(props.message.id)
    dm.dmTranslationLoadingSet.delete(props.message.id)
  }
}

async function submitGuess() {
  if (!guessInput.value.trim() || isGuessing.value) return
  isGuessing.value = true
  try {
    const result = await api.submitGuess(props.message.id, guessInput.value.trim(), displayLang.value)
    guessResult.value = result
    guessMode.value = false
  } catch (e) {
    store.showNotification('猜測失敗：' + (e.message || '未知錯誤'), 'error')
  } finally {
    isGuessing.value = false
  }
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

      <!-- Translation section -->
      <div v-if="showTranslationSection && !translationDismissed" class="translation-section">
        <!-- In-flight: waiting for translation_ready WS event -->
        <div v-if="isTranslationLoading && !translation" class="translation-loading">
          <i class="fas fa-circle-notch fa-spin"></i>
          <span>翻譯中...</span>
        </div>

        <!-- Translation ready -->
        <template v-else-if="translation && translatedText">
          <div class="translation-bar">
            <span class="translation-lang-badge">{{ LANG_LABELS[displayLang] }}</span>
            <span
              class="translation-text"
              :class="{ 'translation-text--blurred': !translationVisible && !guessResult }"
              @click="!translationVisible && !guessResult && (translationVisible = true)"
            >{{ translatedText }}</span>
            <div class="translation-actions">
              <button
                v-if="!guessMode && !guessResult"
                class="trans-btn"
                :title="translationVisible ? '隱藏譯文' : '顯示譯文'"
                @click="translationVisible = !translationVisible"
              >
                <i :class="['fas', translationVisible ? 'fa-eye-slash' : 'fa-eye']"></i>
              </button>
              <button
                v-if="!translationVisible && !guessResult"
                class="trans-btn trans-btn--guess"
                title="猜猜看"
                @click="guessMode = !guessMode"
              >
                <i class="fas fa-question-circle"></i>
              </button>
              <button
                class="trans-btn"
                title="收起翻譯"
                @click="translationDismissed = true; translationVisible = false; guessMode = false"
              >
                <i class="fas fa-xmark"></i>
              </button>
            </div>
          </div>

          <!-- Guess input -->
          <div v-if="guessMode && !guessResult" class="guess-area">
            <input
              v-model="guessInput"
              class="guess-input"
              :placeholder="`猜猜 ${LANG_LABELS[displayLang]} 的意思...`"
              @keydown.enter="submitGuess"
            />
            <button class="btn-sm btn-primary" :disabled="isGuessing" @click="submitGuess">
              {{ isGuessing ? '送出中...' : '送出' }}
            </button>
            <button class="btn-sm btn-secondary" @click="guessMode = false; guessInput = ''">取消</button>
          </div>

          <!-- Guess result -->
          <div
            v-if="guessResult"
            class="guess-result"
            :class="guessResult.is_correct ? 'guess-result--correct' : 'guess-result--wrong'"
          >
            <i :class="['fas', guessResult.is_correct ? 'fa-check-circle' : 'fa-times-circle']"></i>
            <span v-if="guessResult.is_correct">猜對了！</span>
            <span v-else>
              相似度 {{ Math.round((guessResult.similarity_score || 0) * 100) }}%
              <span v-if="guessResult.correct_content" class="guess-answer">
                正解：{{ guessResult.correct_content }}
              </span>
            </span>
          </div>
        </template>

      </div>
    </div>

    <!-- Hover action bar (right side) -->
    <div
      v-if="message.id && !isEditing && (isCurrentUser || (showTranslationSection && (!translation || translationDismissed) && !isTranslationLoading))"
      class="message-actions-bar"
    >
      <button
        v-if="showTranslationSection && (!translation || translationDismissed) && !isTranslationLoading"
        class="msg-action-btn"
        title="翻譯"
        @click="fetchTranslation"
      >
        <i class="fas fa-language"></i>
      </button>
      <template v-if="isCurrentUser">
        <button class="msg-action-btn" title="編輯" @click="startEdit">
          <i class="fas fa-pencil"></i>
        </button>
        <button class="msg-action-btn danger" title="刪除" @click="deleteMessage">
          <i class="fas fa-trash"></i>
        </button>
      </template>
    </div>
  </div>
</template>
