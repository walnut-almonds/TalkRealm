<script setup>
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { api } from '@/api/index.js'

const emit = defineEmits(['select', 'close'])

const rootEl = ref(null)
const query = ref('')
const gifs = ref([])
const loading = ref(false)
const error = ref('')

let searchTimer = null

async function loadGifs(keyword = '') {
  loading.value = true
  error.value = ''
  try {
    gifs.value = await api.searchGIFs(keyword, 18)
  } catch (e) {
    error.value = e?.message || 'GIF 服務暫時無法使用'
  } finally {
    loading.value = false
  }
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

watch(query, (val) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
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

    <div v-else class="gif-picker-grid">
      <button
        v-for="gif in gifs"
        :key="gif.id"
        class="gif-card"
        :title="gif.title || 'GIF'"
        @click="selectGif(gif)"
      >
        <img :src="gif.previewUrl || gif.url" :alt="gif.title || 'gif'" loading="lazy" />
      </button>
    </div>

    <div class="gif-picker-footer">Powered by Tenor</div>
  </div>
</template>

<style scoped>
.gif-picker {
  width: min(92vw, 420px);
  max-height: min(74vh, 520px);
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
  flex: 1;
  min-height: 180px;
  overflow-y: auto;
  padding: 0 12px 10px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.gif-card {
  background: #1e2024;
  border: 1px solid transparent;
  border-radius: 8px;
  overflow: hidden;
  padding: 0;
  aspect-ratio: 1 / 1;
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

.gif-picker-footer {
  border-top: 1px solid var(--border, #3f4147);
  color: var(--text-muted, #a7a9ad);
  font-size: 11px;
  text-align: right;
  padding: 8px 12px;
}
</style>