<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import { useFileUpload } from '@/composables/useFileUpload.js'
import { api } from '@/api/index.js'
import MessageItem from './MessageItem.vue'

const emit = defineEmits(['open-sidebar'])
const store = useAppStore()
const ws = useWebSocket()
const { t } = useI18n()

// posts: newest-first. Each post is the PostResponse (flattened Message + like_count,
// comment_count, liked_by_me) plus lazy UI fields: _expanded, _comments (null until loaded),
// _commentsLoading, _commentDraft.
const posts = ref([])
const hasMore = ref(false)
const loading = ref(false)
const container = ref(null)

// ── Compose box (top) ─────────────────────────────────────────
const composeText = ref('')
const composeFileIds = ref([])
const inputEl = ref(null)

// Twitter-style: textarea grows with content so you always see what you typed.
function autoGrow() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 240) + 'px'
}
function resetHeight() {
  if (inputEl.value) inputEl.value.style.height = 'auto'
}
const { pendingChips, uploadFile, removeChip, clearChips } = useFileUpload({
  checkReady: () => !!store.currentChannel,
  onNotReady: () => store.showNotification(t('chat.selectChannelFirst'), 'error'),
  addFileId: (id) => composeFileIds.value.push(id),
  removeFileId: (id) => {
    const i = composeFileIds.value.indexOf(id)
    if (i !== -1) composeFileIds.value.splice(i, 1)
  },
  showNotification: (m, ty) => store.showNotification(m, ty),
})

function onFileInput(e) {
  Array.from(e.target.files || []).forEach(f => uploadFile(f))
  e.target.value = ''
}

const posting = ref(false)
async function submitPost() {
  const text = composeText.value.trim()
  const fileIds = [...composeFileIds.value]
  if ((!text && fileIds.length === 0) || posting.value) return
  const ch = store.currentChannel
  if (!ch) return
  posting.value = true
  try {
    // The created post arrives back over WS (message_create, parent_id null) and is
    // prepended by onWSMessage — no optimistic insert needed (dedup by id there anyway).
    await api.createPost(ch.id, text, fileIds)
    composeText.value = ''
    composeFileIds.value = []
    clearChips()
    resetHeight()
  } catch (e) {
    store.showNotification(e.message || t('createChannel.createFailed'), 'error')
  } finally {
    posting.value = false
  }
}

// ── Load posts ────────────────────────────────────────────────
async function loadPosts() {
  const ch = store.currentChannel
  if (!ch) return
  loading.value = true
  try {
    const res = await api.getPosts(ch.id)
    posts.value = res.posts || []
    hasMore.value = !!res.has_more
  } catch (e) {
    store.showNotification(e.message || 'Failed to load posts', 'error')
  } finally {
    loading.value = false
  }
}

async function loadOlder() {
  const ch = store.currentChannel
  if (!ch || !hasMore.value || loading.value || posts.value.length === 0) return
  const before = posts.value[posts.value.length - 1].id
  loading.value = true
  try {
    const res = await api.getPosts(ch.id, 20, before)
    posts.value.push(...(res.posts || []))
    hasMore.value = !!res.has_more
  } finally {
    loading.value = false
  }
}

function onScroll() {
  if (container.value && container.value.scrollTop < 60) loadOlder()
}

// ── Likes (optimistic) ────────────────────────────────────────
async function toggleLike(post) {
  const wasLiked = post.liked_by_me
  post.liked_by_me = !wasLiked
  post.like_count = (post.like_count || 0) + (wasLiked ? -1 : 1)
  try {
    const res = wasLiked ? await api.unlikePost(post.id) : await api.likePost(post.id)
    if (res && typeof res.like_count === 'number') post.like_count = res.like_count
  } catch (e) {
    // revert on failure
    post.liked_by_me = wasLiked
    post.like_count = (post.like_count || 0) + (wasLiked ? 1 : -1)
    store.showNotification(e.message || 'Like failed', 'error')
  }
}

// ── Comments ──────────────────────────────────────────────────
async function toggleComments(post) {
  post._expanded = !post._expanded
  if (post._expanded && post._comments == null) {
    post._commentsLoading = true
    try {
      const res = await api.getComments(post.id)
      post._comments = res.messages || []
    } catch (e) {
      post._comments = []
      store.showNotification(e.message || 'Failed to load comments', 'error')
    } finally {
      post._commentsLoading = false
    }
  }
}

async function submitComment(post) {
  const text = (post._commentDraft || '').trim()
  if (!text) return
  const ch = store.currentChannel
  if (!ch) return
  post._commentDraft = ''
  try {
    // Echoes back over WS (message_create with parent_id) → appended by onWSMessage.
    await api.createComment(ch.id, post.id, text)
  } catch (e) {
    store.showNotification(e.message || 'Comment failed', 'error')
  }
}

// ── Realtime (same onMessage/offMessage bus as DMChatArea) ─────
function findPost(id) {
  return posts.value.find(p => p.id === id)
}

