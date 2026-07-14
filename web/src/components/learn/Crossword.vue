<!-- web/src/components/learn/Crossword.vue -->
<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { maskSegments } from './mask.js'
import { buildCells } from './crosswordGrid.js'
import LetterTray from './LetterTray.vue'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const tray = ref(null)

const words = computed(() => learn.crossword?.words || [])
const rows = computed(() => learn.crossword?.rows || 0)
const cols = computed(() => learn.crossword?.cols || 0)
const cells = computed(() => buildCells(words.value, rows.value, cols.value))
const bonusWords = computed(() => words.value.filter(w => !w.dir))
const done = computed(() => words.value.length > 0 && words.value.every(w => w.solved))

async function onSubmit(word) {
    const out = await learn.guessCrossword(word)
    tray.value.reset()

    if (!out) return

    tray.value.setFeedback(out.correct)
}
</script>

<template>
  <div class="crossword">
    <div
      v-if="!done"
      class="crossword__grid"
      :style="{ gridTemplateRows: `repeat(${rows}, 32px)`, gridTemplateColumns: `repeat(${cols}, 32px)` }"
    >
      <template v-for="(row, r) in cells" :key="r">
        <div
          v-for="(cell, c) in row"
          :key="c"
          :class="['cw-cell', { filled: cell, empty: !cell }]"
        >{{ cell?.letter }}</div>
      </template>
    </div>

    <div v-if="!done && bonusWords.length" class="crossword__bonus">
      <h4>{{ t('learn.bonusWords') }}</h4>
      <div class="cw-bonus-row">
        <div v-for="(w, i) in bonusWords" :key="i" class="cw-bonus-word">
          <span
            v-for="(seg, j) in maskSegments(w.solved ? w.word : '_'.repeat(w.length), learn.hardMode && !w.solved)"
            :key="j"
            :class="['cw-bonus-char', { gap: seg.gap }]"
          >{{ seg.ch }}</span>
        </div>
      </div>
    </div>

    <LetterTray v-if="!done" ref="tray" :letters="learn.crossword?.letters || ''" @submit="onSubmit" />

    <div v-else class="crossword__done">
      <h3>{{ t('learn.levelComplete') }}</h3>
      <p class="cw-xp">+{{ learn.lastOutcome?.total_xp || 0 }} XP</p>
      <button class="cw-back" @click="emit('exit')">{{ t('learn.backToHub') }}</button>
    </div>
  </div>
</template>

<style scoped>
.crossword { display: flex; flex-direction: column; gap: 20px; max-width: 480px; margin: 0 auto; padding: 24px 16px; align-items: center; }
.crossword__grid { display: grid; gap: 2px; }
.cw-cell {
    display: flex; align-items: center; justify-content: center;
    font-family: var(--font-mono); font-size: 16px; text-transform: lowercase;
}
.cw-cell.filled { border: 1px solid var(--border); }
.cw-cell.empty { border: none; background: transparent; }
.crossword__bonus { width: 100%; }
.crossword__bonus h4 { font-size: 12px; color: var(--text-muted); margin: 0 0 8px; }
.cw-bonus-row { display: flex; flex-wrap: wrap; gap: 10px; }
.cw-bonus-word { display: flex; gap: 2px; }
.cw-bonus-char {
    width: 20px; height: 26px; display: inline-flex; align-items: center; justify-content: center;
    border-bottom: 1px solid var(--border); font-family: var(--font-mono); font-size: 14px; text-transform: lowercase;
}
.cw-bonus-char.gap { width: 60px; }
.crossword__done { text-align: center; padding: 40px 0; }
.cw-back { padding: 8px 18px; background: transparent; color: inherit; border: 1px solid var(--border); cursor: pointer; }
.cw-back:hover { background: var(--accent); color: var(--bg-tertiary); }
.cw-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
