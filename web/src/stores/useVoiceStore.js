import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useVoiceStore = defineStore('voice', () => {
    const voiceChannel = ref(null)   // { id, name }
    const voiceRoom = ref(null)      // LiveKit Room instance
    const voiceAudioElements = ref(new Map()) // trackKey → HTMLAudioElement

    // { channelId: [{ user_id, username }] }
    const voiceParticipants = ref({})

    // { channelId: { userId: { user_id, username, mic_enabled, deafened } } }
    const voiceParticipantStates = ref({})

    const voiceSelfState = ref({ micEnabled: true, deafened: false, screenSharing: false, cameraEnabled: false })

    // Set of participant identities currently speaking (LiveKit ActiveSpeakers)
    const speakingIdentities = ref(new Set())

    // Map: participant.identity → user_id (to map speakers to known users)
    const identityToUserId = ref(new Map())

    // ── Video tracks ──────────────────────────────────────────────
    // Each entry: { trackSid, participantIdentity, kind ('screen'|'camera'), element, userId, username }
    const remoteVideoTracks = ref([])

    // Whether the video overlay panel is open
    const videoOverlayOpen = ref(false)

    // ── Volume control ─────────────────────────────────
    // identity → volume (0..1)
    const participantVolumes = ref(new Map())
    // identity → Set of trackKeys (so we can find audio elements by identity)
    const identityToAudioKeys = ref(new Map())

    // ── Video quality ─────────────────────────────────
    // camera quality preset: '360p' | '720p' | '1080p'
    const videoQuality = ref('720p')
    // screen share max framerate: 5 | 15 | 30
    const screenShareFps = ref(15)

    let sfxContext = null

    // ── Speaking detection ───────────────────────────────────────
    const speakingUserIds = computed(() => {
        const ids = new Set()
        speakingIdentities.value.forEach(identity => {
            const uid = identityToUserId.value.get(identity)
            if (uid != null) ids.add(uid)
        })
        return ids
    })

    function setSpeakingIdentities(identities) {
        speakingIdentities.value = new Set(identities)
    }

    function isSpeaking(userId) {
        return speakingUserIds.value.has(userId)
    }

    // ── Participant state helpers ─────────────────────────────────
    function upsertParticipantState(channelId, userId, partial = {}) {
        if (!voiceParticipantStates.value[channelId]) {
            voiceParticipantStates.value[channelId] = {}
        }
        const prev = voiceParticipantStates.value[channelId][userId] || {
            user_id: userId, username: partial.username || 'Unknown',
            mic_enabled: true, deafened: false
        }
        voiceParticipantStates.value[channelId][userId] = { ...prev, ...partial, user_id: userId }
    }

    function removeParticipantState(channelId, userId) {
        if (voiceParticipantStates.value[channelId]) {
            delete voiceParticipantStates.value[channelId][userId]
        }
    }

    function getParticipantState(channelId, userId) {
        return voiceParticipantStates.value[channelId]?.[userId] || null
    }

    function upsertParticipant(channelId, userId, username) {
        if (!voiceParticipants.value[channelId]) voiceParticipants.value[channelId] = []
        if (!voiceParticipants.value[channelId].some(p => p.user_id === userId)) {
            voiceParticipants.value[channelId].push({ user_id: userId, username })
        }
    }

    function removeParticipant(channelId, userId) {
        if (voiceParticipants.value[channelId]) {
            voiceParticipants.value[channelId] = voiceParticipants.value[channelId].filter(p => p.user_id !== userId)
        }
    }

    function getChannelParticipants(channelId) {
        return voiceParticipants.value[channelId] || []
    }

    // ── Audio elements ────────────────────────────────────────────
    function ensureAudioContainer() {
        let el = document.getElementById('voice-audio-container')
        if (!el) {
            el = document.createElement('div')
            el.id = 'voice-audio-container'
            el.style.display = 'none'
            el.setAttribute('aria-hidden', 'true')
            document.body.appendChild(el)
        }
        return el
    }

    function trackKey(pub, participant, track) {
        return pub?.trackSid || pub?.sid || track?.sid || `${participant?.sid || participant?.identity || 'unknown'}:audio`
    }

    function attachAudioTrack(track, pub, participant) {
        if (!track || track.kind !== 'audio') return
        const key = trackKey(pub, participant, track)
        if (voiceAudioElements.value.has(key)) return
        const container = ensureAudioContainer()
        const el = track.attach()
        el.autoplay = true
        el.playsInline = true
        el.muted = voiceSelfState.value.deafened
        el.dataset.voiceTrackKey = key
        // Apply stored per-participant volume
        if (participant?.identity) {
            el.volume = participantVolumes.value.get(participant.identity) ?? 1
            if (!identityToAudioKeys.value.has(participant.identity)) {
                identityToAudioKeys.value.set(participant.identity, new Set())
            }
            identityToAudioKeys.value.get(participant.identity).add(key)
        }
        container.appendChild(el)
        voiceAudioElements.value.set(key, el)
        el.play?.().catch(err => console.warn('[voice] autoplay blocked', key, err))
    }

    function detachAudioTrack(track, pub, participant) {
        if (!track || track.kind !== 'audio') return
        const key = trackKey(pub, participant, track)
        const el = voiceAudioElements.value.get(key)
        if (!el) return
        try { track.detach(el) } catch { }
        el.remove()
        voiceAudioElements.value.delete(key)
        if (participant?.identity) {
            identityToAudioKeys.value.get(participant.identity)?.delete(key)
        }
    }

    function cleanupAudio() {
        voiceAudioElements.value.forEach(el => { if (el?.parentNode) el.remove() })
        voiceAudioElements.value.clear()
        identityToAudioKeys.value.clear()
    }

    function applyDeafenState() {
        const muted = voiceSelfState.value.deafened
        voiceAudioElements.value.forEach(el => { if (el) el.muted = muted })
    }

    // 設定指定參與者的音量（0..1）
    function setParticipantVolume(identity, vol) {
        const v = Math.max(0, Math.min(1, vol))
        participantVolumes.value.set(identity, v)
        identityToAudioKeys.value.get(identity)?.forEach(key => {
            const el = voiceAudioElements.value.get(key)
            if (el) el.volume = v
        })
    }

    // ── Video track helpers ────────────────────────────────────────
    function addRemoteVideoTrack(track, pub, participant) {
        if (!track || track.kind !== 'video') return
        const sid = pub?.trackSid || pub?.sid || track?.sid || `${participant?.identity}:video`
        if (remoteVideoTracks.value.some(t => t.trackSid === sid)) return

        // Determine if this is a screen share or camera
        const isScreen = pub?.source === 'screen_share' ||
            pub?.trackName?.toLowerCase().includes('screen') ||
            track?.source === 'screen_share'
        const kind = isScreen ? 'screen' : 'camera'

        const userId = identityToUserId.value.get(participant?.identity)
        const username = participant?.name || participant?.identity || 'Unknown'

        const element = track.attach()
        element.autoplay = true
        element.playsInline = true
        element.muted = true  // video tracks are muted (audio comes from audio tracks)
        element.style.width = '100%'
        element.style.height = '100%'
        element.style.objectFit = 'contain'
        element.style.borderRadius = '6px'

        remoteVideoTracks.value.push({ trackSid: sid, participantIdentity: participant?.identity, kind, element, userId, username })

        // Auto-open overlay when someone starts sharing
        if (!videoOverlayOpen.value) videoOverlayOpen.value = true
    }

    function removeRemoteVideoTrack(track, pub, participant) {
        if (!track || track.kind !== 'video') return
        const sid = pub?.trackSid || pub?.sid || track?.sid || `${participant?.identity}:video`
        const idx = remoteVideoTracks.value.findIndex(t => t.trackSid === sid)
        if (idx === -1) return
        const entry = remoteVideoTracks.value[idx]
        try { track.detach(entry.element) } catch { }
        remoteVideoTracks.value.splice(idx, 1)

        // Close overlay if no video streams remain and self isn't sharing
        if (remoteVideoTracks.value.length === 0 && !voiceSelfState.value.screenSharing && !voiceSelfState.value.cameraEnabled) {
            videoOverlayOpen.value = false
        }
    }

    function cleanupVideoTracks() {
        remoteVideoTracks.value.forEach(entry => {
            try { entry.element?.remove() } catch { }
        })
        remoteVideoTracks.value = []
    }

    // ── SFX ───────────────────────────────────────────────────────
    function playNotificationSound(action) {
        const AudioCtx = window.AudioContext || window.webkitAudioContext
        if (!AudioCtx) return
        try {
            if (!sfxContext) sfxContext = new AudioCtx()
            if (sfxContext.state === 'suspended') sfxContext.resume().catch(() => { })
            const ctx = sfxContext
            const osc = ctx.createOscillator()
            const gain = ctx.createGain()
            const now = ctx.currentTime
            const freq = action === 'join' ? 900 : 650
            osc.type = 'sine'
            osc.frequency.setValueAtTime(freq, now)
            osc.frequency.exponentialRampToValueAtTime(freq * 0.9, now + 0.12)
            gain.gain.setValueAtTime(0.0001, now)
            gain.gain.exponentialRampToValueAtTime(0.035, now + 0.01)
            gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.13)
            osc.connect(gain); gain.connect(ctx.destination)
            osc.start(now); osc.stop(now + 0.14)
        } catch (e) {
            console.debug('SFX skipped', e)
        }
    }

    // ── Reset ─────────────────────────────────────────────────────
    function reset() {
        voiceChannel.value = null
        voiceRoom.value = null
        cleanupAudio()
        cleanupVideoTracks()
        voiceSelfState.value = { micEnabled: true, deafened: false, screenSharing: false, cameraEnabled: false }
        speakingIdentities.value = new Set()
        videoOverlayOpen.value = false
        participantVolumes.value.clear()
        identityToAudioKeys.value.clear()
    }

    return {
        voiceChannel, voiceRoom, voiceParticipants, voiceParticipantStates,
        voiceSelfState, speakingIdentities, identityToUserId,
        speakingUserIds, voiceAudioElements,
        remoteVideoTracks, videoOverlayOpen,
        participantVolumes, identityToAudioKeys, videoQuality, screenShareFps,
        setSpeakingIdentities, isSpeaking,
        upsertParticipantState, removeParticipantState, getParticipantState,
        upsertParticipant, removeParticipant, getChannelParticipants,
        attachAudioTrack, detachAudioTrack, cleanupAudio, applyDeafenState, setParticipantVolume,
        addRemoteVideoTrack, removeRemoteVideoTrack, cleanupVideoTracks,
        ensureAudioContainer, trackKey, playNotificationSound, reset,
    }
})