function onWSMessage(type, data) {
  const ch = store.currentChannel
  if (!ch) return

  if (type === 'message') {
    if (Number(data.channel_id) !== Number(ch.id)) return
    if (data.parent_id == null) {
      // New post → prepend (dedup)
      if (!posts.value.some(p => p.id === data.id)) {
        posts.value.unshift({ ...data, like_count: 0, comment_count: 0, liked_by_me: false })
      }
    } else {
      // New comment → bump count, append if that post's comments are loaded
      const post = findPost(Number(data.parent_id))
      if (post) {
        post.comment_count = (post.comment_count || 0) + 1
        if (Array.isArray(post._comments) && !post._comments.some(c => c.id === data.id)) {
          post._comments.push(data)
        }
      }
    }
  } else if (type === 'message_update') {
    // Payload is a plain Message (no enrichment fields) → Object.assign preserves
    // like_count/comment_count/_comments since they aren't present on data.
    const p = findPost(data.id)
    if (p) { Object.assign(p, data); return }
    for (const post of posts.value) {
      if (Array.isArray(post._comments)) {
        const idx = post._comments.findIndex(c => c.id === data.id)
        if (idx !== -1) { Object.assign(post._comments[idx], data); return }
      }
    }
  } else if (type === 'message_delete') {
    const id = data.message_id
    const idx = posts.value.findIndex(p => p.id === id)
    if (idx !== -1) { posts.value.splice(idx, 1); return } // post (comments cascade in DB)
    for (const post of posts.value) {
      if (Array.isArray(post._comments)) {
        const ci = post._comments.findIndex(c => c.id === id)
        if (ci !== -1) {
          post._comments.splice(ci, 1)
          post.comment_count = Math.max(0, (post.comment_count || 0) - 1)
          return
        }
      }
    }
  } else if (type === 'post_like') {
    const p = findPost(data.message_id)
    if (p && typeof data.like_count === 'number') p.like_count = data.like_count
  }
}

onMounted(() => {
  ws.onMessage(onWSMessage)
  loadPosts()
})

onUnmounted(() => {
  ws.offMessage(onWSMessage)
})

watch(() => store.currentChannel?.id, () => {
  posts.value = []
  hasMore.value = false
  loadPosts()
})
</script>

<template>
  <div class="main-content feed-area">
    <!-- Header (mirrors ChatArea) -->
    <div class="channel-header">
      <div class="channel-info">
        <button class="mobile-hamburger" @click="emit('open-sidebar')" :title="t('chatArea.channelList')">
          <i class="fas fa-bars"></i>
        </button>
        <i class="fas fa-stream"></i>
        <h3>{{ store.currentChannel?.name || t('chatArea.welcome') }}</h3>
        <span v-if="store.currentChannel?.topic" class="channel-topic">
          {{ store.currentChannel.topic }}
        </span>
      </div>
    </div>

    <!-- Scroll region: compose + posts -->
    <div class="feed-scroll" ref="container" @scroll="onScroll">
      <!-- Compose -->
      <div class="feed-compose" v-if="store.currentChannel">
        <div class="feed-compose-main">
          <div class="feed-compose-avatar">
            <img v-if="store.user?.avatar" :src="store.user.avatar" :alt="store.user?.nickname || store.user?.username" />
            <i v-else class="fas fa-user"></i>
          </div>
          <textarea
            ref="inputEl"
            v-model="composeText"
            class="feed-compose-input"
            rows="1"
            :placeholder="t('wall.composePlaceholder')"
            @input="autoGrow"
          ></textarea>
        </div>
        <div v-if="pendingChips.length" class="feed-compose-chips">
          <div v-for="chip in pendingChips" :key="chip.id" class="file-chip">
            <span class="file-chip-name">{{ chip.name }}</span>
            <span v-if="!chip.done" class="file-chip-progress">{{ chip.progress }}%</span>
            <button v-else class="file-chip-remove" @click="removeChip(chip.id, chip.fileId)">
              <i class="fas fa-times"></i>
            </button>
          </div>
        </div>
        <div class="feed-compose-actions">
          <label class="feed-attach-btn" :title="t('dm.uploadFile')">
            <i class="fas fa-image"></i>
            <input type="file" multiple style="display:none" @change="onFileInput" />
          </label>
          <button
            class="btn-primary btn-sm feed-post-btn"
            :disabled="posting || (!composeText.trim() && composeFileIds.length === 0)"
            @click="submitPost"
          >
            {{ posting ? t('wall.posting') : t('wall.post') }}
          </button>
        </div>
      </div>

      <!-- Empty -->
      <div v-if="!loading && posts.length === 0" class="feed-empty">
        <i class="fas fa-stream"></i>
        <p>{{ t('wall.empty') }}</p>
      </div>

      <!-- Posts -->
      <div v-for="post in posts" :key="post.id" class="feed-post">
        <MessageItem :message="post" />

        <!-- Actions -->
        <div class="feed-post-actions">
          <button class="feed-action" :class="{ liked: post.liked_by_me }" @click="toggleLike(post)">
            <i :class="['fas', 'fa-heart']"></i>
            <span>{{ t('wall.like') }}</span>
            <span v-if="post.like_count" class="feed-count">{{ post.like_count }}</span>
          </button>
          <button class="feed-action" :class="{ active: post._expanded }" @click="toggleComments(post)">
            <i class="fas fa-comment"></i>
            <span>{{ t('wall.comment') }}</span>
            <span v-if="post.comment_count" class="feed-count">{{ post.comment_count }}</span>
          </button>
        </div>

        <!-- Comments -->
        <div v-if="post._expanded" class="feed-comments">
          <div v-if="post._commentsLoading" class="feed-comments-loading">
            {{ t('wall.loadingComments') }}
          </div>
          <MessageItem
            v-for="c in (post._comments || [])"
            :key="c.id"
            :message="c"
          />
          <div class="feed-comment-box">
            <textarea
              v-model="post._commentDraft"
              class="feed-comment-input"
              rows="1"
              :placeholder="t('wall.commentPlaceholder')"
              @keydown.enter.exact.prevent="submitComment(post)"
            ></textarea>
            <button class="btn-primary btn-sm" @click="submitComment(post)">{{ t('wall.reply') }}</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed-area {
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  min-width: 0;
}

