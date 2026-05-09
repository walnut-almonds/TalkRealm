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

    const voiceSelfState = ref({ micEnabled: true, deafened: false })

    // Set of participant identities currently speaking (LiveKit ActiveSpeakers)
    const speakingIdentities = ref(new Set())

    // Map: participant.identity → user_id (to map speakers to known users)
    const identityToUserId = ref(new Map())

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
    }

    function cleanupAudio() {
        voiceAudioElements.value.forEach(el => { if (el?.parentNode) el.remove() })
        voiceAudioElements.value.clear()
    }

    function applyDeafenState() {
        const muted = voiceSelfState.value.deafened
        voiceAudioElements.value.forEach(el => { if (el) el.muted = muted })
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
        voiceSelfState.value = { micEnabled: true, deafened: false }
        speakingIdentities.value = new Set()
    }

    return {
        voiceChannel, voiceRoom, voiceParticipants, voiceParticipantStates,
        voiceSelfState, speakingIdentities, identityToUserId,
        speakingUserIds, voiceAudioElements,
        setSpeakingIdentities, isSpeaking,
        upsertParticipantState, removeParticipantState, getParticipantState,
        upsertParticipant, removeParticipant, getChannelParticipants,
        attachAudioTrack, detachAudioTrack, cleanupAudio, applyDeafenState,
        ensureAudioContainer, trackKey, playNotificationSound, reset,
    }
})
