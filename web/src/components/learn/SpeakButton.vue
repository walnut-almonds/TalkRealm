<!-- web/src/components/learn/SpeakButton.vue -->
<script setup>
import { useSpeak } from '@/composables/useSpeak.js'

const props = defineProps({ word: { type: String, required: true } })
const { supported, speak } = useSpeak()
</script>

<template>
  <button
    v-if="supported"
    class="speak-btn"
    type="button"
    :aria-label="`pronounce ${word}`"
    :title="`pronounce ${word}`"
    @click.stop="speak(word)"
  ><i class="fas fa-volume-up"></i></button>
</template>

<style scoped>
.speak-btn {
    display: inline-flex; align-items: center; justify-content: center;
    width: 24px; height: 24px; padding: 0; flex-shrink: 0;
    background: transparent; color: var(--text-muted);
    border: 1px solid var(--border); cursor: pointer; font-size: 11px;
}
.speak-btn:hover { border-color: var(--accent); color: var(--accent); }
</style>
