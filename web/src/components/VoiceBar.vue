<script setup>
import { inject } from 'vue'
import { useVoiceStore } from '@/stores/useVoiceStore.js'

const voice = inject('voice')
const voiceStore = useVoiceStore()
</script>

<template>
  <div class="voice-bar">
    <div class="voice-bar-main">
      <div class="voice-bar-info">
        <i class="fas fa-volume-up voice-bar-icon"></i>
        <div class="voice-bar-text">
          <div class="voice-bar-channel">#{{ voiceStore.voiceChannel?.name }}</div>
          <div class="voice-bar-guild">語音連線中</div>
        </div>
      </div>
      <div class="voice-bar-controls">
        <!-- Microphone -->
        <button
          :class="['voice-bar-toggle', { off: !voiceStore.voiceSelfState.micEnabled }]"
          :title="voiceStore.voiceSelfState.micEnabled ? '關閉麥克風' : '開啟麥克風'"
          @click="voice.toggleMicrophone"
        >
          <i :class="['fas', voiceStore.voiceSelfState.micEnabled ? 'fa-microphone' : 'fa-microphone-slash']"></i>
        </button>
        <!-- Deafen -->
        <button
          :class="['voice-bar-toggle', { off: voiceStore.voiceSelfState.deafened }]"
          :title="voiceStore.voiceSelfState.deafened ? '開啟收音' : '關閉收音'"
          @click="voice.toggleDeafen"
        >
          <i :class="['fas', voiceStore.voiceSelfState.deafened ? 'fa-volume-xmark' : 'fa-volume-high']"></i>
        </button>
        <!-- Camera -->
        <button
          :class="['voice-bar-toggle', { active: voiceStore.voiceSelfState.cameraEnabled }]"
          :title="voiceStore.voiceSelfState.cameraEnabled ? '關閉攝影機' : '開啟攝影機'"
          @click="voice.toggleCamera"
        >
          <i :class="['fas', voiceStore.voiceSelfState.cameraEnabled ? 'fa-video' : 'fa-video-slash']"></i>
        </button>
        <!-- Screen share -->
        <button
          :class="['voice-bar-toggle', { active: voiceStore.voiceSelfState.screenSharing }]"
          :title="voiceStore.voiceSelfState.screenSharing ? '停止螢幕分享' : '分享螢幕'"
          @click="voice.toggleScreenShare"
        >
          <i class="fas fa-display"></i>
        </button>
        <!-- Open video overlay (shown when there are video streams) -->
        <button
          v-if="voiceStore.remoteVideoTracks.length > 0 || voiceStore.voiceSelfState.screenSharing || voiceStore.voiceSelfState.cameraEnabled"
          class="voice-bar-toggle"
          title="開啟視訊視窗"
          @click="voiceStore.videoOverlayOpen = true"
        >
          <i class="fas fa-expand"></i>
        </button>
        <!-- Leave -->
        <button class="voice-bar-leave" title="離開語音" @click="voice.leaveVoiceChannel">
          <i class="fas fa-phone-slash"></i>
        </button>
      </div>
    </div>

    <!-- Current participants -->
    <div class="voice-bar-participants">
      <template v-if="voiceStore.getChannelParticipants(voiceStore.voiceChannel?.id).length">
        <div
          v-for="p in voiceStore.getChannelParticipants(voiceStore.voiceChannel?.id)"
          :key="p.user_id"
          class="voice-bar-participant"
        >
          <div class="voice-bar-participant-avatar">
            <div :class="['vp-avatar', { 'avatar-speaking': voiceStore.isSpeaking(p.user_id) }]">
              <i class="fas fa-user"></i>
            </div>
          </div>
          <span class="voice-bar-participant-name">{{ p.username }}</span>
          <div class="voice-bar-participant-state">
            <i
              v-if="voiceStore.getParticipantState(voiceStore.voiceChannel?.id, p.user_id)?.mic_enabled === false"
              class="fas fa-microphone-slash"
              style="color: var(--danger)"
              title="已靜音"
            ></i>
            <i
              v-if="voiceStore.getParticipantState(voiceStore.voiceChannel?.id, p.user_id)?.deafened"
              class="fas fa-volume-xmark"
              style="color: var(--danger)"
              title="已關閉收音"
            ></i>
            <i
              v-if="voiceStore.getParticipantState(voiceStore.voiceChannel?.id, p.user_id)?.screen_sharing"
              class="fas fa-display"
              style="color: var(--brand)"
              title="螢幕分享中"
            ></i>
            <i
              v-if="voiceStore.getParticipantState(voiceStore.voiceChannel?.id, p.user_id)?.camera_enabled"
              class="fas fa-video"
              style="color: var(--brand)"
              title="攝影機開啟中"
            ></i>
          </div>
        </div>
      </template>
      <div v-else class="voice-bar-empty" style="font-size:12px;color:var(--text-muted);padding:4px 0">
        目前只有你在語音頻道
      </div>
    </div>
  </div>
</template>

