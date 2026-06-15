<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
const store = useAppStore()
const { t } = useI18n()

const typingText = computed(() => {
  const list = store.typingList
  if (list.length === 1) return t('typing.one', { user1: list[0].username })
  if (list.length === 2) return t('typing.two', { user1: list[0].username, user2: list[1].username })
  return t('typing.many')
})
</script>

<template>
  <div class="typing-indicator">
    <template v-if="store.typingList.length">
      <div class="typing-dots">
        <span></span><span></span><span></span>
      </div>
      <span>{{ typingText }}</span>
    </template>
  </div>
</template>
