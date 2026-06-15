<script setup>
import { inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import VoiceBar from './VoiceBar.vue'
import UserPanel from './UserPanel.vue'

const emit = defineEmits(['channel-selected'])
const openModal = inject('openModal')
const voice     = inject('voice')

const store = useAppStore()
const voiceStore = useVoiceStore()
const { t } = useI18n()

async function selectChannel(channel) {
  if (channel.type === 'voice') {
    await voice.joinVoiceChannel(channel.id)
  } else {
    await store.selectChannel(channel.id)
  }
  emit('channel-selected')
}
</script>

<template>
  <div class="channels-sidebar">
    <!-- Guild header -->
    <div class="guild-header" v-if="store.currentGuild">
      <h2>{{ store.currentGuild.name }}</h2>
      <i class="fas fa-cog" :title="t('channelSidebar.guildSettings')" @click="openModal('guildSettings')"></i>
    </div>
    <div class="guild-header" v-else>
      <h2>{{ t('channelSidebar.selectGuild') }}</h2>
    </div>

    <!-- Channels -->
    <div class="channels-container" v-if="store.currentGuild">
      <!-- Text channels -->
      <div class="channel-category">
        <div class="category-header">
          <i class="fas fa-chevron-down"></i>
          <span>{{ t('channelSidebar.textChannels') }}</span>
          <i class="fas fa-plus" :title="t('channelSidebar.createTextChannel')" @click.stop="openModal('createChannel', 'text')"></i>
        </div>
        <div class="channels-list">
          <div
            v-for="ch in store.textChannels"
            :key="ch.id"
            :class="['channel-item', { active: store.currentChannel?.id === ch.id }]"
            @click="selectChannel(ch)"
          >
            <i class="fas fa-hashtag"></i>
            <span>{{ ch.name }}</span>
            <!-- Unread badge: mention (red number) or unread dot (grey) -->
            <template v-if="store.currentChannel?.id !== ch.id">
              <span
                v-if="(store.channelUnreadMap.get(ch.id)?.mention ?? 0) > 0"
                class="badge-mention"
              >{{ store.channelUnreadMap.get(ch.id).mention }}</span>
              <span
                v-else-if="(store.channelUnreadMap.get(ch.id)?.unread ?? 0) > 0"
                class="badge-unread"
              ></span>
            </template>
          </div>
        </div>
      </div>

      <!-- Voice channels -->
      <div class="channel-category">
        <div class="category-header">
          <i class="fas fa-chevron-down"></i>
          <span>{{ t('channelSidebar.voiceChannels') }}</span>
          <i class="fas fa-plus" :title="t('channelSidebar.createVoiceChannel')" @click.stop="openModal('createChannel', 'voice')"></i>
        </div>
        <div class="channels-list">
          <div
            v-for="ch in store.voiceChannels"
            :key="ch.id"
            :class="['channel-item', { active: voiceStore.voiceChannel?.id === ch.id }]"
            @click="selectChannel(ch)"
          >
            <i class="fas fa-volume-up"></i>
            <span>{{ ch.name }}</span>
            <i v-if="voiceStore.voiceChannel?.id === ch.id" class="fas fa-microphone voice-active-icon" :title="t('voice.connected')"></i>

            <!-- Voice participants under the channel -->
            <div v-if="voiceStore.getChannelParticipants(ch.id).length" class="voice-participants" @click.stop>
              <div
                v-for="p in voiceStore.getChannelParticipants(ch.id)"
                :key="p.user_id"
                class="voice-participant"
              >
                <div :class="['voice-participant-avatar', { 'avatar-speaking': voiceStore.isSpeaking(p.user_id) }]">
                  <i class="fas fa-user"></i>
                </div>
                <span class="voice-participant-name">{{ p.username }}</span>
                <div class="voice-participant-state">
                  <i
                    v-if="voiceStore.getParticipantState(ch.id, p.user_id)?.mic_enabled === false"
                    class="fas fa-microphone-slash"
                    style="color:var(--danger)"
                    :title="t('voice.muted')"
                  ></i>
                  <i
                    v-if="voiceStore.getParticipantState(ch.id, p.user_id)?.deafened"
                    class="fas fa-volume-xmark"
                    style="color:var(--danger)"
                    :title="t('voice.deafened')"
                  ></i>
                  <i
                    v-if="voiceStore.getParticipantState(ch.id, p.user_id)?.screen_sharing"
                    class="fas fa-display"
                    style="color:var(--brand)"
                    :title="t('voice.screenSharing')"
                  ></i>
                  <i
                    v-if="voiceStore.getParticipantState(ch.id, p.user_id)?.camera_enabled"
                    class="fas fa-video"
                    style="color:var(--brand)"
                    :title="t('voice.cameraOn')"
                  ></i>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Voice bar -->
    <VoiceBar v-if="voiceStore.voiceChannel" />

    <!-- User panel -->
    <UserPanel @settings="openModal('userSettings')" />
  </div>
</template>
