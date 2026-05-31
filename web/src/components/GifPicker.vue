<script setup>
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { api } from '@/api/index.js'

const emit = defineEmits(['select', 'close'])

const rootEl = ref(null)
const gridEl = ref(null)
const query = ref('')
const gifs = ref([])
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')
const nextCursor = ref('')
const hasMore = ref(false)

const hoverGif = ref(null)
const previewPos = ref({ x: 0, y: 0 })
const PAGE_SIZE = 10

let searchTimer = null

function resetList() {
  gifs.value = []
  nextCursor.value = ''
  hasMore.value = false
}

function mergeByID(origin, incoming) {
  const seen = new Set(origin.map(item => item.id))
  const merged = [...origin]
  incoming.forEach((item) => {
    if (seen.has(item.id)) return
    seen.add(item.id)
    merged.push(item)
  })
  return merged
}

async function loadGifs(keyword = '', { append = false, cursor = '' } = {}) {
  if (append) {
    if (loadingMore.value || loading.value) return
    loadingMore.value = true
  } else {
    if (loading.value) return
    loading.value = true
    error.value = ''
  }

  try {
    const result = await api.searchGIFs(keyword, PAGE_SIZE, cursor)
    const items = result?.items || []
    nextCursor.value = result?.next || ''
    hasMore.value = Boolean(nextCursor.value)
    gifs.value = append ? mergeByID(gifs.value, items) : items
  } catch (e) {
    if (!append) resetList()
    error.value = e?.message || 'GIF 服務暫時無法使用'
  } finally {
    if (append) loadingMore.value = false
    else loading.value = false
  }
}

function loadMoreIfNeeded() {
  if (!hasMore.value || !nextCursor.value || loading.value || loadingMore.value) return
  loadGifs(query.value.trim(), { append: true, cursor: nextCursor.value })
}

function onGridScroll(e) {
  const el = e.target
  if (!el) return
  const remain = el.scrollHeight - el.scrollTop - el.clientHeight
  if (remain < 140) loadMoreIfNeeded()
}

function selectGif(gif) {
  emit('select', gif)
  emit('close')
}

function closePicker() {
  emit('close')
}

function onDocPointerDown(e) {
  if (!rootEl.value) return
  const target = e.target
  if (target instanceof Node && !rootEl.value.contains(target)) {
    closePicker()
  }
}

function calcPreviewPos(clientX, clientY) {
  const previewW = 350
  const previewH = 270
  const offset = 18
  const pad = 12

  let x = clientX + offset
  let y = clientY + offset
  if (x + previewW > window.innerWidth - pad) x = clientX - previewW - offset
  if (y + previewH > window.innerHeight - pad) y = window.innerHeight - previewH - pad
  if (y < pad) y = pad

  previewPos.value = { x, y }
}

function onGifEnter(gif, e) {
  hoverGif.value = gif
  calcPreviewPos(e.clientX, e.clientY)
}

function onGifMove(e) {
  if (!hoverGif.value) return
  calcPreviewPos(e.clientX, e.clientY)
}

function onGifLeave() {
  hoverGif.value = null
}

watch(query, (val) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    resetList()
    loadGifs(val.trim())
  }, 280)
})

onMounted(() => {
  loadGifs('')
  document.addEventListener('pointerdown', onDocPointerDown)
})

onUnmounted(() => {
  clearTimeout(searchTimer)
  document.removeEventListener('pointerdown', onDocPointerDown)
})
</script>

