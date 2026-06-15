<script setup>
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { api } from '@/api/index.js'
import { escapeHtml } from '@/utils/format.js'
import { getStatusText } from '@/utils/format.js'

const emit = defineEmits(['close-mobile'])
const store = useAppStore()
const voiceStore = useVoiceStore()
const { t } = useI18n()

const ROLE_LEVEL = { owner: 4, admin: 3, moderator: 2, member: 1 }
const ROLE_LABEL = {
  owner: () => t('memberSidebar.roleOwner'),
  admin: () => t('memberSidebar.roleAdmin'),
  moderator: () => t('memberSidebar.roleModerator'),
}

function myLevel() {
  return ROLE_LEVEL[store.currentUserRole] || 0
}

function canManage(member) {
  const ml = ROLE_LEVEL[member.role] || 0
  return myLevel() >= ROLE_LEVEL.admin && ml < myLevel() && store.user?.id !== member.user_id
}

async function kickMember(guildId, userId, username) {
  if (!confirm(t('memberSidebar.kickConfirm', { username }))) return
  try {
    await api.kickMember(guildId, userId)
    store.members = store.members.filter(m => m.user_id !== userId)
    store.showNotification(t('memberSidebar.kickSuccess', { username }), 'success')
  } catch (e) {
    store.showNotification(e.message || t('memberSidebar.kickFailed'), 'error')
  }
}

async function updateRole(guildId, userId, username, currentRole) {
  const newRole = currentRole === 'member' ? 'moderator' : 'member'
  const label = newRole === 'moderator' ? t('memberSidebar.roleModerator') : t('memberSidebar.roleMember')
  if (!confirm(t('memberSidebar.updateRoleConfirm', { username, label }))) return
  try {
    await api.updateMemberRole(guildId, userId, newRole)
    const m = store.members.find(m => m.user_id === userId)
    if (m) m.role = newRole
    store.showNotification(t('memberSidebar.updateRoleSuccess', { username }), 'success')
  } catch (e) {
    store.showNotification(e.message || t('memberSidebar.updateRoleFailed'), 'error')
  }
}
</script>

<template>
  <div class="members-sidebar">
    <div class="members-header">
      <h3>{{ t('memberSidebar.membersCount', { count: store.members.length }) }}</h3>
      <button class="members-close-btn" @click="emit('close-mobile')" :title="t('common.close')">
        <i class="fas fa-times"></i>
      </button>
    </div>
    <div class="members-list">
      <div
        v-for="member in store.members"
        :key="member.user_id"
        class="member-item"
      >
        <div class="member-avatar">
          <div :class="['member-avatar-inner', { 'avatar-speaking': voiceStore.isSpeaking(member.user_id) }]">
            <img v-if="member.user?.avatar" :src="member.user.avatar" :alt="member.user?.nickname" />
            <i v-else class="fas fa-user"></i>
          </div>
          <span :class="['status-indicator', member.user?.status || 'offline']"></span>
        </div>
        <div class="member-info">
          <div class="member-name">
            {{ member.user?.nickname || member.user?.username || t('common.unknown') }}
            <span v-if="ROLE_LABEL[member.role]" :class="['role-badge', `role-${member.role}`]">
              {{ ROLE_LABEL[member.role]() }}
            </span>
          </div>
          <div style="font-size:11px;color:var(--text-muted)">
            {{ getStatusText(member.user?.status) }}
            <span v-if="voiceStore.isSpeaking(member.user_id)" style="color:var(--success)">
              🔊 {{ t('memberSidebar.speaking') }}
            </span>
          </div>
        </div>
        <div v-if="canManage(member)" class="member-actions">
          <button
            class="btn-icon-sm"
            :title="t('memberSidebar.changeRole')"
            @click="updateRole(store.currentGuild.id, member.user_id, member.user?.nickname || member.user?.username, member.role)"
          >
            <i class="fas fa-user-shield"></i>
          </button>
          <button
            class="btn-icon-sm danger"
            :title="t('memberSidebar.kickFromGuild')"
            @click="kickMember(store.currentGuild.id, member.user_id, member.user?.nickname || member.user?.username)"
          >
            <i class="fas fa-user-times"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
