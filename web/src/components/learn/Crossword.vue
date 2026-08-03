<!-- web/src/components/learn/Crossword.vue -->
<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { useSpeak } from '@/composables/useSpeak.js'
import { buildCells } from './crosswordGrid.js'
import LetterTray from './LetterTray.vue'
import HintList from './HintList.vue'
import SpeakButton from './SpeakButton.vue'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const { speak } = useSpeak()
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

// 固定關卡：完關後可直接接下一關（最後一關或非 campaign 時為 0 = 不顯示）
const nextCampaign = computed(() => {
    const no = learn.crossword?.campaign
    if (!no || !learn.campaign) return 0
    return no < learn.campaign.total ? no + 1 : 0
})

async function goNextCampaign() {
    await learn.startCampaign(nextCampaign.value)
}

// 網格格子 ↔ 提示列雙向高亮；用 mouseenter/leave（觸控 tap 會觸發相容 mouse 事件，同一份邏輯兩端可用）
const activeIndexes = ref([])
function activate(idxs) { activeIndexes.value = idxs }
function deactivate() { activeIndexes.value = [] }

async function onSubmit(word) {
    const out = await learn.guessCrossword(word)
    tray.value.reset()

    if (!out) return

    tray.value.setFeedback(out.correct)
    if (out.correct) speak(word)
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
    <!-- minmax(0,32px) + aspect-ratio：窄螢幕（手機扣掉 rail 後）欄寬等比縮小，不橫向爆版 -->
    <div
      v-if="!done"
      class="crossword__grid"
      :style="{ gridTemplateColumns: `repeat(${cols}, minmax(0, 32px))` }"
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
      <!-- 完關單字回顧：最後一字猜完直接進完關畫面，釋義在這裡補看 -->
      <ul class="done-words">
        <li v-for="(w, i) in words" :key="i">
          <b class="dw-word">{{ w.word }}</b>
          <SpeakButton :word="w.word" />
          <span class="dw-def">{{ w.definition }}</span>
        </li>
      </ul>
      <div class="cw-done-actions">
        <button v-if="nextCampaign" class="cw-back primary" @click="goNextCampaign">
          {{ t('learn.nextLevel') }}
        </button>
        <button class="cw-back" @click="emit('exit')">{{ t('learn.backToHub') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.crossword { display: flex; flex-direction: column; gap: 20px; max-width: 480px; margin: 0 auto; padding: 24px 16px; align-items: center; }
.crossword__grid { display: grid; gap: 2px; max-width: 100%; }
.cw-cell {
    aspect-ratio: 1 / 1; min-width: 0;
    display: flex; align-items: center; justify-content: center;
    font-family: var(--font-mono); font-size: clamp(11px, 4.2vw, 16px); text-transform: lowercase;
}
.cw-cell.filled { border: 1px solid var(--border-strong); background: var(--bg-secondary); }
.cw-cell.active { border-color: var(--accent); color: var(--accent); }
.cw-cell.empty { border: none; background: transparent; }
.crossword__done { text-align: center; padding: 40px 0; width: 100%; }
.done-words {
    list-style: none; padding: 0; margin: 20px 0;
    display: flex; flex-direction: column; gap: 6px; text-align: left;
}
.done-words li { display: flex; gap: 12px; align-items: baseline; border: 1px solid var(--border); padding: 8px 12px; }
.dw-word { font-family: var(--font-mono); flex-shrink: 0; }
.dw-def { color: var(--text-muted); font-size: 12px; }
.cw-done-actions { display: flex; gap: 8px; justify-content: center; }
.cw-back { padding: 8px 18px; background: transparent; color: inherit; border: 1px solid var(--border); cursor: pointer; }
.cw-back:hover { background: var(--accent); color: var(--bg-tertiary); }
.cw-back.primary { border-color: var(--accent); color: var(--accent); }
.cw-back.primary:hover { color: var(--bg-tertiary); }
.cw-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
