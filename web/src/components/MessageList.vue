<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import MessageItem from './MessageItem.vue'

const store = useAppStore()
const voiceStore = useVoiceStore()
const container = ref(null)
const isAtBottom = ref(true)
const hasMore = ref(true)

// Auto-scroll when new messages arrive if user is at bottom
watch(() => store.messages.length, async () => {
  if (isAtBottom.value) {
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
      <h1>歡迎來到 TalkRealm！</h1>
      <p>選擇一個頻道開始聊天，或建立一個新的社群。</p>
    </div>

    <template v-else>
      <!-- Load more -->
      <button v-if="hasMore && store.messages.length >= 50" class="load-more-btn" @click="loadMore">
        載入更多訊息
      </button>

      <!-- Channel start indicator -->
      <div v-if="!hasMore || store.messages.length < 50" class="welcome-message" style="flex:0;padding:24px 16px">
        <h1 style="font-size:18px"># {{ store.currentChannel.name }}</h1>
        <p>{{ store.currentChannel.topic || '這是頻道的開始。' }}</p>
      </div>

      <!-- Messages -->
      <MessageItem
        v-for="(msg, idx) in store.messages"
        :key="msg.id || msg.nonce"
        :message="msg"
        :grouped="isGrouped(idx)"
        :is-speaking="voiceStore.isSpeaking(msg.user_id)"
      />
    </template>
  </div>
</template>
