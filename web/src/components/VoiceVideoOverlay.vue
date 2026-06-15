<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { useVoice } from '@/composables/useVoice.js'

const voice = useVoiceStore()
const { updateVideoQuality, updateScreenFps } = useVoice()
const { t } = useI18n()

// Pinned track index (null = no pin, show all in grid)
const pinnedIdx = ref(null)

// Settings panel visibility
const showSettings = ref(false)

const CAMERA_QUALITY_OPTIONS = [
  { value: '360p', label: t('voiceOverlay.quality360p') },
  { value: '720p', label: t('voiceOverlay.quality720p') },
  { value: '1080p', label: t('voiceOverlay.quality1080p') },
]

const SCREEN_FPS_OPTIONS = [
  { value: 5,  label: t('voiceOverlay.fps5') },
  { value: 15, label: t('voiceOverlay.fps15') },
  { value: 30, label: t('voiceOverlay.fps30') },
  { value: 60, label: t('voiceOverlay.fps60') },
]

function onQualityChange(val) {
    voice.videoQuality = val
    updateVideoQuality(val)
}

function onFpsChange(val) {
    voice.screenShareFps = Number(val)
    updateScreenFps(Number(val))
}

// Per-participant volume helpers
function getVolume(identity) {
    return Math.round((voice.participantVolumes.get(identity) ?? 1) * 100)
}

function setVolume(identity, event) {
    const vol = Number(event.target.value) / 100
    voice.setParticipantVolume(identity, vol)
}

const selfVideoEl = ref(null)
const selfScreenEl = ref(null)

watch(() => voice.voiceSelfState.cameraEnabled, (enabled) => {
    enabled ? attachSelfTrack('camera') : detachSelfTrack('camera')
})

watch(() => voice.voiceSelfState.screenSharing, (enabled) => {
    enabled ? attachSelfTrack('screen') : detachSelfTrack('screen')
})

function attachSelfTrack(kind) {
    if (!voice.voiceRoom) return
    const pub = kind === 'camera'
        ? voice.voiceRoom.localParticipant.getTrackPublication('camera')
        : voice.voiceRoom.localParticipant.getTrackPublication('screen_share')
    if (!pub?.track) return
    const el = kind === 'camera' ? selfVideoEl.value : selfScreenEl.value
    if (el) {
        pub.track.attach(el)
        el.play?.().catch(() => { })
    }
}

function detachSelfTrack(kind) {
    const el = kind === 'camera' ? selfVideoEl.value : selfScreenEl.value
    if (!el) return
    if (!voice.voiceRoom) return
    const pub = kind === 'camera'
        ? voice.voiceRoom.localParticipant.getTrackPublication('camera')
        : voice.voiceRoom.localParticipant.getTrackPublication('screen_share')
    try { pub?.track?.detach(el) } catch { }
}

watch(() => voice.videoOverlayOpen, (open) => {
    if (open) {
        setTimeout(() => {
            if (voice.voiceSelfState.cameraEnabled) attachSelfTrack('camera')
            if (voice.voiceSelfState.screenSharing) attachSelfTrack('screen')
        }, 50)
    }
})

function mountTrackElement(el, track) {
    if (!el || !track?.element) return
    if (!el.contains(track.element)) {
        el.appendChild(track.element)
    }
}

function close() {
    voice.videoOverlayOpen = false
    pinnedIdx.value = null
}

function pin(idx) {
    pinnedIdx.value = pinnedIdx.value === idx ? null : idx
}

function labelOf(track) {
  const name = track.username || track.participantIdentity || t('voiceOverlay.participant')
  return track.kind === 'screen'
    ? t('voiceOverlay.screenOf', { name })
    : t('voiceOverlay.cameraOf', { name })
}

const hasSelfCamera = computed(() => voice.voiceSelfState.cameraEnabled)
const hasSelfScreen = computed(() => voice.voiceSelfState.screenSharing)
const hasAnyVideo = computed(() =>
    voice.remoteVideoTracks.length > 0 || hasSelfCamera.value || hasSelfScreen.value
)

const panelCount = computed(() => {
    let c = voice.remoteVideoTracks.length
    if (hasSelfCamera.value) c++
    if (hasSelfScreen.value) c++
    return c
})
</script>

