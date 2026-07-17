<!-- web/src/components/learn/Crossword.vue -->
<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { buildCells } from './crosswordGrid.js'
import LetterTray from './LetterTray.vue'
import HintList from './HintList.vue'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const tray = ref(null)

const words = computed(() => learn.crossword?.words || [])
const rows = computed(() => learn.crossword?.rows || 0)
const cols = computed(() => learn.crossword?.cols || 0)
const cells = computed(() => buildCells(words.value, rows.value, cols.value))
const hintItems = computed(() => words.value.map((w, i) => ({
    index: i, length: w.length, masked: w.masked, definition: w.definition,
    solved: w.solved, hintTier: w.hint_tier || 0,
})))
const done = computed(() => words.value.length > 0 && words.value.every(w => w.solved))

// 網格格子 ↔ 提示列雙向高亮；用 mouseenter/leave（觸控 tap 會觸發相容 mouse 事件，同一份邏輯兩端可用）
const activeIndexes = ref([])
function activate(idxs) { activeIndexes.value = idxs }
function deactivate() { activeIndexes.value = [] }

async function onSubmit(word) {
    const out = await learn.guessCrossword(word)
    tray.value.reset()

    if (!out) return

    tray.value.setFeedback(out.correct)
}

async function onHint(index) {
    await learn.hintCrossword(index)
}

async function onReveal(index) {
    await learn.revealCrossword(index)
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
          :class="['cw-cell', { filled: cell, empty: !cell, active: cell && cell.words.some(i => activeIndexes.includes(i)) }]"
          @mouseenter="cell && activate(cell.words)"
          @mouseleave="deactivate()"
        >{{ cell?.letter }}</div>
      </template>
    </div>

    <HintList
      v-if="!done"
      :items="hintItems"
      :active-indexes="activeIndexes"
      @hint="onHint"
      @reveal="onReveal"
      @activate="activate"
      @deactivate="deactivate"
    />

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
.cw-cell.filled { border: 1px solid var(--border-strong); background: var(--bg-secondary); }
.cw-cell.active { border-color: var(--accent); color: var(--accent); }
.cw-cell.empty { border: none; background: transparent; }
.crossword__done { text-align: center; padding: 40px 0; }
.cw-back { padding: 8px 18px; background: transparent; color: inherit; border: 1px solid var(--border); cursor: pointer; }
.cw-back:hover { background: var(--accent); color: var(--bg-tertiary); }
.cw-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
