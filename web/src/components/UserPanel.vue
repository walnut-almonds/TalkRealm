<script setup>
import { useAppStore } from '@/stores/useAppStore.js'
import { getStatusText } from '@/utils/format.js'

const emit = defineEmits(['settings'])
const store = useAppStore()

async function logout() {
  await store.handleLogout()
}
</script>

<template>
  <div class="user-panel">
    <div class="user-info">
      <div class="user-avatar">
        <img v-if="store.user?.avatar" :src="store.user.avatar" :alt="store.user.username" />
        <i v-else class="fas fa-user"></i>
      </div>
      <div class="user-details">
        <div class="user-name">{{ store.user?.nickname || store.user?.username }}</div>
        <div class="user-status">
          <span :class="['status-indicator', store.user?.status || 'online']"></span>
          <span>{{ getStatusText(store.user?.status) }}</span>
        </div>
      </div>
    </div>
    <div class="user-actions">
      <button class="btn-icon" title="使用者設定" @click="emit('settings')">
        <i class="fas fa-cog"></i>
      </button>
      <button class="btn-icon" title="登出" @click="logout">
        <i class="fas fa-sign-out-alt"></i>
      </button>
    </div>
  </div>
</template>
