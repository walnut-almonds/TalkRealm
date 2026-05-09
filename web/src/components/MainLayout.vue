<script setup>
import { ref } from 'vue'
import { provide } from 'vue'
import NavRail from './NavRail.vue'
import CreateGuildModal from './modals/CreateGuildModal.vue'
import CreateChannelModal from './modals/CreateChannelModal.vue'
import GuildSettingsModal from './modals/GuildSettingsModal.vue'
import UserSettingsModal from './modals/UserSettingsModal.vue'
import JoinInviteModal from './modals/JoinInviteModal.vue'

const props = defineProps({ voice: Object })

const createChannelType = ref('text')
const modals = ref({
  createGuild: false,
  createChannel: false,
  guildSettings: false,
  userSettings: false,
  joinInvite: false,
})

function openCreateChannel(type = 'text') {
  createChannelType.value = type
  modals.value.createChannel = true
}

// Expose modal opener and voice composable to all child views via provide
provide('openModal', (name, payload) => {
  if (name === 'createChannel') return openCreateChannel(payload)
  modals.value[name] = true
})
provide('voice', props.voice)
</script>

<template>
  <div class="app-shell">
    <NavRail
      @create-guild="modals.createGuild = true"
      @join-guild="modals.joinInvite = true"
    />
    <!-- Routed view fills the rest -->
    <div class="app-view">
      <RouterView />
    </div>
  </div>

  <!-- Modals (global, shared across all views) -->
  <CreateGuildModal  v-if="modals.createGuild"   @close="modals.createGuild = false" />
  <CreateChannelModal v-if="modals.createChannel" :type="createChannelType" @close="modals.createChannel = false" />
  <GuildSettingsModal v-if="modals.guildSettings" @close="modals.guildSettings = false" />
  <UserSettingsModal  v-if="modals.userSettings"  @close="modals.userSettings = false" />
  <JoinInviteModal    v-if="modals.joinInvite"    @close="modals.joinInvite = false" />
</template>