.feed-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feed-compose {
  background: var(--bg-secondary, #2f3136);
  border: 1px solid var(--border-color, #40444b);
  border-radius: var(--radius, 8px);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feed-compose-main {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.feed-compose-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  flex-shrink: 0;
  overflow: hidden;
  background: var(--bg-input, #40444b);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted, #8e9297);
}
.feed-compose-avatar img { width: 100%; height: 100%; object-fit: cover; }

/* Compose input: Twitter-style — text sits on the card, roomy, auto-growing */
.feed-compose-input {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: none;
  color: var(--text-primary, #dcddde);
  font-size: 17px;
  line-height: 1.55;
  padding: 8px 0;
  resize: none;
  outline: none;
  font-family: inherit;
  min-height: 52px;
  max-height: 240px;
  overflow-y: auto;
}

/* Comment input keeps the compact filled-box look */
.feed-comment-input {
  width: 100%;
  background: var(--bg-input, #40444b);
  border: none;
  border-radius: var(--radius, 8px);
  color: var(--text-primary, #dcddde);
  font-size: 14px;
  padding: 10px 12px;
  resize: none;
  outline: none;
  font-family: inherit;
  line-height: 1.5;
}

.feed-compose-input::placeholder,
.feed-comment-input::placeholder {
  color: var(--text-muted, #8e9297);
}

.feed-compose-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: flex-end;
  padding-left: 56px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color, #40444b);
}

.feed-post-btn {
  width: auto;
  min-width: 96px;
  padding: 8px 22px;
}

.feed-attach-btn {
  color: var(--text-muted, #8e9297);
  cursor: pointer;
  font-size: 18px;
  margin-right: auto;
  display: flex;
  align-items: center;
  transition: color 0.1s;
}

.feed-attach-btn:hover { color: var(--text-primary, #dcddde); }

.feed-compose-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.file-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-input, #40444b);
  border-radius: var(--radius, 8px);
  padding: 4px 10px;
  font-size: 12px;
  color: var(--text-normal, #dcddde);
}

.file-chip-name { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-chip-progress { color: var(--text-muted, #8e9297); }
.file-chip-remove { background: none; border: none; color: var(--text-muted, #8e9297); cursor: pointer; padding: 0; line-height: 1; }
.file-chip-remove:hover { color: var(--text-primary, #dcddde); }

.feed-post {
  background: var(--bg-secondary, #2f3136);
  border: 1px solid var(--border-color, #40444b);
  border-radius: var(--radius, 8px);
  padding: 8px 12px 4px;
}

.feed-post-actions {
  display: flex;
  gap: 8px;
  padding: 4px 0 8px 52px;
  border-top: 1px solid var(--border-color, #40444b);
  margin-top: 4px;
}

.feed-action {
  background: none;
  border: none;
  color: var(--text-muted, #8e9297);
  cursor: pointer;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: var(--radius, 8px);
  transition: color 0.1s, background 0.1s;
}

.feed-action:hover { background: var(--bg-hover, #4f545c); color: var(--text-primary, #dcddde); }
.feed-action.liked { color: var(--danger, #ed4245); }
.feed-action.active { color: var(--accent, #5865f2); }

.feed-count {
  background: var(--bg-input, #40444b);
  border-radius: 10px;
  padding: 0 6px;
  font-size: 11px;
  min-width: 16px;
  text-align: center;
}

.feed-comments {
  padding: 4px 0 8px 20px;
  border-top: 1px solid var(--border-color, #40444b);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.feed-comments-loading {
  color: var(--text-muted, #8e9297);
  font-size: 12px;
  padding: 6px 0;
}

.feed-comment-box {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  padding: 8px 0 0 32px;
}

.feed-comment-input { flex: 1; }

.feed-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted, #8e9297);
  gap: 12px;
  padding: 48px 16px;
}

.feed-empty i { font-size: 42px; opacity: 0.4; }
</style>
