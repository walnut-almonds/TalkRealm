<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'
import FeedComposer from '@/components/feed/FeedComposer.vue'
import FeedPostCard from '@/components/feed/FeedPostCard.vue'
import FeedProfile from '@/components/feed/FeedProfile.vue'
import FeedFollowSuggestions from '@/components/feed/FeedFollowSuggestions.vue'

const store = useAppStore()
const { t } = useI18n()

const posts = ref([])            // newest-first
const hasMore = ref(false)
const loading = ref(false)
const container = ref(null)
const tab = ref('discover')      // 'discover' (offset, default) | 'following' (cursor)
const activeProfileUserId = ref(null)  // when set, show that user's profile instead of the timeline

function openProfile(userId) {
  if (userId) activeProfileUserId.value = userId
}

async function load() {
  loading.value = true
  try {
    const res = tab.value === 'discover' ? await api.getDiscover(0, 20) : await api.getTimeline()
    posts.value = res.posts || []
    hasMore.value = !!res.has_more
  } catch (e) {
    store.showNotification(e.message || 'Failed to load feed', 'error')
  } finally {
    loading.value = false
  }
}

async function loadOlder() {
  if (!hasMore.value || loading.value || posts.value.length === 0) return
  loading.value = true
  try {
    const res = tab.value === 'discover'
      ? await api.getDiscover(posts.value.length, 20)
      : await api.getTimeline(20, posts.value[posts.value.length - 1].id)
    posts.value.push(...(res.posts || []))
    hasMore.value = !!res.has_more
  } catch (e) {
    store.showNotification(e.message || 'Failed to load feed', 'error')
  } finally {
    loading.value = false
  }
}

function switchTab(next) {
  if (tab.value === next) return
  tab.value = next
  posts.value = []
  hasMore.value = false
  load()
}

// Newest-first timeline grows downward: load older when scrolled near the bottom.
function onScroll() {
  const el = container.value
  if (!el) return
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 200) loadOlder()
}

function onPosted(post) {
  if (post) posts.value.unshift(post)
}

function onDeleted(id) {
  const i = posts.value.findIndex(p => p.id === id)
  if (i !== -1) posts.value.splice(i, 1)
}

onMounted(load)
</script>

<template>
  <div class="feed-view">
    <FeedProfile
      v-if="activeProfileUserId"
      :userId="activeProfileUserId"
      @close="activeProfileUserId = null"
      @open-profile="openProfile"
    />
    <div v-else class="feed-main" ref="container" @scroll="onScroll">
      <div class="feed-column">
        <div class="feed-tabs">
          <button class="feed-tab" :class="{ active: tab === 'discover' }" @click="switchTab('discover')">{{ t('feed.tabForYou') }}</button>
          <button class="feed-tab" :class="{ active: tab === 'following' }" @click="switchTab('following')">{{ t('feed.tabFollowing') }}</button>
        </div>

        <FeedComposer @posted="onPosted" />

        <div v-if="!loading && posts.length === 0" class="feed-empty">
          <i class="fas fa-rss"></i>
          <p>{{ t('feed.empty') }}</p>
        </div>

        <FeedPostCard
          v-for="post in posts"
          :key="post.id"
          :post="post"
          @deleted="onDeleted"
          @author-click="openProfile"
        />

        <div v-if="loading && posts.length" class="feed-loading-more">…</div>
      </div>
    </div>

    <!-- Wide-screen right column: follow suggestions -->
    <aside class="feed-aside">
      <FeedFollowSuggestions @open-profile="openProfile" />
    </aside>
  </div>
</template>

<style scoped>
.feed-view {
  display: flex;
  flex: 1;
  height: 100%;
  min-width: 0;
}

.feed-main {
  flex: 1;
  overflow-y: auto;
  min-width: 0;
  display: flex;
  justify-content: center;
  padding: 16px;
}

.feed-column {
  width: 100%;
  max-width: 640px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feed-aside {
  width: 300px;
  flex-shrink: 0;
  border-left: 1px solid var(--border-color, #40444b);
  padding: 16px;
  overflow-y: auto;
}

.feed-tabs {
  display: flex;
  gap: 8px;
  border-bottom: 1px solid var(--border-color, #40444b);
}

.feed-tab {
  background: none;
  border: none;
  padding: 10px 14px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  color: var(--text-muted, #8e9297);
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.feed-tab:hover { color: var(--text-color, #dcddde); }

.feed-tab.active {
  color: var(--accent, #5865f2);
  border-bottom-color: var(--accent, #5865f2);
}

.feed-loading-more {
  text-align: center;
  color: var(--text-muted, #8e9297);
  padding: 8px;
}

.feed-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted, #8e9297);
  gap: 12px;
  padding: 48px 16px;
}

.feed-empty i { font-size: 42px; opacity: 0.4; }

@media (max-width: 900px) {
  .feed-aside { display: none; }
}
</style>
