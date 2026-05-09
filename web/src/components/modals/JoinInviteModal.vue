<script setup>
import { ref } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()

const inviteCode = ref('')
const loading = ref(false)

async function joinGuild() {
  const code = inviteCode.value.trim()
  if (!code) return
  loading.value = true
  try {
    const guild = await api.joinByInvite(code)
    await store.loadGuilds()
    if (guild?.id) await store.selectGuild(guild.id)
    store.showNotification('成功加入社群！', 'success')
    emit('close')
  } catch (e) {
    store.showNotification('加入失敗：' + (e.message || '邀請碼無效或已過期'), 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>加入社群</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <p style="color:var(--text-muted);margin-bottom:16px">輸入邀請碼以加入一個已存在的社群。</p>
        <div class="form-group">
          <label>邀請碼</label>
          <input
            v-model="inviteCode"
            placeholder="輸入邀請碼..."
            @keydown.enter="joinGuild"
            autocomplete="off"
          />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="!inviteCode.trim() || loading" @click="joinGuild">
          {{ loading ? '加入中...' : '加入社群' }}
        </button>
      </div>
    </div>
  </div>
</template>
