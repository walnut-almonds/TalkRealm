<script setup>
import { ref } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()

const channelName = ref('')
const channelTopic = ref('')
const channelType = ref('text')
const loading = ref(false)

async function submit() {
  if (!channelName.value.trim()) return
  if (!store.currentGuild) return
  loading.value = true
  try {
    const ch = await api.createChannel(store.currentGuild.id, {
      name: channelName.value.trim(),
      topic: channelTopic.value.trim(),
      type: channelType.value,
    })
    store.showNotification('頻道建立成功！', 'success')
    if (channelType.value === 'text') {
      await store.selectChannel(ch)
    }
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
        <h2>建立頻道</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>頻道類型</label>
          <div class="channel-type-selector">
            <label class="type-option" :class="{ active: channelType === 'text' }">
              <input type="radio" v-model="channelType" value="text" />
              <i class="fas fa-hashtag"></i> 文字頻道
            </label>
            <label class="type-option" :class="{ active: channelType === 'voice' }">
              <input type="radio" v-model="channelType" value="voice" />
              <i class="fas fa-volume-up"></i> 語音頻道
            </label>
          </div>
        </div>
        <div class="form-group">
          <label>頻道名稱 *</label>
          <input v-model="channelName" placeholder="新頻道" maxlength="100" @keydown.enter="submit" />
        </div>
        <div class="form-group" v-if="channelType === 'text'">
          <label>頻道主題（可選）</label>
          <input v-model="channelTopic" placeholder="這個頻道的主題..." maxlength="200" />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="!channelName.trim() || loading" @click="submit">
          {{ loading ? '建立中...' : '建立頻道' }}
        </button>
      </div>
    </div>
  </div>
</template>
