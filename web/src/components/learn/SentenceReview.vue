<!-- web/src/components/learn/SentenceReview.vue -->
<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import { useSpeechInput } from '@/composables/useSpeechInput.js'
import SpeakButton from './SpeakButton.vue'

const RELEARN_MS = 5 * 60 * 1000 // 答錯後 5 分鐘再考（前面沒別的卡就會被 pickNext 直接挑回）

const emit = defineEmits(['exit'])
const learn = useLearnStore()
const { t } = useI18n()
const { supported: micSupported, listening, start: startMic } = useSpeechInput()

const guess = ref('')
const result = ref(null) // SRSAnswerOutcome | null
const submitting = ref(false)
const finished = ref(false)
const totalXP = ref(0)

// 佇列制：答對才退場，答錯排回佇列稍後再考，直到每張都答對過一次。
const queue = ref([]) // [{ index, dueAt }]，fresh 卡 dueAt=0 永遠優先
const current = ref(null) // { index, dueAt }
const retired = ref(0)

const total = computed(() => learn.srs?.total || 0)
const cards = computed(() => learn.srs?.cards || [])
const card = computed(() => (current.value ? cards.value[current.value.index] : null))
// 例句以 "{{}}" 標記挖空 → 拆成前後兩段，中間放輸入框/答案
const parts = computed(() => {
    const [pre, post] = (card.value?.text_en || '').split('{{}}')
    return { pre: pre ?? '', post: post ?? '' }
})

// pickNext 挑出下一張：優先已到期（dueAt<=now）中最早的；都還沒到期就挑最早的
// 那張（＝前面沒別的可考時，答錯的卡不必真的等滿 5 分鐘，直接再出現）
function pickNext() {
    if (!queue.value.length) {
        current.value = null
        finished.value = true
        return
    }

    let best = 0
    for (let i = 1; i < queue.value.length; i++) {
        if (queue.value[i].dueAt < queue.value[best].dueAt) best = i
    }

    current.value = queue.value.splice(best, 1)[0]
}

async function submit() {
    if (!guess.value.trim() || result.value || submitting.value || !current.value) return

    submitting.value = true
    const out = await learn.answerSRS(current.value.index, guess.value)
    submitting.value = false

    if (!out) return

    result.value = out

    if (out.correct) {
        retired.value++
        if (out.completed) totalXP.value = out.total_xp || 0
    }
}

function next() {
    // 答錯 → 排回佇列（5 分鐘後到期）；答對 → 退場，不再排入
    if (result.value && !result.value.correct) {
        queue.value.push({ index: current.value.index, dueAt: Date.now() + RELEARN_MS })
    }

    guess.value = ''
    result.value = null
    pickNext()
}

function mic() {
    startMic((text) => { guess.value = text })
}

function exit() {
    learn.srs = null
    emit('exit')
}

onMounted(() => {
    queue.value = cards.value.map((_, i) => ({ index: i, dueAt: 0 }))
    pickNext()
})
</script>

