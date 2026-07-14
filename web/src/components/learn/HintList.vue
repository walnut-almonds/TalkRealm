<!-- web/src/components/learn/HintList.vue -->
<script setup>
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { maskSegments } from './mask.js'

defineProps({
    items: { type: Array, required: true }, // [{ index, length, masked, definition, solved, hintTier }]
})
const emit = defineEmits(['hint', 'reveal'])
const learn = useLearnStore()
const { t } = useI18n()
</script>

<template>
  <div class="hint-list">
    <div v-for="item in items" :key="item.index" :class="['hl-row', { solved: item.solved }]">
      <span class="hl-word">
        <span
          v-for="(seg, j) in maskSegments(item.masked, learn.hardMode && !item.solved)"
          :key="j"
          :class="['hl-char', { gap: seg.gap }]"
        >{{ seg.ch }}</span>
      </span>
      <span v-if="item.definition" class="hl-def">{{ item.definition }}</span>
      <div v-if="!item.solved" class="hl-actions">
        <button
          v-if="item.hintTier < 2"
          class="hl-hint-btn"
          @click="emit('hint', item.index)"
        >{{ item.hintTier === 0 ? t('learn.hintLetter') : t('learn.hintDefinition') }}</button>
        <button class="hl-hint-btn reveal" @click="emit('reveal', item.index)">
          {{ t('learn.hintReveal') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hint-list { display: flex; flex-direction: column; gap: 8px; width: 100%; }
.hl-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; padding: 8px; border: 1px solid var(--border); }
.hl-row.solved { opacity: 0.55; }
.hl-word { display: flex; gap: 2px; }
.hl-char {
    width: 20px; height: 26px; display: inline-flex; align-items: center; justify-content: center;
    border-bottom: 1px solid var(--border-strong); font-family: var(--font-mono); font-size: 14px; text-transform: lowercase;
}
.hl-char.gap { width: 56px; }
.hl-def { font-size: 12px; color: var(--text-muted); }
.hl-actions { display: flex; gap: 6px; margin-left: auto; }
.hl-hint-btn {
    padding: 4px 10px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer; font-size: 11px;
    font-family: var(--font-mono);
}
.hl-hint-btn:hover { border-color: var(--accent); color: var(--accent); }
.hl-hint-btn.reveal:hover { border-color: var(--danger); color: var(--danger); }
</style>
