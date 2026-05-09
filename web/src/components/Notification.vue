<script setup>
import { useAppStore } from '@/stores/useAppStore.js'
const store = useAppStore()
</script>

<template>
  <Teleport to="body">
    <Transition name="notif">
      <div v-if="store.notification.visible" :class="['notification-toast', 'show', store.notification.type]">
        {{ store.notification.message }}
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.notif-enter-active, .notif-leave-active { transition: transform 0.25s, opacity 0.25s; }
.notif-enter-from, .notif-leave-to { transform: translateY(100px); opacity: 0; }
</style>
