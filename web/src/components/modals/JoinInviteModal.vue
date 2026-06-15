<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()
const { t } = useI18n()

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
    store.showNotification(t('joinInvite.joinSuccess'), 'success')
    emit('close')
  } catch (e) {
    store.showNotification(t('joinInvite.joinFailed') + (e.message || t('joinInvite.invalidOrExpired')), 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>{{ t('joinInvite.title') }}</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <p style="color:var(--text-muted);margin-bottom:16px">{{ t('joinInvite.description') }}</p>
        <div class="form-group">
          <label>{{ t('joinInvite.inviteCode') }}</label>
          <input
            v-model="inviteCode"
            :placeholder="t('joinInvite.inputPlaceholder')"
            @keydown.enter="joinGuild"
            autocomplete="off"
          />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!inviteCode.trim() || loading" @click="joinGuild">
          {{ loading ? t('joinInvite.joining') : t('joinInvite.joinButton') }}
        </button>
      </div>
    </div>
  </div>
</template>
