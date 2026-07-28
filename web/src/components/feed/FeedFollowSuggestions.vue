<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['open-profile'])
const store = useAppStore()
const { t } = useI18n()

const users = ref([])
const loading = ref(true)
const pending = ref({}) // userId -> true while the follow request is in flight

async function load() {
  loading.value = true
  try {
    const res = await api.getFollowSuggestions()
    users.value = res || []
  } catch (e) {
    store.showNotification(e.message || 'Failed to load suggestions', 'error')
  } finally {
    loading.value = false
  }
}

async function doFollow(u) {
  pending.value = { ...pending.value, [u.id]: true }
  try {
    await api.follow(u.id)
    users.value = users.value.filter(x => x.id !== u.id) // followed → drop from suggestions
  } catch (e) {
    store.showNotification(e.message || 'Follow failed', 'error')
  } finally {
    const next = { ...pending.value }
    delete next[u.id]
    pending.value = next
  }
}

function name(u) { return u.nickname || u.username }

onMounted(load)
</script>

<template>
  <div class="feed-suggestions">
    <h3 class="feed-suggestions-title">{{ t('feed.suggestionsTitle') }}</h3>
    <div v-if="loading" class="feed-suggestions-empty">…</div>
    <div v-else-if="users.length === 0" class="feed-suggestions-empty">{{ t('feed.emptySuggestions') }}</div>
    <div v-for="u in users" :key="u.id" class="feed-suggest-row">
      <div class="message-avatar feed-suggest-avatar" @click="emit('open-profile', u.id)">
        <img v-if="u.avatar" :src="u.avatar" :alt="name(u)" />
        <i v-else class="fas fa-user"></i>
      </div>
      <span class="feed-suggest-name" @click="emit('open-profile', u.id)">{{ name(u) }}</span>
      <button class="btn-primary btn-sm" :disabled="pending[u.id]" @click="doFollow(u)">
        {{ t('feed.follow') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.feed-suggestions-title {
  font-size: 12px;
  color: var(--text-muted, #8e9297);
  margin: 0 0 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.feed-suggestions-empty { color: var(--text-muted, #8e9297); font-size: 13px; }
.feed-suggest-row { display: flex; align-items: center; gap: 10px; padding: 6px 0; }
.feed-suggest-avatar { width: 36px; height: 36px; cursor: pointer; flex-shrink: 0; }
.feed-suggest-name {
  flex: 1;
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary, #dcddde);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.feed-suggest-name:hover { text-decoration: underline; }
</style>