<template>
  <Teleport to="body">
    <div v-if="voice.videoOverlayOpen" class="voice-video-overlay" @click.self="close">
      <div class="vvo-panel">

        <!-- Header -->
        <div class="vvo-header">
          <span class="vvo-title">
            <i class="fas fa-video"></i>
            {{ t('voiceOverlay.title') }} &mdash; {{ voice.voiceChannel?.name ?? '' }}
          </span>
          <div class="vvo-header-actions">
            <button
              :class="['vvo-btn-icon', { active: showSettings }]"
              :title="t('voiceOverlay.settings')"
              @click="showSettings = !showSettings"
            >
              <i class="fas fa-sliders"></i>
            </button>
            <button class="vvo-btn-icon" :title="t('voiceOverlay.close')" @click="close">
              <i class="fas fa-xmark"></i>
            </button>
          </div>
        </div>

        <!-- Quality settings panel -->
        <Transition name="vvo-slide">
          <div v-if="showSettings" class="vvo-settings">
            <div class="vvo-settings-row">
              <label class="vvo-settings-label">
                <i class="fas fa-video"></i> {{ t('voiceOverlay.cameraQuality') }}
              </label>
              <select
                class="vvo-select"
                :value="voice.videoQuality"
                @change="onQualityChange($event.target.value)"
              >
                <option v-for="opt in CAMERA_QUALITY_OPTIONS" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
            <div class="vvo-settings-row">
              <label class="vvo-settings-label">
                <i class="fas fa-display"></i> {{ t('voiceOverlay.screenFps') }}
              </label>
              <select
                class="vvo-select"
                :value="voice.screenShareFps"
                @change="onFpsChange($event.target.value)"
              >
                <option v-for="opt in SCREEN_FPS_OPTIONS" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </div>
        </Transition>

        <!-- Video grid -->
        <div :class="['vvo-grid', `vvo-grid-${Math.min(panelCount, 4)}`]">

          <!-- Remote video tracks -->
          <div
            v-for="(track, idx) in voice.remoteVideoTracks"
            :key="track.trackSid"
            :class="['vvo-tile', { 'vvo-tile-pinned': pinnedIdx === idx }]"
            @click="pin(idx)"
          >
            <div class="vvo-video-wrapper" :ref="el => mountTrackElement(el, track)"></div>

            <!-- Label -->
            <div class="vvo-tile-label">
              <i :class="['fas', track.kind === 'screen' ? 'fa-display' : 'fa-video']"></i>
              {{ labelOf(track) }}
            </div>

            <!-- Volume slider -->
            <div class="vvo-volume-control" @click.stop>
              <i class="fas fa-volume-low vvo-vol-icon"></i>
              <input
                type="range"
                class="vvo-volume-slider"
                min="0"
                max="100"
                :value="getVolume(track.participantIdentity)"
                @input="setVolume(track.participantIdentity, $event)"
              />
              <span class="vvo-vol-pct">{{ getVolume(track.participantIdentity) }}%</span>
            </div>

            <!-- Pin button -->
            <button
              class="vvo-pin-btn"
              :title="pinnedIdx === idx ? t('voiceOverlay.unpin') : t('voiceOverlay.pin')"
              @click.stop="pin(idx)"
            >
              <i :class="['fas', pinnedIdx === idx ? 'fa-thumbtack fa-rotate-90' : 'fa-thumbtack']"></i>
            </button>
          </div>

          <!-- Self camera -->
          <div v-if="hasSelfCamera" class="vvo-tile vvo-tile-self">
            <video ref="selfVideoEl" class="vvo-self-video" autoplay playsinline muted></video>
            <div class="vvo-tile-label">
              <i class="fas fa-video"></i>
              {{ t('voiceOverlay.yourCameraPreview') }}
            </div>
          </div>

          <!-- Self screen share -->
          <div v-if="hasSelfScreen" class="vvo-tile vvo-tile-self">
            <video ref="selfScreenEl" class="vvo-self-video" autoplay playsinline muted></video>
            <div class="vvo-tile-label">
              <i class="fas fa-display"></i>
              {{ t('voiceOverlay.yourScreenPreview') }}
            </div>
          </div>

          <!-- Empty state -->
          <div v-if="!hasAnyVideo" class="vvo-empty">
            <i class="fas fa-video-slash"></i>
            <p>{{ t('voiceOverlay.noStream') }}</p>
            <p class="vvo-empty-hint">{{ t('voiceOverlay.noStreamHint') }}</p>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
