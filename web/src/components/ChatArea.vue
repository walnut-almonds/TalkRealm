<script setup>
import { inject } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import MessageList from './MessageList.vue'
import MessageInput from './MessageInput.vue'
import TypingIndicator from './TypingIndicator.vue'

const emit = defineEmits(['toggle-members', 'open-sidebar'])
const voice = inject('voice')
const store = useAppStore()
</script>

<template>
  <div class="main-content">
    <!-- Channel header -->
    <div class="channel-header">
      <div class="channel-info">
        <button class="mobile-hamburger" @click="emit('open-sidebar')" title="頻道列表">
          <i class="fas fa-bars"></i>
        </button>
        <i :class="['fas', store.currentChannel?.type === 'voice' ? 'fa-volume-up' : 'fa-hashtag']"></i>
        <h3>{{ store.currentChannel?.name || '歡迎' }}</h3>
        <span v-if="store.currentChannel?.topic" class="channel-topic">
          {{ store.currentChannel.topic }}
        </span>
      </div>
      <div class="channel-actions">
        <i class="fas fa-bell" title="通知設定"></i>
        <i class="fas fa-thumbtack" title="釘選訊息"></i>
        <i class="fas fa-users" title="成員列表" @click="emit('toggle-members')"></i>
        <i class="fas fa-search" title="搜尋"></i>
      </div>
    </div>

    <!-- Messages -->
    <MessageList />

    <!-- Typing indicator -->
    <TypingIndicator />

    <!-- Input -->
    <MessageInput v-if="store.currentChannel && store.currentChannel.type !== 'voice'" />
  </div>
</template>
