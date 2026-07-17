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

// 圓形佈局：字母沿半徑 R 均分，12 點鐘方向起始
const SIZE = 220
const R = 82
const HIT_R2 = 26 * 26 // 拖曳命中半徑（平方，省開根號）
const positions = computed(() => letterArr.value.map((_, i) => {
    const ang = -Math.PI / 2 + (2 * Math.PI * i) / letterArr.value.length
    return { x: SIZE / 2 + R * Math.cos(ang), y: SIZE / 2 + R * Math.sin(ang) }
}))

// 拖曳連線：pointerdown 起手、滑入字母加選、滑回上一顆撤銷、放開送出（Wordscapes 慣例）。
// 命中判定用字母中心距離而非 DOM 事件——setPointerCapture 後 pointerenter 不會落在其他元素上。
// 點選（tap/鍵盤）仍保留：tap 走同一套 pointer 邏輯，鍵盤 Enter 走按鈕 click。
const wheelEl = ref(null)
const dragging = ref(false)
const pointer = ref({ x: 0, y: 0 })
let dragMoved = false
let pendingRemove = -1
let rect = null

function hitIndex(e) {
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    let best = -1
    let bestD = HIT_R2
    positions.value.forEach((p, i) => {
        const d = (p.x - x) ** 2 + (p.y - y) ** 2
        if (d < bestD) { bestD = d; best = i }
    })
    return best
}

function onPointerDown(e) {
    rect = wheelEl.value.getBoundingClientRect()
    const idx = hitIndex(e)
    if (idx < 0) return

    dragging.value = true
    dragMoved = false
    pendingRemove = -1
    feedback.value = ''
    pointer.value = { x: e.clientX - rect.left, y: e.clientY - rect.top }

    if (picked.value.some(p => p.idx === idx)) pendingRemove = idx // tap 已選字母 = 取消選取（放開時才生效）
    else picked.value.push({ ch: letterArr.value[idx], idx })

    try { wheelEl.value.setPointerCapture(e.pointerId) } catch { /* 合成/測試事件沒有 active pointer，捕捉失敗不影響拖曳 */ }
}

function onPointerMove(e) {
    if (!dragging.value) return

    pointer.value = { x: e.clientX - rect.left, y: e.clientY - rect.top }
    const idx = hitIndex(e)
    if (idx < 0) return

    const last = picked.value[picked.value.length - 1]
    if (last && idx === last.idx) return

    dragMoved = true
    pendingRemove = -1
    const prev = picked.value[picked.value.length - 2]
    if (prev && idx === prev.idx) picked.value.pop() // 滑回上一顆 = 撤銷
    else if (!picked.value.some(p => p.idx === idx)) picked.value.push({ ch: letterArr.value[idx], idx })
}

function onPointerUp() {
    if (!dragging.value) return

    dragging.value = false

    if (!dragMoved) {
        // 純 tap：只增減選取，送出交給按鈕
        if (pendingRemove >= 0) {
            const at = picked.value.findIndex(p => p.idx === pendingRemove)
            if (at >= 0) picked.value.splice(at, 1)
        }
        return
    }

    if (word.value.length >= 3) emit('submit', word.value)
    else picked.value = []
}

function onPointerCancel() { dragging.value = false }

const linePoints = computed(() => {
    const pts = picked.value.map(p => positions.value[p.idx]).map(p => `${p.x},${p.y}`)
    if (dragging.value) pts.push(`${pointer.value.x},${pointer.value.y}`)
    return pts.join(' ')
})

// 鍵盤（Enter/Space 觸發 click）：滑鼠/觸控被 pointer-events:none 擋掉，不會走到這裡
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
    <div
      ref="wheelEl"
      class="lt-wheel"
      :style="{ width: `${SIZE}px`, height: `${SIZE}px` }"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerCancel"
    >
      <svg class="lt-lines" :viewBox="`0 0 ${SIZE} ${SIZE}`">
        <polyline :points="linePoints" />
      </svg>
      <button
        v-for="(ch, idx) in letterArr"
        :key="idx"
        :class="['lt-letter', { picked: picked.some(p => p.idx === idx) }]"
        :style="{ left: `${positions[idx].x}px`, top: `${positions[idx].y}px` }"
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
.lt-wheel { position: relative; touch-action: none; cursor: pointer; user-select: none; }
.lt-lines { position: absolute; inset: 0; width: 100%; height: 100%; }
.lt-lines polyline { fill: none; stroke: var(--accent); stroke-width: 2; opacity: 0.7; }
.lt-letter {
    position: absolute; transform: translate(-50%, -50%);
    width: 44px; height: 44px; font-size: 18px; text-transform: lowercase;
    background: var(--bg-secondary); color: inherit;
    border: 1px solid var(--border); cursor: pointer;
    font-family: var(--font-mono);
    pointer-events: none; /* 滑鼠/觸控由 wheel 的 pointer 邏輯統一處理；按鈕保留給鍵盤焦點 */
}
.lt-letter.picked { border-color: var(--accent); color: var(--accent); }
.lt-actions { display: flex; gap: 8px; }
.lt-btn {
    padding: 8px 18px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
}
.lt-btn.primary:hover { background: var(--accent); color: var(--bg-tertiary); }
</style>
