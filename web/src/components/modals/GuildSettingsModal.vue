<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { api } from '@/api/index.js'

const emit = defineEmits(['close'])
const store = useAppStore()

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
    store.showNotification('設定已儲存', 'success')
    await store.loadGuilds()
    emit('close')
  } catch (e) {
    store.showNotification('儲存失敗：' + (e.message || '未知錯誤'), 'error')
  } finally {
    saving.value = false
  }
}

async function generateInvite() {
  try {
    const expiresHours = inviteExpires.value === '1d' ? 24 : inviteExpires.value === '7d' ? 168 : inviteExpires.value === '30d' ? 720 : 0
    const result = await api.createInvite(guild.value.id, { expires_hours: expiresHours })
    inviteCode.value = result.code || result.invite_code || ''
    store.showNotification('邀請碼已生成！', 'success')
  } catch (e) {
    store.showNotification('生成失敗：' + (e.message || '未知錯誤'), 'error')
  }
}

function copyInvite() {
  if (!inviteCode.value) return
  navigator.clipboard.writeText(inviteCode.value).then(() => store.showNotification('邀請碼已複製！', 'success'))
}

async function deleteGuild() {
  if (deleteConfirm.value !== guild.value.name) {
    store.showNotification('請輸入正確的社群名稱以確認刪除', 'error')
    return
  }
  deleting.value = true
  try {
    await api.deleteGuild(guild.value.id)
    store.showNotification('社群已刪除', 'success')
    store.currentGuild = null
    store.currentChannel = null
    await store.loadGuilds()
    emit('close')
  } catch (e) {
    store.showNotification('刪除失敗：' + (e.message || '未知錯誤'), 'error')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal modal-large">
      <div class="modal-header">
        <h2>社群設定</h2>
        <button class="modal-close" @click="emit('close')">×</button>
      </div>
      <div class="modal-tabs">
        <button :class="['modal-tab', { active: activeTab === 'general' }]" @click="activeTab = 'general'">一般</button>
        <button :class="['modal-tab', { active: activeTab === 'invite' }]" @click="activeTab = 'invite'">邀請</button>
        <button v-if="isOwner" :class="['modal-tab', { active: activeTab === 'danger' }]" @click="activeTab = 'danger'" style="color:#f04747">危險操作</button>
      </div>
      <div class="modal-body">
        <!-- General -->
        <template v-if="activeTab === 'general'">
          <div class="form-group">
            <label>社群名稱 *</label>
            <input v-model="guildName" maxlength="100" />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="guildDescription" rows="3" maxlength="500"></textarea>
          </div>
        </template>

        <!-- Invite -->
        <template v-if="activeTab === 'invite'">
          <div class="form-group">
            <label>邀請有效期限</label>
            <select v-model="inviteExpires">
              <option value="1d">1 天</option>
              <option value="7d">7 天</option>
              <option value="30d">30 天</option>
              <option value="never">永不過期</option>
            </select>
          </div>
          <button class="btn btn-primary" @click="generateInvite">生成邀請碼</button>
          <div v-if="inviteCode" class="invite-code-box">
            <code>{{ inviteCode }}</code>
            <button class="btn btn-secondary btn-sm" @click="copyInvite">複製</button>
          </div>
        </template>

        <!-- Danger -->
        <template v-if="activeTab === 'danger'">
          <div class="danger-zone">
            <p class="danger-warning">⚠️ 刪除社群後無法復原！所有頻道和訊息都將永久消失。</p>
            <div class="form-group">
              <label>請輸入「<strong>{{ guild?.name }}</strong>」以確認刪除</label>
              <input v-model="deleteConfirm" placeholder="輸入社群名稱..." />
            </div>
            <button class="btn btn-danger" :disabled="deleteConfirm !== guild?.name || deleting" @click="deleteGuild">
              {{ deleting ? '刪除中...' : '永久刪除社群' }}
            </button>
          </div>
        </template>
      </div>
      <div class="modal-footer" v-if="activeTab === 'general'">
        <button class="btn btn-secondary" @click="emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="!guildName.trim() || saving" @click="saveGeneral">
          {{ saving ? '儲存中...' : '儲存設定' }}
        </button>
      </div>
    </div>
  </div>
</template>
