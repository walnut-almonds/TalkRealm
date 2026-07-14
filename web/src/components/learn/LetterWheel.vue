<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { maskSegments } from './mask.js'
import LetterTray from './LetterTray.vue'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const tray = ref(null)

const done = computed(() => learn.level?.slots.every(s => s.solved))

async function onSubmit(word) {
    const out = await learn.guess(-1, word)
    tray.value.reset()

    if (!out) return

    tray.value.setFeedback(out.correct)

    if (out.correct && out.slot >= 0) {
        const s = learn.level.slots[out.slot]
        s.solved = true
        s.word = out.word
        s.masked = out.word
        s.definition = out.definition
    }
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

    <LetterTray v-if="!done" ref="tray" :letters="learn.level.letters" @submit="onSubmit" />

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
.wh-btn {
    padding: 8px 18px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.wh-btn.primary:hover { background: var(--accent); color: var(--bg-tertiary); }
.wheel__done { text-align: center; padding: 40px 0; }
.wh-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
