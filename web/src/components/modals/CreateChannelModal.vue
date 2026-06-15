<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()
const { t } = useI18n()

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
    store.showNotification(t('createChannel.createdSuccess'), 'success')
    if (channelType.value === 'text') {
      await store.selectChannel(ch)
    }
    emit('close')
  } catch (e) {
    store.showNotification(t('createChannel.createFailed') + (e.message || t('common.unknownError')), 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2>{{ t('createChannel.title') }}</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>{{ t('createChannel.channelType') }}</label>
          <div class="channel-type-selector">
            <label class="type-option" :class="{ active: channelType === 'text' }">
              <input type="radio" v-model="channelType" value="text" />
              <i class="fas fa-hashtag"></i> {{ t('createChannel.textChannel') }}
            </label>
            <label class="type-option" :class="{ active: channelType === 'voice' }">
              <input type="radio" v-model="channelType" value="voice" />
              <i class="fas fa-volume-up"></i> {{ t('createChannel.voiceChannel') }}
            </label>
          </div>
        </div>
        <div class="form-group">
          <label>{{ t('createChannel.channelNameRequired') }}</label>
          <input v-model="channelName" :placeholder="t('createChannel.channelNamePlaceholder')" maxlength="100" @keydown.enter="submit" />
        </div>
        <div class="form-group" v-if="channelType === 'text'">
          <label>{{ t('createChannel.channelTopicOptional') }}</label>
          <input v-model="channelTopic" :placeholder="t('createChannel.channelTopicPlaceholder')" maxlength="200" />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!channelName.trim() || loading" @click="submit">
          {{ loading ? t('createChannel.creating') : t('createChannel.createButton') }}
        </button>
      </div>
    </div>
  </div>
</template>
