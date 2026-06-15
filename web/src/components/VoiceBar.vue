<script setup>
import { inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVoiceStore } from '@/stores/useVoiceStore.js'

const voice = inject('voice')
const voiceStore = useVoiceStore()
const { t } = useI18n()
</script>

<template>
  <div class="voice-bar">
    <!-- Channel info: icon + name + status -->
    <div class="voice-bar-info">
      <i class="fas fa-volume-up voice-bar-icon"></i>
      <div class="voice-bar-text">
        <div class="voice-bar-channel">#{{ voiceStore.voiceChannel?.name }}</div>
        <div class="voice-bar-guild">{{ t('voice.connected') }}</div>
      </div>
    </div>

    <!-- Control buttons centered below -->
    <div class="voice-bar-controls">
      <!-- Microphone -->
      <button
        :class="['voice-bar-toggle', { off: !voiceStore.voiceSelfState.micEnabled }]"
        :title="voiceStore.voiceSelfState.micEnabled ? t('voice.micOff') : t('voice.micOn')"
        @click="voice.toggleMicrophone"
      >
        <i :class="['fas', voiceStore.voiceSelfState.micEnabled ? 'fa-microphone' : 'fa-microphone-slash']"></i>
      </button>
      <!-- Deafen -->
      <button
        :class="['voice-bar-toggle', { off: voiceStore.voiceSelfState.deafened }]"
        :title="voiceStore.voiceSelfState.deafened ? t('voice.deafenOff') : t('voice.deafenOn')"
        @click="voice.toggleDeafen"
      >
        <i :class="['fas', voiceStore.voiceSelfState.deafened ? 'fa-volume-xmark' : 'fa-volume-high']"></i>
      </button>
      <!-- Camera -->
      <button
        :class="['voice-bar-toggle', { active: voiceStore.voiceSelfState.cameraEnabled }]"
        :title="voiceStore.voiceSelfState.cameraEnabled ? t('voice.cameraOff') : t('voice.cameraOnAction')"
        @click="voice.toggleCamera"
      >
        <i :class="['fas', voiceStore.voiceSelfState.cameraEnabled ? 'fa-video' : 'fa-video-slash']"></i>
      </button>
      <!-- Screen share -->
      <button
        :class="['voice-bar-toggle', { active: voiceStore.voiceSelfState.screenSharing }]"
        :title="voiceStore.voiceSelfState.screenSharing ? t('voice.stopScreenShare') : t('voice.startScreenShare')"
        @click="voice.toggleScreenShare"
      >
        <i class="fas fa-display"></i>
      </button>
      <!-- Open video overlay -->
      <button
        v-if="voiceStore.remoteVideoTracks.length > 0 || voiceStore.voiceSelfState.screenSharing || voiceStore.voiceSelfState.cameraEnabled"
        class="voice-bar-toggle"
        :title="t('voice.openVideoOverlay')"
        @click="voiceStore.videoOverlayOpen = true"
      >
        <i class="fas fa-expand"></i>
      </button>
      <!-- Leave -->
      <button class="voice-bar-leave" :title="t('voice.leave')" @click="voice.leaveVoiceChannel">
        <i class="fas fa-phone-slash"></i>
      </button>
    </div>
  </div>
</template>

