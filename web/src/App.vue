<script setup>
import { onMounted, watch } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import { useVoice } from '@/composables/useVoice.js'
import AuthPage from './components/AuthPage.vue'
import MainLayout from './components/MainLayout.vue'
import Notification from './components/Notification.vue'
import LoadingSpinner from './components/LoadingSpinner.vue'
import Lightbox from './components/Lightbox.vue'

const store = useAppStore()
const ws = useWebSocket()
const voice = useVoice(store)

// Wire WebSocket events to store and voice composable
ws.onMessage((type, data) => {
  switch (type) {
    case 'message':            store.handleNewMessage(data); break
    case 'message_update':     store.handleMessageUpdate(data); break
    case 'message_delete':     store.handleMessageDelete(data); break
    case 'typing':             store.handleTyping(data); break
    case 'user_status':        store.handleUserStatus(data); break
    case 'channel_create':     store.handleChannelCreate(data); break
    case 'channel_update':     store.handleChannelUpdate(data); break
    case 'channel_delete':     store.handleChannelDelete(data); break
    case 'guild_update':       store.handleGuildUpdate(data); break
    case 'guild_delete':       store.handleGuildDelete(data); break
    case 'member_add':         store.handleMemberAdd(data); break
    case 'member_remove':      store.handleMemberRemove(data); break
    case 'member_update':      store.handleMemberUpdate(data); break
    case 'voice_state_update': voice.handleVoiceStateUpdate(data); break
    case 'translation_ready':   store.handleTranslationReady(data); break
    case 'reconnect_failed':   store.showNotification('WebSocket 連線失敗，請重新整理頁面', 'error'); break
  }
})

// 監聴目前頻道變化，自動訂閱/取消訂閱 WS（包含初始載入自動切換的情況）
watch(() => store.currentChannel?.id, (newId, oldId) => {
  if (oldId && oldId !== newId) ws.unsubscribeFromChannel(oldId)
  if (newId) ws.subscribeToChannel(newId)
})

onMounted(async () => {
  await store.checkAuth()
  if (store.isAuthenticated) {
    ws.connect(localStorage.getItem('talkrealm_token'))
  }
})

watch(() => store.isAuthenticated, (val) => {
  if (val) ws.connect(localStorage.getItem('talkrealm_token'))
  else ws.disconnect()
})
</script>

<template>
  <AuthPage v-if="!store.isAuthenticated" />
  <MainLayout v-else :voice="voice" />
  <Notification />
  <LoadingSpinner />
  <Lightbox />
</template>
