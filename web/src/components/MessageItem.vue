<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { useDMStore } from '@/stores/useDMStore.js'
import { renderMarkdown, setMentionResolver } from '@/utils/markdown.js'
import { formatTimestamp, formatFullTimestamp, formatFileSize, IMAGE_TYPES } from '@/utils/format.js'
import { api } from '@/api/index.js'
import LinkPreview from '@/components/LinkPreview.vue'

const props = defineProps({
  message: Object,
  grouped: Boolean,
  isSpeaking: Boolean,
  isDM: Boolean,
})

const store = useAppStore()
const voiceStore = useVoiceStore()
const dm = useDMStore()
const { t } = useI18n()

// Set mention resolver once (idempotent — same fn reference is fine)
setMentionResolver(store.resolveMessageUser)

const user = computed(() => store.resolveMessageUser(props.message))
const nickname = computed(() => user.value?.nickname || user.value?.username || t('chat.unknownUser'))
const avatar = computed(() => user.value?.avatar || null)
const timestamp = computed(() => formatTimestamp(props.message.created_at))
const fullTimestamp = computed(() => formatFullTimestamp(props.message.created_at))
const _rendered = computed(() => renderMarkdown(props.message.content))
const bodyHtml = computed(() => _rendered.value.html)
const bodyUrls = computed(() => _rendered.value.urls)
const isCurrentUser = computed(() => store.user?.id === props.message.user_id)

// Detect if current user is mentioned in this message
const isMentioned = computed(() => {
  const uid = store.user?.id
  if (!uid || !props.message.content) return false
  return props.message.content.includes(`<@${uid}>`) ||
    /\B@(here|everyone)\b/.test(props.message.content)
})

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
    store.showNotification(t('notifications.editFailed'), 'error')
  }
}

function onEditKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); saveEdit() }
  if (e.key === 'Escape') cancelEdit()
}

// ── Delete ────────────────────────────────────────────────────
async function deleteMessage() {
  if (!confirm(t('chat.deleteConfirm'))) return
  try {
    if (props.isDM) {
      await api.deleteDMMessage(props.message.id)
    } else {
      await api.deleteMessage(props.message.id)
    }
    store.handleMessageDelete({ message_id: props.message.id })
    dm.handleDMMessageDelete({ id: props.message.id })
  } catch (e) {
    store.showNotification(t('notifications.deleteFailed'), 'error')
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
  else store.showNotification(t('notifications.fileUrlFailed'), 'error')
}

// ── Translation & Guess ───────────────────────────────────────
const langLabels = computed(() => ({
  zh: t('settings.langZh'),
  'zh-tw': t('settings.langZhTw'),
  ja: t('settings.langJa'),
  en: t('settings.langEn'),
}))

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
    store.showNotification(`${t('notifications.guessFailed')}: ${e.message || t('common.unknownError')}`, 'error')
  } finally {
    isGuessing.value = false
  }
}
</script>

<template>
  <div :class="['message', { 'message--pending': message._pending, 'message-group-start': !grouped, 'message--mentioned': isMentioned }]">
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
        <span v-if="message.is_edited" class="message-edited">({{ t('chat.edited') }})</span>
      </div>

      <!-- Normal / Edit mode -->
      <template v-if="!isEditing">
        <div class="message-text" v-html="bodyHtml"></div>
        <LinkPreview v-if="bodyUrls.length" :urls="bodyUrls" />
      </template>
      <div v-else class="message-edit-box">
        <textarea
          :id="`edit-${message.id}`"
          v-model="editContent"
          class="message-edit-input"
          rows="3"
          @keydown="onEditKeydown"
        ></textarea>
        <div class="message-edit-hints">
          <span>{{ t('chat.editHint') }}</span>
          <div style="display:flex;gap:6px">
            <button class="btn-secondary btn-sm" @click="cancelEdit">{{ t('common.cancel') }}</button>
            <button class="btn-primary btn-sm" @click="saveEdit">{{ t('common.save') }}</button>
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
            <button class="attachment-download" @click="openAttachment(att.file.id)" :title="t('chat.download')">
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
          <span>{{ t('chat.translating') }}</span>
        </div>

        <!-- Translation ready -->
        <template v-else-if="translation && translatedText">
          <div class="translation-bar">
            <span class="translation-lang-badge">{{ langLabels[displayLang] }}</span>
            <span
              class="translation-text"
              :class="{ 'translation-text--blurred': !translationVisible && !guessResult }"
              @click="!translationVisible && !guessResult && (translationVisible = true)"
            >{{ translatedText }}</span>
            <div class="translation-actions">
              <button
                v-if="!guessMode && !guessResult"
                class="trans-btn"
                :title="translationVisible ? t('chat.hideTranslation') : t('chat.showTranslation')"
                @click="translationVisible = !translationVisible"
              >
                <i :class="['fas', translationVisible ? 'fa-eye-slash' : 'fa-eye']"></i>
              </button>
              <button
                v-if="!translationVisible && !guessResult"
                class="trans-btn trans-btn--guess"
                :title="t('chat.guessAction')"
                @click="guessMode = !guessMode"
              >
                <i class="fas fa-question-circle"></i>
              </button>
              <button
                class="trans-btn"
                :title="t('chat.collapseTranslation')"
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
              :placeholder="t('chat.guessPlaceholder', { lang: langLabels[displayLang] })"
              @keydown.enter="submitGuess"
            />
            <button class="btn-sm btn-primary" :disabled="isGuessing" @click="submitGuess">
              {{ isGuessing ? t('common.submitting') : t('common.submit') }}
            </button>
            <button class="btn-sm btn-secondary" @click="guessMode = false; guessInput = ''">{{ t('common.cancel') }}</button>
          </div>

          <!-- Guess result -->
          <div
            v-if="guessResult"
            class="guess-result"
            :class="guessResult.is_correct ? 'guess-result--correct' : 'guess-result--wrong'"
          >
            <i :class="['fas', guessResult.is_correct ? 'fa-check-circle' : 'fa-times-circle']"></i>
            <span v-if="guessResult.is_correct">{{ t('chat.guessCorrect') }}</span>
            <span v-else>
              {{ t('chat.similarity') }} {{ Math.round((guessResult.similarity_score || 0) * 100) }}%
              <span v-if="guessResult.correct_content" class="guess-answer">
                {{ t('chat.correctAnswer') }}{{ guessResult.correct_content }}
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
        :title="t('chat.translate')"
        @click="fetchTranslation"
      >
        <i class="fas fa-language"></i>
      </button>
      <template v-if="isCurrentUser">
        <button class="msg-action-btn" :title="t('chat.edit')" @click="startEdit">
          <i class="fas fa-pencil"></i>
        </button>
        <button class="msg-action-btn danger" :title="t('chat.delete')" @click="deleteMessage">
          <i class="fas fa-trash"></i>
        </button>
      </template>
    </div>
  </div>
</template>
