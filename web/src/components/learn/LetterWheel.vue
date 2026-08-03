<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { useSpeak } from '@/composables/useSpeak.js'
import LetterTray from './LetterTray.vue'
import HintList from './HintList.vue'
import SpeakButton from './SpeakButton.vue'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const { speak } = useSpeak()
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
    if (out.correct) speak(word)
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
      <!-- 完關單字回顧：最後一字猜完直接進完關畫面，釋義在這裡補看 -->
      <ul class="done-words">
        <li v-for="(s, i) in learn.level?.slots || []" :key="i">
          <b class="dw-word">{{ s.word }}</b>
          <SpeakButton :word="s.word" />
          <span class="dw-def">{{ s.definition }}</span>
        </li>
      </ul>
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
.done-words {
    list-style: none; padding: 0; margin: 20px 0;
    display: flex; flex-direction: column; gap: 6px; text-align: left;
}
.done-words li { display: flex; gap: 12px; align-items: baseline; border: 1px solid var(--border); padding: 8px 12px; }
.dw-word { font-family: var(--font-mono); flex-shrink: 0; }
.dw-def { color: var(--text-muted); font-size: 12px; }
</style>
