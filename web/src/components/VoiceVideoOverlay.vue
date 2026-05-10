<script setup>
import { ref, computed, watch } from 'vue'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { useVoice } from '@/composables/useVoice.js'

const voice = useVoiceStore()
const { updateVideoQuality, updateScreenFps } = useVoice()

// Pinned track index (null = no pin, show all in grid)
const pinnedIdx = ref(null)

// Settings panel visibility
const showSettings = ref(false)

const CAMERA_QUALITY_OPTIONS = [
    { value: '360p', label: '360p（省頻）' },
    { value: '720p', label: '720p（建議）' },
    { value: '1080p', label: '1080p（高畫質）' },
]

const SCREEN_FPS_OPTIONS = [
    { value: 5,  label: '5 FPS（省頻）' },
    { value: 15, label: '15 FPS（建議）' },
    { value: 30, label: '30 FPS（流暢）' },
    { value: 60, label: '60 FPS（最高）' },
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
    const name = track.username || track.participantIdentity || '參與者'
    return track.kind === 'screen' ? `${name} 的螢幕` : `${name} 的攝影機`
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
            語音視訊 &mdash; {{ voice.voiceChannel?.name ?? '' }}
          </span>
          <div class="vvo-header-actions">
            <button
              :class="['vvo-btn-icon', { active: showSettings }]"
              title="畫質 / FPS 設定"
              @click="showSettings = !showSettings"
            >
              <i class="fas fa-sliders"></i>
            </button>
            <button class="vvo-btn-icon" title="關閉視訊視窗" @click="close">
              <i class="fas fa-xmark"></i>
            </button>
          </div>
        </div>

        <!-- Quality settings panel -->
        <Transition name="vvo-slide">
          <div v-if="showSettings" class="vvo-settings">
            <div class="vvo-settings-row">
              <label class="vvo-settings-label">
                <i class="fas fa-video"></i> 攝影機畫質
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
                <i class="fas fa-display"></i> 螢幕分享 FPS
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
              :title="pinnedIdx === idx ? '取消固定' : '固定此畫面'"
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
              你的攝影機（本地預覽）
            </div>
          </div>

          <!-- Self screen share -->
          <div v-if="hasSelfScreen" class="vvo-tile vvo-tile-self">
            <video ref="selfScreenEl" class="vvo-self-video" autoplay playsinline muted></video>
            <div class="vvo-tile-label">
              <i class="fas fa-display"></i>
              你的螢幕分享（本地預覽）
            </div>
          </div>

          <!-- Empty state -->
          <div v-if="!hasAnyVideo" class="vvo-empty">
            <i class="fas fa-video-slash"></i>
            <p>目前沒有視訊串流</p>
            <p class="vvo-empty-hint">按下螢幕分享或攝影機按鈕以開始</p>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
