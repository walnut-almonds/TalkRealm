<script setup>
import { inject } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import VoiceBar from './VoiceBar.vue'
import UserPanel from './UserPanel.vue'

const openModal = inject('openModal')
const voice     = inject('voice')

const store = useAppStore()
const voiceStore = useVoiceStore()

async function selectChannel(channel) {
  if (channel.type === 'voice') {
    await voice.joinVoiceChannel(channel.id)
    return
  }
  await store.selectChannel(channel.id)
}
</script>

<template>
  <div class="channels-sidebar">
    <!-- Guild header -->
    <div class="guild-header" v-if="store.currentGuild">
      <h2>{{ store.currentGuild.name }}</h2>
      <i class="fas fa-cog" title="社群設定" @click="openModal('guildSettings')"></i>
    </div>
    <div class="guild-header" v-else>
      <h2>選擇一個社群</h2>
    </div>

    <!-- Channels -->
    <div class="channels-container" v-if="store.currentGuild">
      <!-- Text channels -->
      <div class="channel-category">
        <div class="category-header">
          <i class="fas fa-chevron-down"></i>
          <span>文字頻道</span>
          <i class="fas fa-plus" title="建立文字頻道" @click.stop="openModal('createChannel', 'text')"></i>
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
          </div>
        </div>
      </div>

      <!-- Voice channels -->
      <div class="channel-category">
        <div class="category-header">
          <i class="fas fa-chevron-down"></i>
          <span>語音頻道</span>
          <i class="fas fa-plus" title="建立語音頻道" @click.stop="openModal('createChannel', 'voice')"></i>
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
            <i v-if="voiceStore.voiceChannel?.id === ch.id" class="fas fa-microphone voice-active-icon" title="語音連線中"></i>

            <!-- Voice participants under the channel -->
            <div v-if="voiceStore.getChannelParticipants(ch.id).length" class="voice-participants" @click.stop>
              <div
                v-for="p in voiceStore.getChannelParticipants(ch.id)"
                :key="p.user_id"
                class="voice-participant"
              >
                <div class="voice-participant-avatar-wrap">
                  <div :class="['voice-participant-avatar', { 'avatar-speaking': voiceStore.isSpeaking(p.user_id) }]">
                    <i class="fas fa-user"></i>
                  </div>
                </div>
                <span>{{ p.username }}</span>
                <div class="voice-participant-state">
                  <i
                    :class="['fas', voiceStore.getParticipantState(ch.id, p.user_id)?.mic_enabled !== false ? 'fa-microphone' : 'fa-microphone-slash',
                             voiceStore.getParticipantState(ch.id, p.user_id)?.mic_enabled !== false ? '' : 'muted']"
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
