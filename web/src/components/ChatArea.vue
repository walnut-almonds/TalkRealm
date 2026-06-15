<script setup>
import { inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import MessageList from './MessageList.vue'
import MessageInput from './MessageInput.vue'
import TypingIndicator from './TypingIndicator.vue'

const emit = defineEmits(['toggle-members', 'open-sidebar'])
const voice = inject('voice')
const store = useAppStore()
const { t } = useI18n()
</script>

<template>
  <div class="main-content">
    <!-- Channel header -->
    <div class="channel-header">
      <div class="channel-info">
        <button class="mobile-hamburger" @click="emit('open-sidebar')" :title="t('chatArea.channelList')">
          <i class="fas fa-bars"></i>
        </button>
        <i :class="['fas', store.currentChannel?.type === 'voice' ? 'fa-volume-up' : 'fa-hashtag']"></i>
        <h3>{{ store.currentChannel?.name || t('chatArea.welcome') }}</h3>
        <span v-if="store.currentChannel?.topic" class="channel-topic">
          {{ store.currentChannel.topic }}
        </span>
      </div>
      <div class="channel-actions">
        <i class="fas fa-bell" :title="t('chatArea.notificationSettings')"></i>
        <i class="fas fa-thumbtack" :title="t('chatArea.pinnedMessages')"></i>
        <i class="fas fa-users" :title="t('chatArea.memberList')" @click="emit('toggle-members')"></i>
        <i class="fas fa-search" :title="t('chatArea.search')"></i>
      </div>
    </div>

    <!-- Messages -->
    <MessageList />

    <!-- Typing indicator -->
    <TypingIndicator />

    <!-- Input -->
    <MessageInput v-if="store.currentChannel" />
  </div>
</template>
