<script setup>
import { useAppStore } from '@/stores/useAppStore.js'
const store = useAppStore()
function close() { store.lightboxUrl = null }
function onKeydown(e) { if (e.key === 'Escape') close() }
</script>

<template>
  <Teleport to="body">
    <div
      v-if="store.lightboxUrl"
      class="lightbox-overlay"
      @click="close"
      @keydown.esc="close"
    >
      <span class="lightbox-close" @click.stop="close">&times;</span>
      <img :src="store.lightboxUrl" alt="" @click.stop />
    </div>
  </Teleport>
</template>
