<script setup>
import { useAppStore } from '@/stores/useAppStore.js'
const store = useAppStore()
</script>

<template>
  <div class="typing-indicator">
    <template v-if="store.typingList.length">
      <div class="typing-dots">
        <span></span><span></span><span></span>
      </div>
      <span v-if="store.typingList.length === 1">
        <strong>{{ store.typingList[0].username }}</strong> 正在輸入...
      </span>
      <span v-else-if="store.typingList.length === 2">
        <strong>{{ store.typingList[0].username }}</strong> 和
        <strong>{{ store.typingList[1].username }}</strong> 正在輸入...
      </span>
      <span v-else>
        多人正在輸入...
      </span>
    </template>
  </div>
</template>
