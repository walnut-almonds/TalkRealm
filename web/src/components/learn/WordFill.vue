<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()

const activeSlot = ref(0)
const input = ref('')
const feedback = ref('') // '' | 'correct' | 'wrong'

const done = computed(() => learn.level?.slots.every(s => s.solved))
const current = computed(() => learn.level?.slots[activeSlot.value])

function selectSlot(i) {
    if (learn.level.slots[i].solved) return
    activeSlot.value = i
    input.value = ''
    feedback.value = ''
}

async function submit() {
    const word = input.value.trim().toLowerCase()
    if (!word || !current.value || current.value.solved) return

    const out = await learn.guess(activeSlot.value, word)
    if (!out) return

    feedback.value = out.correct ? 'correct' : 'wrong'
    input.value = ''

    if (out.correct && !out.completed) {
        // 跳到下一個未解格
        const next = learn.level.slots.findIndex(s => !s.solved)
        if (next >= 0) activeSlot.value = next
    }
}
</script>

<template>
  <div class="wordfill">
    <div v-if="!done" class="wordfill__board">
      <button
        v-for="(slot, i) in learn.level.slots"
        :key="i"
        :class="['wf-slot', { active: i === activeSlot, solved: slot.solved }]"
        @click="selectSlot(i)"
      >
        <span class="wf-slot__word">
          <span v-for="(ch, j) in slot.masked" :key="j" class="wf-char">
            {{ ch === '_' ? '' : ch }}
          </span>
        </span>
        <span class="wf-slot__def">{{ slot.definition }}</span>
      </button>
    </div>

    <div v-if="!done" class="wordfill__input">
      <input
        v-model="input"
        :placeholder="t('learn.typeAnswer')"
        :class="feedback"
        maxlength="8"
        autocapitalize="off"
        autocomplete="off"
        spellcheck="false"
        @keyup.enter="submit"
      />
      <button class="wf-submit" @click="submit">{{ t('learn.submit') }}</button>
    </div>

    <div v-else class="wordfill__done">
      <h3>{{ t('learn.levelComplete') }}</h3>
      <p class="wf-xp">+{{ learn.lastOutcome?.total_xp || 0 }} XP</p>
      <button class="wf-submit" @click="emit('exit')">{{ t('learn.backToHub') }}</button>
    </div>
  </div>
</template>

<style scoped>
/* Kinetic Noir：直角、hairline、accent 唯一裝飾色、Geist Mono 計分 */
.wordfill { display: flex; flex-direction: column; gap: 16px; max-width: 560px; margin: 0 auto; padding: 24px 16px; }
.wordfill__board { display: flex; flex-direction: column; gap: 8px; }
.wf-slot {
    display: flex; flex-direction: column; gap: 6px; padding: 12px;
    background: var(--bg-secondary); border: 1px solid var(--border);
    text-align: left; cursor: pointer;
}
.wf-slot.active { border-color: var(--accent); }
.wf-slot.solved { opacity: 0.55; cursor: default; }
.wf-slot__word { display: flex; gap: 4px; }
.wf-char {
    width: 24px; height: 30px; display: inline-flex; align-items: center; justify-content: center;
    border-bottom: 1px solid var(--border-strong);
    font-family: var(--font-mono); font-size: 18px; text-transform: lowercase;
}
.wf-slot__def { font-size: 13px; color: var(--text-muted); }
.wordfill__input { display: flex; gap: 8px; }
.wordfill__input input {
    flex: 1; padding: 10px 12px; background: var(--bg-input);
    border: 1px solid var(--border); color: inherit;
    font-family: var(--font-mono);
}
.wordfill__input input.correct { border-color: var(--accent); }
.wordfill__input input.wrong { border-color: var(--danger); }
.wf-submit {
    padding: 10px 20px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.wf-submit:hover { background: var(--accent); color: var(--bg-tertiary); }
.wordfill__done { text-align: center; padding: 40px 0; }
.wf-xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