<template>
  <div class="srs">
    <template v-if="!finished && card">
      <div class="srs__progress">
        <span class="mono"><i class="fas fa-check"></i> {{ retired }} / {{ total }}</span>
        <span v-if="card.is_new" class="srs__new">{{ t('learn.srsNew') }}</span>
      </div>

      <!-- 例句：挖空處在未作答時是輸入框，作答後填入答案並上色 -->
      <p class="srs__sentence">
        <span>{{ parts.pre }}</span>
        <input
          v-if="!result"
          v-model="guess"
          class="srs__blank"
          :style="{ width: `${Math.max(card.length + 1, 4)}ch` }"
          autocapitalize="off"
          autocomplete="off"
          spellcheck="false"
          @keyup.enter="submit"
        />
        <span
          v-else
          :class="['srs__answer', result.correct ? 'ok' : 'bad']"
        >{{ result.answer }}</span>
        <span>{{ parts.post }}</span>
      </p>

      <p v-if="card.trans" class="srs__trans">{{ card.trans }}</p>

      <!-- 作答前：輸入列（打字 + 語音）；作答後：結果 + 下一張 -->
      <div v-if="!result" class="srs__actions">
        <button
          v-if="micSupported"
          :class="['srs__mic', { on: listening }]"
          :title="t('learn.srsVoice')"
          @click="mic"
        ><i class="fas fa-microphone"></i></button>
        <button class="srs__btn primary" :disabled="submitting" @click="submit">
          {{ t('learn.submit') }}
        </button>
      </div>

      <div v-else class="srs__result">
        <p :class="['srs__verdict', result.correct ? 'ok' : 'bad']">
          <i :class="['fas', result.correct ? 'fa-check' : 'fa-xmark']"></i>
          <span>{{ result.correct ? t('learn.srsCorrect') : t('learn.srsWrong') }}</span>
          <SpeakButton :word="result.answer" />
          <span v-if="result.phonetic" class="srs__phon">/{{ result.phonetic }}/</span>
        </p>
        <p v-if="!result.correct" class="srs__yours">
          {{ t('learn.srsYourAnswer') }}: <s>{{ guess }}</s>
        </p>
        <button class="srs__btn primary" @click="next">
          {{ result.completed ? t('learn.srsFinish') : t('learn.srsNext') }}
        </button>
      </div>
    </template>

    <div v-else class="srs__done">
      <h3>{{ t('learn.srsSessionDone') }}</h3>
      <p class="srs__score mono">{{ total }} {{ t('learn.srsWords') }}</p>
      <p class="srs__xp">+{{ totalXP }} XP</p>
      <button class="srs__btn" @click="exit">{{ t('learn.backToHub') }}</button>
    </div>
  </div>
</template>

<style scoped>
.srs { max-width: 560px; margin: 0 auto; padding: 24px 16px; display: flex; flex-direction: column; gap: 20px; }
.srs__progress { display: flex; align-items: center; gap: 10px; color: var(--text-muted); font-size: 13px; }
.srs__new {
    font-family: var(--font-mono); font-size: 11px; padding: 2px 8px;
    border: 1px solid var(--accent); color: var(--accent);
}
.srs__sentence { font-size: 20px; line-height: 1.9; }
.srs__blank {
    background: var(--bg-input); border: none; border-bottom: 2px solid var(--accent);
    color: inherit; font-family: var(--font-mono); font-size: 18px;
    text-align: center; padding: 2px 4px; margin: 0 4px;
}
.srs__answer { font-family: var(--font-mono); margin: 0 4px; border-bottom: 2px solid; padding: 2px 4px; }
.srs__answer.ok { color: var(--accent); border-color: var(--accent); }
.srs__answer.bad { color: var(--danger); border-color: var(--danger); }
.srs__trans { color: var(--text-muted); font-size: 14px; }
.srs__actions, .srs__result { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.srs__btn {
    padding: 10px 22px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.srs__btn.primary { border-color: var(--accent); color: var(--accent); }
.srs__btn.primary:hover { background: var(--accent); color: var(--bg-tertiary); }
.srs__mic {
    width: 42px; height: 42px; display: inline-flex; align-items: center; justify-content: center;
    background: transparent; color: var(--text-muted); border: 1px solid var(--border); cursor: pointer;
}
.srs__mic.on { border-color: var(--danger); color: var(--danger); }
.srs__verdict { display: flex; align-items: center; gap: 8px; font-weight: 600; }
.srs__verdict.ok { color: var(--accent); }
.srs__verdict.bad { color: var(--danger); }
.srs__phon { font-family: var(--font-mono); font-size: 13px; color: var(--text-muted); font-weight: 400; }
.srs__yours { color: var(--text-muted); font-size: 13px; }
.srs__yours s { color: var(--danger); }
.srs__done { text-align: center; padding: 40px 0; }
.srs__score { font-size: 20px; }
.srs__xp { font-family: var(--font-mono); font-size: 24px; color: var(--accent); }
</style>
