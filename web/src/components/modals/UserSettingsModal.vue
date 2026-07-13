<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'
import { getLocale, setLocale } from '@/i18n/index.js'
import { useLearnStore } from '@/stores/useLearnStore.js'

const emit = defineEmits(['close'])
const store = useAppStore()
const learn = useLearnStore()
const { t } = useI18n()

const displayName = ref('')
const avatarUrl = ref('')
const status = ref('online')
const preferredLang = ref('zh')
const uiLocale = ref('zh')
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
    uiLocale.value = user.ui_locale || getLocale()
  } else {
    uiLocale.value = getLocale()
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
      ui_locale: uiLocale.value,
    })
    // update local user
    if (store.user) {
      store.user.display_name = displayName.value.trim()
      store.user.avatar_url = avatarUrl.value.trim()
      store.user.status = status.value
      store.user.preferred_lang = preferredLang.value
      store.user.ui_locale = uiLocale.value
    }

    setLocale(uiLocale.value)

    api.setGIFConfig({
      provider: gifProvider.value,
      apiKey: gifApiKey.value,
      clientKey: gifClientKey.value,
    })

    store.showNotification(t('settings.profileSaved'), 'success')
    emit('close')
  } catch (e) {
    store.showNotification(`${t('settings.profileSaveFailed')}: ${e.message || 'Unknown error'}`, 'error')
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  if (!oldPassword.value || !newPassword.value) {
    store.showNotification(t('settings.passwordRequired'), 'error')
    return
  }
  saving.value = true
  try {
    await api.changePassword({ old_password: oldPassword.value, new_password: newPassword.value })
    oldPassword.value = ''
    newPassword.value = ''
    store.showNotification(t('settings.passwordSaved'), 'success')
  } catch (e) {
    store.showNotification(`${t('settings.passwordSaveFailed')}: ${e.message || 'Unknown error'}`, 'error')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>{{ t('settings.title') }}</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="settings-section">
          <h3>{{ t('settings.profile') }}</h3>
          <div class="form-group">
            <label>{{ t('settings.displayName') }}</label>
            <input v-model="displayName" maxlength="50" :placeholder="t('settings.displayNamePlaceholder')" />
          </div>
          <div class="form-group">
            <label>{{ t('settings.avatarUrl') }}</label>
            <input v-model="avatarUrl" type="url" placeholder="https://..." />
          </div>
          <div class="form-group">
            <label>{{ t('settings.status') }}</label>
            <select v-model="status">
              <option value="online">🟢 {{ t('settings.statusOnline') }}</option>
              <option value="idle">🌙 {{ t('settings.statusIdle') }}</option>
              <option value="dnd">🔴 {{ t('settings.statusDnd') }}</option>
              <option value="invisible">⚫ {{ t('settings.statusInvisible') }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>{{ t('settings.translationLanguage') }}</label>
            <select v-model="preferredLang">
              <option value="zh">🇨🇳 {{ t('settings.langZh') }}</option>
              <option value="zh-tw">🇹🇼 {{ t('settings.langZhTw') }}</option>
              <option value="ja">🇯🇵 {{ t('settings.langJa') }}</option>
              <option value="en">🇺🇸 {{ t('settings.langEn') }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>{{ t('settings.interfaceLanguage') }}</label>
            <select v-model="uiLocale">
              <option value="zh">🇨🇳 {{ t('settings.langZh') }}</option>
              <option value="zh-tw">🇹🇼 {{ t('settings.langZhTw') }}</option>
              <option value="ja">🇯🇵 {{ t('settings.langJa') }}</option>
              <option value="en">🇺🇸 {{ t('settings.langEn') }}</option>
            </select>
          </div>
        </div>

        <hr style="border-color:var(--border-color);margin:16px 0" />

        <div class="settings-section">
          <h3>{{ t('settings.gifSearch') }}</h3>
          <div class="form-group">
            <label>{{ t('settings.provider') }}</label>
            <select v-model="gifProvider">
              <option value="auto">{{ t('settings.gifProviderAuto') }}</option>
              <option value="tenor-v2">{{ t('settings.gifProviderV2') }}</option>
              <option value="tenor-v1">{{ t('settings.gifProviderV1') }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>{{ t('settings.gifApiKey') }}</label>
            <input v-model="gifApiKey" :placeholder="t('settings.gifApiKeyPlaceholder')" />
          </div>
          <div class="form-group">
            <label>{{ t('settings.gifClientKey') }}</label>
            <input v-model="gifClientKey" placeholder="talkrealm-web" />
          </div>
          <div class="form-hint">{{ t('settings.gifHint') }}</div>
        </div>

        <hr style="border-color:var(--border-color);margin:16px 0" />

        <div class="settings-section">
          <h3>{{ t('settings.learn') }}</h3>
          <label class="form-group" style="flex-direction:row;align-items:center;gap:8px;cursor:pointer">
            <input
              type="checkbox"
              :checked="learn.hardMode"
              style="width:auto"
              @change="learn.setHardMode($event.target.checked)"
            />
            <span>{{ t('learn.hardMode') }}</span>
          </label>
          <div class="form-hint">{{ t('learn.hardModeDesc') }}</div>
        </div>

        <hr style="border-color:var(--border-color);margin:16px 0" />

        <div class="settings-section">
          <h3>{{ t('settings.changePassword') }}</h3>
          <div class="form-group">
            <label>{{ t('settings.oldPassword') }}</label>
            <input v-model="oldPassword" type="password" :placeholder="t('settings.oldPasswordPlaceholder')" />
          </div>
          <div class="form-group">
            <label>{{ t('settings.newPassword') }}</label>
            <input v-model="newPassword" type="password" :placeholder="t('settings.newPasswordPlaceholder')" />
          </div>
          <button class="btn btn-secondary" :disabled="saving" @click="changePassword">{{ t('settings.changePasswordAction') }}</button>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="saving" @click="saveProfile">
          {{ saving ? t('common.saving') : t('settings.saveSettings') }}
        </button>
      </div>
    </div>
  </div>
</template>