<template>
  <div ref="rootEl" class="gif-picker" role="dialog" aria-label="GIF 選擇器">
    <div class="gif-picker-header">
      <div class="gif-picker-title">
        <i class="fas fa-film"></i>
        <span>GIF</span>
      </div>
      <button class="gif-picker-close" title="關閉" @click="closePicker">
        <i class="fas fa-times"></i>
      </button>
    </div>

    <div class="gif-picker-search-wrap">
      <i class="fas fa-search"></i>
      <input
        v-model="query"
        type="text"
        class="gif-picker-search"
        placeholder="搜尋 GIF（例如：happy、cat、wow）"
      />
    </div>

    <div v-if="loading" class="gif-picker-state">載入中...</div>
    <div v-else-if="error" class="gif-picker-state gif-picker-state-error">{{ error }}</div>
    <div v-else-if="!gifs.length" class="gif-picker-state">找不到 GIF，換個關鍵字試試</div>

    <div v-else ref="gridEl" class="gif-picker-grid" @scroll="onGridScroll">
      <button
        v-for="gif in gifs"
        :key="gif.id"
        class="gif-card"
        :title="gif.title || 'GIF'"
        @mouseenter="onGifEnter(gif, $event)"
        @mousemove="onGifMove($event)"
        @mouseleave="onGifLeave"
        @click="selectGif(gif)"
      >
        <img :src="gif.previewUrl || gif.url" :alt="gif.title || 'gif'" loading="lazy" />
      </button>
    </div>

    <div v-if="loadingMore" class="gif-picker-more">載入更多...</div>
    <div v-else-if="!hasMore && gifs.length > 0" class="gif-picker-more gif-picker-more-end">沒有更多 GIF 了</div>

    <Teleport to="body">
      <div
        v-if="hoverGif"
        class="gif-hover-preview"
        :style="{ left: `${previewPos.x}px`, top: `${previewPos.y}px` }"
      >
        <img :src="hoverGif.url || hoverGif.previewUrl" :alt="hoverGif.title || 'GIF 預覽'" />
        <div class="gif-hover-preview-title">{{ hoverGif.title || 'GIF 預覽' }}</div>
      </div>
    </Teleport>

    <div class="gif-picker-footer">Powered by Tenor</div>
  </div>
</template>

<style scoped>
.gif-picker {
  width: min(92vw, 460px);
  max-height: min(76vh, 560px);
  background: var(--bg-modal, #2b2d31);
  border: 1px solid var(--border, #3f4147);
  border-radius: 12px;
  box-shadow: 0 10px 36px rgba(0, 0, 0, 0.45);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.gif-picker-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border, #3f4147);
}

.gif-picker-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary, #f2f3f5);
}

.gif-picker-close {
  background: none;
  color: var(--text-muted, #9ca3af);
  padding: 4px;
  border-radius: 6px;
}

.gif-picker-close:hover {
  color: var(--text-primary, #f2f3f5);
  background: var(--bg-hover, #363840);
}

.gif-picker-search-wrap {
  margin: 10px 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-input, #1f2125);
  border-radius: 8px;
  padding: 8px 10px;
  color: var(--text-muted, #a7a9ad);
}

.gif-picker-search {
  width: 100%;
  background: transparent;
  border: none;
  color: var(--text-primary, #f2f3f5);
  font-size: 13px;
}

.gif-picker-state {
  color: var(--text-muted, #a7a9ad);
  font-size: 13px;
  padding: 14px 12px;
}

.gif-picker-state-error {
  color: var(--danger, #f87171);
}

.gif-picker-grid {
  height: 360px;
  overflow-y: auto;
  padding: 2px 12px 6px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-auto-rows: 128px;
  gap: 12px;
  align-content: start;
}

.gif-card {
  background: #1e2024;
  border: 1px solid transparent;
  border-radius: 8px;
  overflow: hidden;
  padding: 0;
}

.gif-card:hover {
  border-color: var(--primary, #5865f2);
}

.gif-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.gif-picker-more {
  text-align: center;
  color: var(--text-muted, #a7a9ad);
  font-size: 12px;
  padding: 8px 12px 10px;
}

.gif-picker-more-end {
  opacity: 0.8;
}

.gif-hover-preview {
  position: fixed;
  width: min(48vw, 340px);
  max-height: min(48vh, 260px);
  z-index: 2200;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: #17181b;
  box-shadow: 0 16px 42px rgba(0, 0, 0, 0.5);
  pointer-events: none;
}

.gif-hover-preview img {
  width: 100%;
  height: calc(100% - 30px);
  max-height: 230px;
  object-fit: cover;
  display: block;
}

.gif-hover-preview-title {
  color: #e5e7eb;
  font-size: 12px;
  padding: 7px 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.gif-picker-footer {
  border-top: 1px solid var(--border, #3f4147);
  color: var(--text-muted, #a7a9ad);
  font-size: 11px;
  text-align: right;
  padding: 8px 12px;
}
</style>