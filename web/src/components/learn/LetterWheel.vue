<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import LetterTray from './LetterTray.vue'
import HintList from './HintList.vue'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const tray = ref(null)

const done = computed(() => learn.level?.slots.every(s => s.solved))
const hintItems = computed(() => (learn.level?.slots || []).map((s, i) => ({
    index: i, length: s.length, masked: s.masked, definition: s.definition,
    solved: s.solved, hintTier: s.hint_tier || 0,
})))

async function onSubmit(word) {
    // 回填 solved slot 的責任在 useLearnStore 的 guess() action，這裡只負責畫面回饋
    const out = await learn.guess(-1, word)
    tray.value.reset()

    if (!out) return

    tray.value.setFeedback(out.correct)
}

async function onHint(index) {
    await learn.hint(index)
}

async function onReveal(index) {
    await learn.reveal(index)
}
</script>

<template>
  <div class="wheel">
    <HintList v-if="!done" :items="hintItems" @hint="onHint" @reveal="onReveal" />

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
.wh-btn {
    padding: 8px 18px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.wh-btn.primary:hover { background: var(--accent); color: var(--bg-tertiary); }
.wheel__done { text-align: center; padding: 40px 0; }
.wh-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
