<script setup>
import { useAppStore } from '@/stores/useAppStore.js'
const store = useAppStore()
</script>

<template>
  <Teleport to="body">
    <div v-if="store.isLoading" class="loading-overlay">
      <div class="loading-spinner"></div>
    </div>
  </Teleport>
</template>
