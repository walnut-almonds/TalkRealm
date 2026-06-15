<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import MessageItem from './MessageItem.vue'

const store = useAppStore()
const voiceStore = useVoiceStore()
const { t } = useI18n()
const container = ref(null)
const isAtBottom = ref(true)
const hasMore = ref(true)

const firstUnreadIndex = computed(() => {
  if (!store.currentChannel || store.currentChannel.type !== 'text') return -1
  const unread = Number(store.currentChannelUnreadCount || 0)
  if (!unread || unread < 1 || !store.messages.length) return -1

  let seenUnread = 0
  for (let i = store.messages.length - 1; i >= 0; i--) {
    const msg = store.messages[i]
    if (!msg || msg.user_id === store.user?.id) continue
    seenUnread += 1
    if (seenUnread >= unread) return i
  }

  // If older unread messages are not loaded yet, show the marker at top.
  return 0
})

// Auto-scroll when new messages arrive if user is at bottom
watch(() => store.messages.length, async () => {
  if (isAtBottom.value) {
    await nextTick()
    scrollToBottom()
  }
})

// 也監聴最後一筆訊息的 id 變化（pending 被真實 echo 取代時 length 不變，但需要 scroll）
watch(() => store.messages.at?.(-1)?.id, async (newId) => {
  if (newId && isAtBottom.value) {
    await nextTick()
    scrollToBottom()
  }
})

watch(() => store.currentChannel, async () => {
  hasMore.value = true
  await nextTick()
  scrollToBottom()
})

function scrollToBottom() {
  if (container.value) container.value.scrollTop = container.value.scrollHeight
}

function onScroll() {
  if (!container.value) return
  const { scrollTop, scrollHeight, clientHeight } = container.value
  isAtBottom.value = scrollHeight - scrollTop - clientHeight < 60
}

async function loadMore() {
  if (!store.currentChannel || store.messages.length === 0) return
  const firstId = store.messages[0]?.id
  const msgs = await store.loadMessages(store.currentChannel.id, firstId)
  if (msgs.length < 50) hasMore.value = false
  // Restore scroll position after prepend
  await nextTick()
  if (container.value) {
    const newScrollHeight = container.value.scrollHeight
    container.value.scrollTop = newScrollHeight - container.value.clientHeight
  }
}

// Check if consecutive messages are from the same user (for grouping)
function isGrouped(idx) {
  if (idx === 0) return false
  const cur = store.messages[idx]
  const prev = store.messages[idx - 1]
  return prev.user_id === cur.user_id
}
</script>

<template>
  <div class="messages-container" ref="container" @scroll="onScroll">
    <!-- Welcome / Empty state -->
    <div v-if="!store.currentChannel" class="welcome-message">
      <h1>{{ t('messageList.welcomeTitle') }}</h1>
      <p>{{ t('messageList.welcomeSubtitle') }}</p>
    </div>

    <template v-else>
      <!-- Load more -->
      <button v-if="hasMore && store.messages.length >= 50" class="load-more-btn" @click="loadMore">
        {{ t('messageList.loadMore') }}
      </button>

      <!-- Channel start indicator -->
      <div v-if="!hasMore || store.messages.length < 50" class="welcome-message" style="flex:0;padding:24px 16px">
        <h1 style="font-size:18px"># {{ store.currentChannel.name }}</h1>
        <p>{{ store.currentChannel.topic || t('messageList.channelStart') }}</p>
      </div>

      <!-- Messages -->
      <template v-for="(msg, idx) in store.messages" :key="msg.id || msg.nonce">
        <div v-if="idx === firstUnreadIndex" class="unread-divider">
          <span>{{ t('messageList.unreadDivider') }}</span>
        </div>
        <MessageItem
          :message="msg"
          :grouped="isGrouped(idx)"
          :is-speaking="voiceStore.isSpeaking(msg.user_id)"
        />
      </template>
    </template>
  </div>
</template>
