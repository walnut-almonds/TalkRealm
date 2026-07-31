<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { renderMarkdown } from '@/utils/markdown.js'
import { formatTimestamp, formatFullTimestamp, formatFileSize, IMAGE_TYPES } from '@/utils/format.js'
import { api } from '@/api/index.js'

const props = defineProps({ post: { type: Object, required: true } })
const emit = defineEmits(['deleted', 'author-click'])

const store = useAppStore()
const { t } = useI18n()

const author = computed(() => props.post.author || {})
const nickname = computed(() => author.value.nickname || author.value.username || t('chat.unknownUser'))
const avatar = computed(() => author.value.avatar || null)
const timestamp = computed(() => formatTimestamp(props.post.created_at))
const fullTimestamp = computed(() => formatFullTimestamp(props.post.created_at))
const bodyHtml = computed(() => renderMarkdown(props.post.content).html)
const isOwn = computed(() => store.user?.id && store.user.id === (props.post.author_id ?? author.value.id))

// ── Attachments (mirror MessageItem image loading + lightbox) ──
async function loadImage(fileId, imgEl) {
  const url = await store.getImageUrl(fileId)
  if (url && imgEl) {
    imgEl.src = url
    imgEl.onload = () => { imgEl.style.opacity = '1' }
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
}

// ── Like (optimistic via liked_by_me) ──
async function toggleLike() {
  const p = props.post
  const wasLiked = p.liked_by_me
  p.liked_by_me = !wasLiked
  p.like_count = (p.like_count || 0) + (wasLiked ? -1 : 1)
  try {
    const res = wasLiked ? await api.unlikeFeedPost(p.id) : await api.likeFeedPost(p.id)
    if (res && typeof res.like_count === 'number') p.like_count = res.like_count
  } catch (e) {
    p.liked_by_me = wasLiked
    p.like_count = (p.like_count || 0) + (wasLiked ? 1 : -1)
    store.showNotification(e.message || 'Like failed', 'error')
  }
}

// ── Comments (lazy) ──
const expanded = ref(false)
const comments = ref(null)
const commentsLoading = ref(false)
const commentDraft = ref('')

async function toggleComments() {
  expanded.value = !expanded.value
  if (expanded.value && comments.value == null) {
    commentsLoading.value = true
    try {
      const res = await api.getFeedComments(props.post.id)
      comments.value = res.comments || []
    } catch (e) {
      comments.value = []
      store.showNotification(e.message || 'Failed to load comments', 'error')
    } finally {
      commentsLoading.value = false
    }
  }
}

async function submitComment() {
  const text2 = commentDraft.value.trim()
  if (!text2) return
  commentDraft.value = ''
  try {
    const c = await api.addFeedComment(props.post.id, text2)
    if (!Array.isArray(comments.value)) comments.value = []
    comments.value.push(c)
    props.post.comment_count = (props.post.comment_count || 0) + 1
  } catch (e) {
    store.showNotification(e.message || 'Comment failed', 'error')
  }
}

async function toggleCommentLike(c) {
  const wasLiked = c.liked_by_me
  c.liked_by_me = !wasLiked
  c.like_count = (c.like_count || 0) + (wasLiked ? -1 : 1)
  try {
    const res = wasLiked ? await api.unlikeFeedComment(c.id) : await api.likeFeedComment(c.id)
    if (res && typeof res.like_count === 'number') c.like_count = res.like_count
  } catch (e) {
    c.liked_by_me = wasLiked
    c.like_count = (c.like_count || 0) + (wasLiked ? 1 : -1)
    store.showNotification(e.message || 'Like failed', 'error')
  }
}

function commentAuthorName(c) {
  const a = c.author || {}
  return a.nickname || a.username || t('chat.unknownUser')
}

// ── Own-post edit / delete ──
const isEditing = ref(false)
const editContent = ref('')

function startEdit() {
  editContent.value = props.post.content
  isEditing.value = true
}
function cancelEdit() { isEditing.value = false; editContent.value = '' }

async function saveEdit() {
  const text2 = editContent.value.trim()
  if (!text2 || text2 === props.post.content) { cancelEdit(); return }
  try {
    const updated = await api.updateFeedPost(props.post.id, text2)
    Object.assign(props.post, updated || {}, { content: text2, is_edited: true })
    cancelEdit()
  } catch (e) {
    store.showNotification(e.message || 'Edit failed', 'error')
  }
}

async function remove() {
  if (!confirm(t('feed.deleteConfirm'))) return
  try {
    await api.deleteFeedPost(props.post.id)
    emit('deleted', props.post.id)
  } catch (e) {
    store.showNotification(e.message || 'Delete failed', 'error')
  }
}
</script>

<template>
  <div class="feed-post">
    <div class="message message-group-start">
      <!-- Avatar -->
      <div class="message-avatar feed-clickable" @click="emit('author-click', author.id)">
        <img v-if="avatar" :src="avatar" :alt="nickname" />
        <i v-else class="fas fa-user"></i>
      </div>

      <div class="message-content">
        <div class="message-header">
          <span class="message-author feed-clickable" @click="emit('author-click', author.id)">{{ nickname }}</span>
          <span class="message-timestamp" :title="fullTimestamp">{{ timestamp }}</span>
          <span v-if="post.is_edited" class="message-edited">({{ t('feed.edited') }})</span>
        </div>

        <template v-if="!isEditing">
          <div class="message-text" v-html="bodyHtml"></div>
        </template>
        <div v-else class="message-edit-box">
          <textarea
            v-model="editContent"
            class="message-edit-input"
            rows="3"
            @keydown.enter.exact.prevent="saveEdit"
            @keydown.esc="cancelEdit"
          ></textarea>
          <div class="message-edit-hints">
            <div style="display:flex;gap:6px">
              <button class="btn-secondary btn-sm" @click="cancelEdit">{{ t('common.cancel') }}</button>
              <button class="btn-primary btn-sm" @click="saveEdit">{{ t('common.save') }}</button>
            </div>
          </div>
        </div>

        <!-- Attachments -->
        <div v-if="post.attachments?.length" class="message-attachments">
          <template v-for="att in post.attachments" :key="att.id">
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
      </div>

      <!-- Own-post actions -->
      <div v-if="isOwn && !isEditing" class="message-actions-bar">
        <button class="msg-action-btn" :title="t('feed.edit')" @click="startEdit">
          <i class="fas fa-pencil"></i>
        </button>
        <button class="msg-action-btn danger" :title="t('feed.delete')" @click="remove">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    </div>

    <!-- Like / comment actions -->
    <div class="feed-post-actions">
      <button class="feed-action" :class="{ liked: post.liked_by_me }" @click="toggleLike">
        <i class="fas fa-heart"></i>
        <span>{{ t('feed.like') }}</span>
        <span v-if="post.like_count" class="feed-count">{{ post.like_count }}</span>
      </button>
      <button class="feed-action" :class="{ active: expanded }" @click="toggleComments">
        <i class="fas fa-comment"></i>
        <span>{{ t('feed.comment') }}</span>
        <span v-if="post.comment_count" class="feed-count">{{ post.comment_count }}</span>
      </button>
    </div>

    <!-- Comments -->
    <div v-if="expanded" class="feed-comments">
      <div v-if="commentsLoading" class="feed-comments-loading">…</div>
      <div v-else-if="comments && comments.length === 0" class="feed-comments-loading">
        {{ t('feed.emptyComments') }}
      </div>
      <div v-for="c in (comments || [])" :key="c.id" class="feed-comment">
        <div class="feed-comment-avatar">
          <img v-if="c.author?.avatar" :src="c.author.avatar" :alt="commentAuthorName(c)" />
          <i v-else class="fas fa-user"></i>
        </div>
        <div class="feed-comment-body">
          <div class="feed-comment-head">
            <span class="feed-comment-author">{{ commentAuthorName(c) }}</span>
            <span class="feed-comment-time">{{ formatTimestamp(c.created_at) }}</span>
          </div>
          <div class="feed-comment-text">{{ c.content }}</div>
        </div>
        <button
          class="feed-comment-like"
          :class="{ liked: c.liked_by_me }"
          :title="t('feed.like')"
          @click="toggleCommentLike(c)"
        >
          <i class="fas fa-heart"></i>
          <span v-if="c.like_count" class="feed-comment-like-count">{{ c.like_count }}</span>
        </button>
      </div>
      <div class="feed-comment-box">
        <textarea
          v-model="commentDraft"
          class="feed-compose-input feed-comment-input"
          rows="1"
          :placeholder="t('feed.commentPlaceholder')"
          @keydown.enter.exact.prevent="submitComment"
        ></textarea>
        <button class="btn-primary btn-sm" @click="submitComment">{{ t('feed.addComment') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed-post {
  background: var(--bg-secondary, #2f3136);
  border: 1px solid var(--border-color, #40444b);
  border-radius: var(--radius, 8px);
  padding: 8px 12px 4px;
}

.feed-clickable { cursor: pointer; }

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
  gap: 8px;
}

.feed-comments-loading {
  color: var(--text-muted, #8e9297);
  font-size: 12px;
  padding: 6px 0;
}

.feed-comment { display: flex; gap: 8px; align-items: flex-start; }

.feed-comment-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--bg-input, #40444b);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted, #8e9297);
}
.feed-comment-avatar img { width: 100%; height: 100%; object-fit: cover; }

.feed-comment-head { display: flex; align-items: baseline; gap: 8px; }
.feed-comment-author { font-weight: 600; font-size: 13px; color: var(--text-primary, #dcddde); }
.feed-comment-time { font-size: 11px; color: var(--text-muted, #8e9297); }
.feed-comment-text { font-size: 14px; color: var(--text-normal, #dcddde); white-space: pre-wrap; word-break: break-word; }

.feed-comment-like {
  background: none;
  border: none;
  color: var(--text-muted, #8e9297);
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px;
  flex-shrink: 0;
  transition: color 0.1s;
}
.feed-comment-like:hover { color: var(--text-primary, #dcddde); }
.feed-comment-like.liked { color: var(--danger, #ed4245); }
.feed-comment-like-count { font-size: 11px; }

.feed-comment-box {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  padding-top: 4px;
}

.feed-compose-input {
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
.feed-comment-input { flex: 1; }
.feed-compose-input::placeholder { color: var(--text-muted, #8e9297); }
</style>
