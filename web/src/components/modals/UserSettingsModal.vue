<script setup>
import { ref, onMounted } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'
import { STORAGE_KEYS } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()

const displayName = ref('')
const avatarUrl = ref('')
const status = ref('online')
const oldPassword = ref('')
const newPassword = ref('')
const saving = ref(false)

onMounted(() => {
  const user = store.currentUser
  if (user) {
    displayName.value = user.display_name || user.username || ''
    avatarUrl.value = user.avatar_url || ''
    status.value = user.status || 'online'
  }
})

async function saveProfile() {
  saving.value = true
  try {
    await api.updateProfile({ display_name: displayName.value.trim(), avatar_url: avatarUrl.value.trim(), status: status.value })
    // update local user
    if (store.currentUser) {
      store.currentUser.display_name = displayName.value.trim()
      store.currentUser.avatar_url = avatarUrl.value.trim()
      store.currentUser.status = status.value
    }
    store.showNotification('個人資料已更新', 'success')
    emit('close')
  } catch (e) {
    store.showNotification('更新失敗：' + (e.message || '未知錯誤'), 'error')
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  if (!oldPassword.value || !newPassword.value) {
    store.showNotification('請填寫舊密碼和新密碼', 'error')
    return
  }
  saving.value = true
  try {
    await api.changePassword({ old_password: oldPassword.value, new_password: newPassword.value })
    oldPassword.value = ''
    newPassword.value = ''
    store.showNotification('密碼已更新', 'success')
  } catch (e) {
    store.showNotification('更改密碼失敗：' + (e.message || '未知錯誤'), 'error')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>使用者設定</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="settings-section">
          <h3>個人資料</h3>
          <div class="form-group">
            <label>顯示名稱</label>
            <input v-model="displayName" maxlength="50" placeholder="顯示名稱" />
          </div>
          <div class="form-group">
            <label>頭像網址</label>
            <input v-model="avatarUrl" type="url" placeholder="https://..." />
          </div>
          <div class="form-group">
            <label>狀態</label>
            <select v-model="status">
              <option value="online">🟢 線上</option>
              <option value="idle">🌙 閒置</option>
              <option value="dnd">🔴 請勿打擾</option>
              <option value="invisible">⚫ 隱身</option>
            </select>
          </div>
        </div>

        <hr style="border-color:var(--border-color);margin:16px 0" />

        <div class="settings-section">
          <h3>更改密碼</h3>
          <div class="form-group">
            <label>舊密碼</label>
            <input v-model="oldPassword" type="password" placeholder="目前的密碼" />
          </div>
          <div class="form-group">
            <label>新密碼</label>
            <input v-model="newPassword" type="password" placeholder="新密碼" />
          </div>
          <button class="btn btn-secondary" :disabled="saving" @click="changePassword">更改密碼</button>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="saving" @click="saveProfile">
          {{ saving ? '儲存中...' : '儲存設定' }}
        </button>
      </div>
    </div>
  </div>
</template>
