<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useVoiceStore } from '@/stores/useVoiceStore.js'

const voice = useVoiceStore()

// Pinned track index (null = no pin, show all in grid)
const pinnedIdx = ref(null)

// Tracks to display: remote tracks + self camera/screen (managed via local video elements)
const allTracks = computed(() => voice.remoteVideoTracks)

const selfVideoEl = ref(null)       // <video> for self camera preview
const selfScreenEl = ref(null)      // <video> for self screen share preview

// Watch for self camera / screen share changes to attach local tracks
watch(() => voice.voiceSelfState.cameraEnabled, (enabled) => {
    if (enabled) {
        attachSelfTrack('camera')
    } else {
        detachSelfTrack('camera')
    }
})

watch(() => voice.voiceSelfState.screenSharing, (enabled) => {
    if (enabled) {
        attachSelfTrack('screen')
    } else {
        detachSelfTrack('screen')
    }
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

// When overlay opens, try to attach self tracks that may already be active
watch(() => voice.videoOverlayOpen, (open) => {
    if (open) {
        // defer to next tick so <video> refs are mounted
        setTimeout(() => {
            if (voice.voiceSelfState.cameraEnabled) attachSelfTrack('camera')
            if (voice.voiceSelfState.screenSharing) attachSelfTrack('screen')
        }, 50)
    }
})

// Mount remote video tracks into their wrapper <div> elements
function mountTrackElement(el, track) {
    if (!el || !track?.element) return
    // Only append once
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
const hasAnyVideo = computed(() => allTracks.value.length > 0 || hasSelfCamera.value || hasSelfScreen.value)

// Total panel count for grid sizing
const panelCount = computed(() => {
    let c = allTracks.value.length
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
            <button class="vvo-btn-icon" title="關閉視訊視窗" @click="close">
              <i class="fas fa-xmark"></i>
            </button>
          </div>
        </div>

        <!-- Video grid -->
        <div :class="['vvo-grid', `vvo-grid-${Math.min(panelCount, 4)}`]">

          <!-- Remote video tracks -->
          <div
            v-for="(track, idx) in allTracks"
            :key="track.trackSid"
            :class="['vvo-tile', { 'vvo-tile-pinned': pinnedIdx === idx }]"
            @click="pin(idx)"
          >
            <div
              class="vvo-video-wrapper"
              :ref="el => mountTrackElement(el, track)"
            ></div>
            <div class="vvo-tile-label">
              <i :class="['fas', track.kind === 'screen' ? 'fa-display' : 'fa-video']"></i>
              {{ labelOf(track) }}
            </div>
            <button
              class="vvo-pin-btn"
              :title="pinnedIdx === idx ? '取消固定' : '固定此畫面'"
              @click.stop="pin(idx)"
            >
              <i :class="['fas', pinnedIdx === idx ? 'fa-thumbtack fa-rotate-90' : 'fa-thumbtack']"></i>
            </button>
          </div>

          <!-- Self camera preview -->
          <div v-if="hasSelfCamera" class="vvo-tile vvo-tile-self">
            <video ref="selfVideoEl" class="vvo-self-video" autoplay playsinline muted></video>
            <div class="vvo-tile-label">
              <i class="fas fa-video"></i>
              你的攝影機
            </div>
          </div>

          <!-- Self screen share preview -->
          <div v-if="hasSelfScreen" class="vvo-tile vvo-tile-self">
            <video ref="selfScreenEl" class="vvo-self-video" autoplay playsinline muted></video>
            <div class="vvo-tile-label">
              <i class="fas fa-display"></i>
              你的螢幕分享
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
