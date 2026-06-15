<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()
const { t } = useI18n()

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
    store.showNotification(t('createGuild.createdSuccess'), 'success')
    emit('close')
  } catch (e) {
    store.showNotification(t('createGuild.createFailed') + (e.message || t('common.unknownError')), 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>{{ t('createGuild.title') }}</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>{{ t('createGuild.nameRequired') }}</label>
          <input v-model="name" :placeholder="t('createGuild.namePlaceholder')" maxlength="100" @keydown.enter="submit" />
        </div>
        <div class="form-group">
          <label>{{ t('createGuild.descriptionOptional') }}</label>
          <textarea v-model="description" :placeholder="t('createGuild.descriptionPlaceholder')" rows="3" maxlength="500"></textarea>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!name.trim() || loading" @click="submit">
          {{ loading ? t('createGuild.creating') : t('createGuild.createButton') }}
        </button>
      </div>
    </div>
  </div>
</template>
