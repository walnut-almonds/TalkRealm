<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const props = defineProps({ messageId: { type: Number, required: true } })

const store = useAppStore()
const { t } = useI18n()

const state = ref('loading') // 'loading' | 'resolved' | 'error'
const data = ref(null)       // { message, channel }

const author = computed(() => data.value?.message?.author || {})
const authorName = computed(() => author.value.nickname || author.value.username || t('chat.unknownUser'))
const avatar = computed(() => author.value.avatar || null)
const channelName = computed(() => data.value?.channel?.name || '')
const snippet = computed(() => (data.value?.message?.content || '').trim())

onMounted(async () => {
  try {
    data.value = await api.resolvePermalink(props.messageId)
    state.value = 'resolved'
  } catch {
    state.value = 'error'
  }
})

function onClick() {
  if (state.value !== 'resolved') return
  // ponytail: store.jumpToPermalink arrives in Task 6 — optional-call until then
  store.jumpToPermalink?.(props.messageId)
}
</script>

<template>
  <div v-if="state === 'loading'" class="permalink-card permalink-card--loading">
    <i class="fas fa-link permalink-icon"></i>
    <div class="permalink-skeleton"></div>
  </div>

  <div
    v-else-if="state === 'error'"
    class="permalink-card permalink-card--disabled"
    :title="t('chat.permalinkUnavailable')"
  >
    <i class="fas fa-link permalink-icon"></i>
    <span class="permalink-unavailable">{{ t('chat.permalinkUnavailable') }}</span>
  </div>

  <div v-else class="permalink-card permalink-card--clickable" @click="onClick">
    <i class="fas fa-link permalink-icon"></i>
    <div class="permalink-avatar">
      <img v-if="avatar" :src="avatar" :alt="authorName" />
      <i v-else class="fas fa-user"></i>
    </div>
    <div class="permalink-body">
      <div class="permalink-head">
        <span class="permalink-author">{{ authorName }}</span>
        <span class="permalink-sep">·</span>
        <span class="permalink-channel">#{{ channelName }}</span>
      </div>
      <div class="permalink-snippet">{{ snippet }}</div>
    </div>
  </div>
</template>

<style scoped>
.permalink-card {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 8px 10px;
  background: var(--bg-secondary, #2f3136);
  border: 1px solid var(--border-color, #40444b);
  border-left: 3px solid var(--accent, #5865f2);
  border-radius: var(--radius, 8px);
  max-width: 420px;
}

.permalink-icon { color: var(--text-muted, #8e9297); font-size: 12px; flex-shrink: 0; }

.permalink-card--clickable { cursor: pointer; transition: background 0.1s; }
.permalink-card--clickable:hover { background: var(--bg-hover, #4f545c); }

.permalink-card--disabled { opacity: 0.6; cursor: not-allowed; }
.permalink-unavailable { color: var(--text-muted, #8e9297); font-size: 13px; }

.permalink-skeleton {
  flex: 1;
  height: 14px;
  border-radius: 4px;
  background: var(--bg-hover, #40444b);
  animation: permalinkPulse 1.2s ease-in-out infinite;
}
@keyframes permalinkPulse { 0%, 100% { opacity: 0.5; } 50% { opacity: 1; } }

.permalink-avatar {
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
.permalink-avatar img { width: 100%; height: 100%; object-fit: cover; }

.permalink-body { min-width: 0; flex: 1; }
.permalink-head { display: flex; align-items: baseline; gap: 6px; }
.permalink-author { font-weight: 600; font-size: 13px; color: var(--text-primary, #dcddde); }
.permalink-sep { color: var(--text-muted, #8e9297); font-size: 12px; }
.permalink-channel { font-size: 12px; color: var(--accent, #5865f2); }
.permalink-snippet {
  font-size: 13px;
  color: var(--text-muted, #8e9297);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
