<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { maskSegments } from './mask.js'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()

const picked = ref([]) // [{ch, idx}] 已選字母（idx 對應 letters 位置，同字母可區分）
const feedback = ref('')

const letters = computed(() => (learn.level?.letters || '').split(''))
const word = computed(() => picked.value.map(p => p.ch).join(''))
const done = computed(() => learn.level?.slots.every(s => s.solved))

function toggle(ch, idx) {
    const at = picked.value.findIndex(p => p.idx === idx)
    if (at >= 0) picked.value.splice(at, 1)
    else picked.value.push({ ch, idx })
    feedback.value = ''
}

function clearPick() { picked.value = []; feedback.value = '' }

async function submit() {
    if (word.value.length < 3) return

    // slot=-1：後端自行判定命中哪一格，store 依 out.slot 回填
    const out = await learn.guess(-1, word.value)
    clearPick()

    if (!out) return
    feedback.value = out.correct ? 'correct' : 'wrong'
}
</script>

<template>
  <div class="wheel">
    <div v-if="!done" class="wheel__slots">
      <div v-for="(slot, i) in learn.level.slots" :key="i" :class="['wh-slot', { solved: slot.solved }]">
        <span class="wh-word">
          <span
            v-for="(seg, j) in maskSegments(slot.masked, learn.hardMode && !slot.solved)"
            :key="j"
            :class="['wh-char', { gap: seg.gap }]"
          >{{ seg.ch }}</span>
        </span>
        <span v-if="slot.solved" class="wh-def">{{ slot.definition }}</span>
      </div>
    </div>

    <div v-if="!done" class="wheel__play">
      <div :class="['wh-current', feedback]">{{ word || ' ' }}</div>
      <div class="wh-letters">
        <button
          v-for="(ch, idx) in letters"
          :key="idx"
          :class="['wh-letter', { picked: picked.some(p => p.idx === idx) }]"
          @click="toggle(ch, idx)"
        >{{ ch }}</button>
      </div>
      <div class="wh-actions">
        <button class="wh-btn" @click="clearPick">{{ t('learn.clear') }}</button>
        <button class="wh-btn primary" @click="submit">{{ t('learn.submit') }}</button>
      </div>
    </div>

    <div v-else class="wheel__done">
      <h3>{{ t('learn.levelComplete') }}</h3>
      <p class="wh-xp">+{{ learn.lastOutcome?.total_xp || 0 }} XP</p>
      <button class="wh-btn primary" @click="emit('exit')">{{ t('learn.backToHub') }}</button>
    </div>
  </div>
</template>

<style scoped>
/* Kinetic Noir：直角、hairline、accent 唯一裝飾色、Geist Mono 計分 */
.wheel { display: flex; flex-direction: column; gap: 20px; max-width: 480px; margin: 0 auto; padding: 24px 16px; }
.wheel__slots { display: flex; flex-direction: column; gap: 8px; }
.wh-slot { display: flex; align-items: baseline; gap: 12px; }
.wh-word { display: flex; gap: 4px; }
.wh-char {
    width: 22px; height: 28px; display: inline-flex; align-items: center; justify-content: center;
    border: 1px solid var(--border);
    font-family: var(--font-mono); font-size: 16px; text-transform: lowercase;
}
/* 困難模式：一段連續缺字只給一個固定寬度格 */
.wh-char.gap { width: 60px; }
.wh-slot.solved .wh-char { border-color: var(--accent); }
.wh-def { font-size: 12px; color: var(--text-muted); }
.wheel__play { display: flex; flex-direction: column; gap: 14px; align-items: center; }
.wh-current {
    min-height: 32px; min-width: 160px; text-align: center;
    font-family: var(--font-mono); font-size: 22px; letter-spacing: 4px;
    border-bottom: 1px solid var(--border-strong); padding: 2px 12px;
}
.wh-current.correct { border-color: var(--accent); color: var(--accent); }
.wh-current.wrong { border-color: var(--danger); color: var(--danger); }
.wh-letters { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.wh-letter {
    width: 44px; height: 44px; font-size: 18px; text-transform: lowercase;
    background: var(--bg-secondary); color: inherit;
    border: 1px solid var(--border); cursor: pointer;
    font-family: var(--font-mono);
}
.wh-letter.picked { border-color: var(--accent); color: var(--accent); }
.wh-actions { display: flex; gap: 8px; }
.wh-btn {
    padding: 8px 18px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.wh-btn.primary:hover { background: var(--accent); color: var(--bg-tertiary); }
.wheel__done { text-align: center; padding: 40px 0; }
.wh-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
