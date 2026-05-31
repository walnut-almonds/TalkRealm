<script setup>
import { ref, onMounted } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()

const displayName = ref('')
const avatarUrl = ref('')
const status = ref('online')
const preferredLang = ref('zh')
const oldPassword = ref('')
const newPassword = ref('')
const saving = ref(false)
const gifProvider = ref('auto')
const gifApiKey = ref('')
const gifClientKey = ref('talkrealm-web')

onMounted(() => {
  const user = store.user
  if (user) {
    displayName.value = user.display_name || user.username || ''
    avatarUrl.value = user.avatar_url || ''
    status.value = user.status || 'online'
    preferredLang.value = user.preferred_lang || 'zh'
  }

  const gifConfig = api.getGIFConfig()
  gifProvider.value = gifConfig.provider || 'auto'
  gifApiKey.value = gifConfig.apiKey || ''
  gifClientKey.value = gifConfig.clientKey || 'talkrealm-web'
})

async function saveProfile() {
  saving.value = true
  try {
    await api.updateProfile({
      display_name: displayName.value.trim(),
      avatar_url: avatarUrl.value.trim(),
      status: status.value,
      preferred_lang: preferredLang.value,
    })
    // update local user
    if (store.user) {
      store.user.display_name = displayName.value.trim()
      store.user.avatar_url = avatarUrl.value.trim()
      store.user.status = status.value
      store.user.preferred_lang = preferredLang.value
    }

    api.setGIFConfig({
      provider: gifProvider.value,
      apiKey: gifApiKey.value,
      clientKey: gifClientKey.value,
    })

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
          <div class="form-group">
            <label>翻譯語言</label>
            <select v-model="preferredLang">
              <option value="zh">🇨🇳 中文（簡體）</option>
              <option value="zh-tw">🇹🇼 繁體中文</option>
              <option value="ja">🇯🇵 日本語</option>
              <option value="en">🇺🇸 English</option>
            </select>
          </div>
        </div>

        <hr style="border-color:var(--border-color);margin:16px 0" />

        <div class="settings-section">
          <h3>GIF 搜尋設定</h3>
          <div class="form-group">
            <label>Provider</label>
            <select v-model="gifProvider">
              <option value="auto">Auto（優先 Tenor v2，失敗 fallback v1）</option>
              <option value="tenor-v2">Tenor v2（需 API Key）</option>
              <option value="tenor-v1">Tenor v1（相容模式）</option>
            </select>
          </div>
          <div class="form-group">
            <label>Tenor API Key（v2）</label>
            <input v-model="gifApiKey" placeholder="未填寫時 Auto 會走 v1 相容模式" />
          </div>
          <div class="form-group">
            <label>Tenor Client Key</label>
            <input v-model="gifClientKey" placeholder="talkrealm-web" />
          </div>
          <div class="form-hint">這些設定會儲存在你的瀏覽器本地端，不會上傳到伺服器。</div>
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
