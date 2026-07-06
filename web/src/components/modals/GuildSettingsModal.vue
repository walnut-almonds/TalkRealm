<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()
const { t } = useI18n()

const activeTab = ref('general')
const guildName = ref('')
const guildDescription = ref('')
const inviteCode = ref('')
const inviteExpires = ref('7d')
const saving = ref(false)
const deleting = ref(false)
const deleteConfirm = ref('')

const guild = computed(() => store.currentGuild)
const isOwner = computed(() => store.currentUserRole === 'owner')

onMounted(() => {
  if (guild.value) {
    guildName.value = guild.value.name || ''
    guildDescription.value = guild.value.description || ''
  }
})

async function saveGeneral() {
  if (!guildName.value.trim()) return
  saving.value = true
  try {
    await api.updateGuild(guild.value.id, { name: guildName.value.trim(), description: guildDescription.value.trim() })
    store.showNotification(t('guildSettings.saved'), 'success')
    await store.loadGuilds()
    emit('close')
  } catch (e) {
    store.showNotification(t('guildSettings.saveFailed') + (e.message || t('common.unknownError')), 'error')
  } finally {
    saving.value = false
  }
}

async function generateInvite() {
  try {
    const expiresHours = inviteExpires.value === '1d' ? 24 : inviteExpires.value === '7d' ? 168 : inviteExpires.value === '30d' ? 720 : 0
    const result = await api.createInvite(guild.value.id, { expires_hours: expiresHours })
    inviteCode.value = result.code || result.invite_code || ''
    store.showNotification(t('guildSettings.inviteGenerated'), 'success')
  } catch (e) {
    store.showNotification(t('guildSettings.generateFailed') + (e.message || t('common.unknownError')), 'error')
  }
}

function copyInvite() {
  if (!inviteCode.value) return
  navigator.clipboard.writeText(inviteCode.value).then(() => store.showNotification(t('guildSettings.inviteCopied'), 'success'))
}

async function deleteGuild() {
  if (deleteConfirm.value !== guild.value.name) {
    store.showNotification(t('guildSettings.deleteConfirmMismatch'), 'error')
    return
  }
  deleting.value = true
  try {
    await api.deleteGuild(guild.value.id)
    store.showNotification(t('guildSettings.deleted'), 'success')
    store.currentGuild = null
    store.currentChannel = null
    await store.loadGuilds()
    emit('close')
  } catch (e) {
    store.showNotification(t('guildSettings.deleteFailed') + (e.message || t('common.unknownError')), 'error')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal modal-large">
      <div class="modal-header">
        <h2>{{ t('guildSettings.title') }}</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-tabs">
        <button :class="['modal-tab', { active: activeTab === 'general' }]" @click="activeTab = 'general'">{{ t('guildSettings.tabGeneral') }}</button>
        <button :class="['modal-tab', { active: activeTab === 'invite' }]" @click="activeTab = 'invite'">{{ t('guildSettings.tabInvite') }}</button>
        <button v-if="isOwner" :class="['modal-tab', { active: activeTab === 'danger' }]" @click="activeTab = 'danger'" style="color:var(--danger)">{{ t('guildSettings.tabDanger') }}</button>
      </div>
      <div class="modal-body">
        <!-- General -->
        <template v-if="activeTab === 'general'">
          <div class="form-group">
            <label>{{ t('guildSettings.guildNameRequired') }}</label>
            <input v-model="guildName" maxlength="100" />
          </div>
          <div class="form-group">
            <label>{{ t('guildSettings.description') }}</label>
            <textarea v-model="guildDescription" rows="3" maxlength="500"></textarea>
          </div>
        </template>

        <!-- Invite -->
        <template v-if="activeTab === 'invite'">
          <div class="form-group">
            <label>{{ t('guildSettings.inviteExpiry') }}</label>
            <select v-model="inviteExpires">
              <option value="1d">{{ t('guildSettings.expire1d') }}</option>
              <option value="7d">{{ t('guildSettings.expire7d') }}</option>
              <option value="30d">{{ t('guildSettings.expire30d') }}</option>
              <option value="never">{{ t('guildSettings.expireNever') }}</option>
            </select>
          </div>
          <button class="btn btn-primary" @click="generateInvite">{{ t('guildSettings.generateInvite') }}</button>
          <div v-if="inviteCode" class="invite-code-box">
            <code>{{ inviteCode }}</code>
            <button class="btn btn-secondary btn-sm" @click="copyInvite">{{ t('common.copy') }}</button>
          </div>
        </template>

        <!-- Danger -->
        <template v-if="activeTab === 'danger'">
          <div class="danger-zone">
            <p class="danger-warning">{{ t('guildSettings.dangerWarning') }}</p>
            <div class="form-group">
              <label>{{ t('guildSettings.typeNameToDelete') }} <strong>{{ guild?.name }}</strong> {{ t('guildSettings.toConfirm') }}</label>
              <input v-model="deleteConfirm" :placeholder="t('guildSettings.deleteInputPlaceholder')" />
            </div>
            <button class="btn btn-danger" :disabled="deleteConfirm !== guild?.name || deleting" @click="deleteGuild">
              {{ deleting ? t('guildSettings.deleting') : t('guildSettings.deletePermanently') }}
            </button>
          </div>
        </template>
      </div>
      <div class="modal-footer" v-if="activeTab === 'general'">
        <button class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!guildName.trim() || saving" @click="saveGeneral">
          {{ saving ? t('guildSettings.saving') : t('guildSettings.saveSettings') }}
        </button>
      </div>
    </div>
  </div>
</template>
