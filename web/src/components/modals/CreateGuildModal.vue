<script setup>
import { ref } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()

const name = ref('')
const description = ref('')
const loading = ref(false)

async function submit() {
  if (!name.value.trim()) return
  loading.value = true
  try {
    const guild = await api.createGuild({ name: name.value.trim(), description: description.value.trim() })
    await store.loadGuilds()
    await store.selectGuild(guild.id)
    store.showNotification('社群建立成功！', 'success')
    emit('close')
  } catch (e) {
    store.showNotification('建立失敗：' + (e.message || '未知錯誤'), 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>建立新社群</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>社群名稱 *</label>
          <input v-model="name" placeholder="我的社群" maxlength="100" @keydown.enter="submit" />
        </div>
        <div class="form-group">
          <label>描述（可選）</label>
          <textarea v-model="description" placeholder="關於這個社群的簡短描述..." rows="3" maxlength="500"></textarea>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="!name.trim() || loading" @click="submit">
          {{ loading ? '建立中...' : '建立社群' }}
        </button>
      </div>
    </div>
  </div>
</template>
