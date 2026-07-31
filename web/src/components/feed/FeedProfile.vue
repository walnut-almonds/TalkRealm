<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'
import FeedPostCard from '@/components/feed/FeedPostCard.vue'

const props = defineProps({ userId: { type: Number, required: true } })
const emit = defineEmits(['close', 'open-profile'])
const store = useAppStore()
const { t } = useI18n()

const user = ref(null)
const posts = ref([])
const hasMore = ref(false)
const loading = ref(false)
const followingCount = ref(0)
const followersCount = ref(0)
const followersList = ref([])
const followingList = ref([])
const listView = ref(null) // null = posts | 'followers' | 'following'
const isFollowing = ref(false)
const busy = ref(false)

const shownList = computed(() => (listView.value === 'followers' ? followersList.value : followingList.value))

function personName(u) { return u.nickname || u.username }

const isSelf = computed(() => store.user?.id === props.userId)
const name = computed(() => (user.value ? user.value.nickname || user.value.username : ''))

async function load() {
  loading.value = true
  try {
    const [pub, timeline, followers, following] = await Promise.all([
      api.getPublicUser(props.userId).catch(() => null),
      api.getUserPosts(props.userId).catch(() => ({ posts: [], has_more: false })),
      api.getFollowers(props.userId).catch(() => ({ users: [], count: 0 })),
      api.getFollowing(props.userId).catch(() => ({ users: [], count: 0 })),
    ])
    user.value = pub
    posts.value = timeline.posts || []
    hasMore.value = !!timeline.has_more
    followersList.value = followers.users || []
    followingList.value = following.users || []
    followersCount.value = followers.count || 0
    followingCount.value = following.count || 0
    listView.value = null
    // I follow this user iff I'm among their followers
    isFollowing.value = !!followersList.value.find(u => u.id === store.user?.id)
  } catch (e) {
    store.showNotification(e.message || 'Failed to load profile', 'error')
  } finally {
    loading.value = false
  }
}

async function loadOlder() {
  if (!hasMore.value || loading.value || posts.value.length === 0) return
  const before = posts.value[posts.value.length - 1].id
  loading.value = true
  try {
    const res = await api.getUserPosts(props.userId, 20, before)
    posts.value.push(...(res.posts || []))
    hasMore.value = !!res.has_more
  } finally {
    loading.value = false
  }
}

function onScroll(e) {
  const el = e.target
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 200) loadOlder()
}

async function toggleFollow() {
  busy.value = true
  const was = isFollowing.value
  try {
    if (was) {
      await api.unfollow(props.userId)
      followersCount.value = Math.max(0, followersCount.value - 1)
    } else {
      await api.follow(props.userId)
      followersCount.value += 1
    }
    isFollowing.value = !was
  } catch (e) {
    store.showNotification(e.message || 'Action failed', 'error')
  } finally {
    busy.value = false
  }
}

function onDeleted(id) {
  const i = posts.value.findIndex(p => p.id === id)
  if (i !== -1) posts.value.splice(i, 1)
}

watch(() => props.userId, load)
onMounted(load)
</script>

<template>
  <div class="feed-profile" @scroll="onScroll">
    <div class="feed-profile-header">
      <button class="feed-back" :title="t('common.cancel')" @click="emit('close')">
        <i class="fas fa-arrow-left"></i>
      </button>
      <div class="message-avatar feed-profile-avatar">
        <img v-if="user?.avatar" :src="user.avatar" :alt="name" />
        <i v-else class="fas fa-user"></i>
      </div>
      <div class="feed-profile-meta">
        <h2 class="feed-profile-name">{{ name }}</h2>
        <div class="feed-profile-stats">
          <button class="feed-stat" :class="{ active: listView === 'followers' }" @click="listView = listView === 'followers' ? null : 'followers'">
            <strong>{{ followersCount }}</strong> {{ t('feed.followers') }}
          </button>
          <button class="feed-stat" :class="{ active: listView === 'following' }" @click="listView = listView === 'following' ? null : 'following'">
            <strong>{{ followingCount }}</strong> {{ t('feed.following') }}
          </button>
        </div>
      </div>
      <button
        v-if="!isSelf"
        class="btn-primary btn-sm feed-follow-btn"
        :class="{ following: isFollowing }"
        :disabled="busy"
        @click="toggleFollow"
      >
        {{ isFollowing ? t('feed.unfollow') : t('feed.follow') }}
      </button>
    </div>

    <!-- Followers / following list -->
    <div v-if="listView" class="feed-profile-posts">
      <div v-if="shownList.length === 0" class="feed-empty">
        <i class="fas fa-users"></i>
        <p>{{ listView === 'followers' ? t('feed.followers') : t('feed.following') }}</p>
      </div>
      <div
        v-for="u in shownList"
        :key="u.id"
        class="feed-userlist-row"
        @click="emit('open-profile', u.id)"
      >
        <div class="message-avatar feed-userlist-avatar">
          <img v-if="u.avatar" :src="u.avatar" :alt="personName(u)" />
          <i v-else class="fas fa-user"></i>
        </div>
        <span class="feed-userlist-name">{{ personName(u) }}</span>
      </div>
    </div>

    <!-- Posts -->
    <div v-else class="feed-profile-posts">
      <div v-if="!loading && posts.length === 0" class="feed-empty">
        <i class="fas fa-rss"></i>
        <p>{{ t('feed.empty') }}</p>
      </div>
      <FeedPostCard
        v-for="post in posts"
        :key="post.id"
        :post="post"
        @deleted="onDeleted"
        @author-click="emit('open-profile', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.feed-profile { flex: 1; overflow-y: auto; padding: 16px; min-width: 0; }
.feed-profile-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color, #40444b);
  margin: 0 auto 16px;
  max-width: 640px;
}
.feed-back { background: none; border: none; color: var(--text-muted, #8e9297); cursor: pointer; font-size: 18px; padding: 4px 8px; }
.feed-back:hover { color: var(--text-primary, #dcddde); }
.feed-profile-avatar { width: 56px; height: 56px; flex-shrink: 0; }
.feed-profile-meta { flex: 1; min-width: 0; }
.feed-profile-name { font-size: 20px; margin: 0; color: var(--text-primary, #dcddde); }
.feed-profile-stats { display: flex; gap: 16px; margin-top: 4px; }
.feed-stat {
  background: none;
  border: none;
  padding: 2px 0;
  font-size: 13px;
  color: var(--text-muted, #8e9297);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: color 0.1s, border-color 0.1s;
}
.feed-stat:hover { color: var(--text-primary, #dcddde); }
.feed-stat.active { color: var(--text-primary, #dcddde); border-bottom-color: var(--accent, #5865f2); }
.feed-stat strong { color: var(--text-primary, #dcddde); }

.feed-userlist-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  border-radius: var(--radius, 8px);
  cursor: pointer;
  transition: background 0.1s;
}
.feed-userlist-row:hover { background: var(--bg-hover, #4f545c); }
.feed-userlist-avatar { width: 40px; height: 40px; flex-shrink: 0; }
.feed-userlist-name { font-weight: 600; font-size: 15px; color: var(--text-primary, #dcddde); }
.feed-follow-btn.following { background: var(--bg-input, #40444b); }
.feed-profile-posts { max-width: 640px; margin: 0 auto; display: flex; flex-direction: column; gap: 12px; }
.feed-empty { display: flex; flex-direction: column; align-items: center; color: var(--text-muted, #8e9297); gap: 12px; padding: 48px 16px; }
.feed-empty i { font-size: 42px; opacity: 0.4; }
</style>
