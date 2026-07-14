<!-- web/src/components/learn/LetterTray.vue -->
<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({ letters: { type: String, required: true } })
const emit = defineEmits(['submit'])
const { t } = useI18n()

const picked = ref([]) // [{ch, idx}] 已選字母（idx 對應 letters 位置，同字母可區分）
const feedback = ref('')

const letterArr = computed(() => (props.letters || '').split(''))
const word = computed(() => picked.value.map(p => p.ch).join(''))

function toggle(ch, idx) {
    const at = picked.value.findIndex(p => p.idx === idx)
    if (at >= 0) picked.value.splice(at, 1)
    else picked.value.push({ ch, idx })
    feedback.value = ''
}

function clearPick() { picked.value = []; feedback.value = '' }

function submit() {
    if (word.value.length < 3) return
    emit('submit', word.value)
}

function setFeedback(correct) {
    feedback.value = correct ? 'correct' : 'wrong'
}

function reset() {
    picked.value = []
    feedback.value = ''
}

defineExpose({ setFeedback, reset })
</script>

<template>
  <div class="letter-tray">
    <div :class="['lt-current', feedback]">{{ word || ' ' }}</div>
    <div class="lt-letters">
      <button
        v-for="(ch, idx) in letterArr"
        :key="idx"
        :class="['lt-letter', { picked: picked.some(p => p.idx === idx) }]"
        @click="toggle(ch, idx)"
      >{{ ch }}</button>
    </div>
    <div class="lt-actions">
      <button class="lt-btn" @click="clearPick">{{ t('learn.clear') }}</button>
      <button class="lt-btn primary" @click="submit">{{ t('learn.submit') }}</button>
    </div>
  </div>
</template>

<style scoped>
.letter-tray { display: flex; flex-direction: column; gap: 14px; align-items: center; }
.lt-current {
    min-height: 32px; min-width: 160px; text-align: center;
    font-family: var(--font-mono); font-size: 22px; letter-spacing: 4px;
    border-bottom: 1px solid var(--border-strong); padding: 2px 12px;
}
.lt-current.correct { border-color: var(--accent); color: var(--accent); }
.lt-current.wrong { border-color: var(--danger); color: var(--danger); }
.lt-letters { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.lt-letter {
    width: 44px; height: 44px; font-size: 18px; text-transform: lowercase;
    background: var(--bg-secondary); color: inherit;
    border: 1px solid var(--border); cursor: pointer;
    font-family: var(--font-mono);
}
.lt-letter.picked { border-color: var(--accent); color: var(--accent); }
.lt-actions { display: flex; gap: 8px; }
.lt-btn {
    padding: 8px 18px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.lt-btn.primary:hover { background: var(--accent); color: var(--bg-tertiary); }
</style>
