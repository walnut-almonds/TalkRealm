<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useFileUpload } from '@/composables/useFileUpload.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['posted'])
const store = useAppStore()
const { t } = useI18n()

const text = ref('')
const fileIds = ref([])
const posting = ref(false)

const { pendingChips, uploadFile, removeChip, clearChips } = useFileUpload({
  addFileId: (id) => fileIds.value.push(id),
  removeFileId: (id) => {
    const i = fileIds.value.indexOf(id)
    if (i !== -1) fileIds.value.splice(i, 1)
  },
  showNotification: (m, ty) => store.showNotification(m, ty),
})

function onFileInput(e) {
  Array.from(e.target.files || []).forEach(f => uploadFile(f))
  e.target.value = ''
}

async function submit() {
  const content = text.value.trim()
  const ids = [...fileIds.value]
  if ((!content && ids.length === 0) || posting.value) return
  posting.value = true
  try {
    const post = await api.createFeedPost(content, ids)
    emit('posted', post)
    text.value = ''
    fileIds.value = []
    clearChips()
  } catch (e) {
    store.showNotification(e.message || 'Failed', 'error')
  } finally {
    posting.value = false
  }
}
</script>

<template>
  <div class="feed-compose">
    <textarea
      v-model="text"
      class="feed-compose-input"
      rows="2"
      :placeholder="t('feed.composePlaceholder')"
    ></textarea>
    <div v-if="pendingChips.length" class="feed-compose-chips">
      <div v-for="chip in pendingChips" :key="chip.id" class="file-chip">
        <span class="file-chip-name">{{ chip.name }}</span>
        <span v-if="!chip.done" class="file-chip-progress">{{ chip.progress }}%</span>
        <button v-else class="file-chip-remove" @click="removeChip(chip.id, chip.fileId)">
          <i class="fas fa-times"></i>
        </button>
      </div>
    </div>
    <div class="feed-compose-actions">
      <label class="feed-attach-btn" :title="t('dm.uploadFile')">
        <i class="fas fa-image"></i>
        <input type="file" multiple style="display:none" @change="onFileInput" />
      </label>
      <button
        class="btn-primary btn-sm"
        :disabled="posting || (!text.trim() && fileIds.length === 0)"
        @click="submit"
      >
        {{ posting ? t('feed.publishing') : t('feed.publish') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.feed-compose {
  background: var(--bg-secondary, #2f3136);
  border: 1px solid var(--border-color, #40444b);
  border-radius: var(--radius, 8px);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.feed-compose-input {
  width: 100%;
  background: var(--bg-input, #40444b);
  border: none;
  border-radius: var(--radius, 8px);
  color: var(--text-primary, #dcddde);
  font-size: 14px;
  padding: 10px 12px;
  resize: none;
  outline: none;
  font-family: inherit;
  line-height: 1.5;
}

.feed-compose-input::placeholder { color: var(--text-muted, #8e9297); }

.feed-compose-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: flex-end;
}

.feed-attach-btn {
  color: var(--text-muted, #8e9297);
  cursor: pointer;
  font-size: 18px;
  margin-right: auto;
  display: flex;
  align-items: center;
  transition: color 0.1s;
}

.feed-attach-btn:hover { color: var(--text-primary, #dcddde); }

.feed-compose-chips { display: flex; flex-wrap: wrap; gap: 6px; }

.file-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-input, #40444b);
  border-radius: var(--radius, 8px);
  padding: 4px 10px;
  font-size: 12px;
  color: var(--text-normal, #dcddde);
}

.file-chip-name { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-chip-progress { color: var(--text-muted, #8e9297); }
.file-chip-remove { background: none; border: none; color: var(--text-muted, #8e9297); cursor: pointer; padding: 0; line-height: 1; }
.file-chip-remove:hover { color: var(--text-primary, #dcddde); }
</style>
