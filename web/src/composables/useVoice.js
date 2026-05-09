import { api } from '@/api/index.js'
import { useVoiceStore } from '@/stores/useVoiceStore.js'
import { useWebSocket } from './useWebSocket.js'
import { Room, RoomEvent } from 'livekit-client'

export function useVoice(appStore) {
    const voice = useVoiceStore()
    const ws = useWebSocket()

    async function joinVoiceChannel(channelId) {
        // toggle: if already in this channel, leave
        if (voice.voiceChannel?.id === channelId) {
            await leaveVoiceChannel()
            return
        }
        // if in another channel, leave first
        if (voice.voiceChannel) await leaveVoiceChannel()

        const channel = appStore.channels.find(c => c.id === channelId)
        if (!channel) return

        try {
            appStore.setLoading(true)
            const { token, url } = await api.getVoiceToken(channelId)
            const room = new Room()

            voice.cleanupAudio()

            // ── Event handlers ──────────────────────────────────────
            room.on(RoomEvent.TrackSubscribed, (track, pub, participant) => {
                voice.attachAudioTrack(track, pub, participant)
            })
            room.on(RoomEvent.TrackUnsubscribed, (track, pub, participant) => {
                voice.detachAudioTrack(track, pub, participant)
            })
            room.on(RoomEvent.TrackSubscriptionFailed, (sid, participant) => {
                console.warn('[voice] subscription failed', sid, participant?.identity)
            })

            // Speaking indicator: glow on active speakers
            room.on(RoomEvent.ActiveSpeakersChanged, (speakers) => {
                voice.setSpeakingIdentities(speakers.map(s => s.identity))
            })

            // Data channel: sync mic/deafen state
            room.on(RoomEvent.DataReceived, (payload, participant, _kind, topic) => {
                handleVoiceData(payload, participant, topic)
            })

            room.on(RoomEvent.ParticipantConnected, (participant) => {
                // Broadcast our own state so the new participant knows about us
                broadcastSelfState(room, voice.voiceChannel?.id, appStore.user)
                // Add them to the visible participant list (identity→userId registered via DataReceived)
                const userId = voice.identityToUserId.get(participant.identity)
                if (userId) {
                    voice.upsertParticipant(voice.voiceChannel?.id ?? channelId, userId, participant.name || participant.identity)
                }
            })

            room.on(RoomEvent.ParticipantDisconnected, (participant) => {
                const userId = voice.identityToUserId.get(participant.identity)
                const cid = voice.voiceChannel?.id ?? channelId
                if (userId) {
                    voice.removeParticipant(cid, userId)
                    voice.removeParticipantState(cid, userId)
                    voice.identityToUserId.delete(participant.identity)
                }
            })

            room.on(RoomEvent.Disconnected, () => {
                voice.cleanupAudio()
                voice.voiceChannel = null
                voice.voiceRoom = null
                voice.setSpeakingIdentities([])
            })

            room.on(RoomEvent.AudioPlaybackStatusChanged, () => {
                if (!room.canPlaybackAudio) {
                    room.startAudio().catch(e => console.warn('[voice] startAudio failed', e))
                }
            })

            await room.connect(url, token)

            // Register our own identity → userId mapping & add self to participant list
            if (appStore.user) {
                voice.identityToUserId.set(room.localParticipant.identity, appStore.user.id)
                voice.upsertParticipant(channelId, appStore.user.id, appStore.user.nickname || appStore.user.username || 'Unknown')
                voice.upsertParticipantState(channelId, appStore.user.id, {
                    user_id: appStore.user.id,
                    username: appStore.user.nickname || appStore.user.username || 'Unknown',
                    mic_enabled: true,
                    deafened: false,
                })
            }

            // Attach existing remote tracks & add them to participant list
            room.remoteParticipants.forEach(p => {
                p.audioTrackPublications?.forEach(pub => {
                    if (pub.track && pub.isSubscribed) voice.attachAudioTrack(pub.track, pub, p)
                })
                // Register and show existing remote participants
                const remoteUserId = voice.identityToUserId.get(p.identity)
                if (remoteUserId) {
                    voice.upsertParticipant(channelId, remoteUserId, p.name || p.identity)
                }
            })

            await room.localParticipant.setMicrophoneEnabled(true)
            voice.voiceSelfState.micEnabled = true
            voice.voiceSelfState.deafened = false
            voice.voiceRoom = room
            voice.voiceChannel = channel

            // Broadcast join via WS
            ws.sendVoiceStateUpdate(channelId, 'join')
            broadcastSelfState(room, channelId, appStore.user)

            appStore.showNotification(`已加入語音頻道 #${channel.name}`, 'success')
        } catch (e) {
            console.error('[voice] join failed', e)
            appStore.showNotification(`加入語音頻道失敗：${e.message}`, 'error')
            voice.reset()
        } finally {
            appStore.setLoading(false)
        }
    }

    async function leaveVoiceChannel() {
        if (!voice.voiceChannel) return
        const { id: channelId, name: channelName } = voice.voiceChannel
        if (voice.voiceRoom) {
            await voice.voiceRoom.disconnect()
            voice.voiceRoom = null
        }
        voice.cleanupAudio()
        ws.sendVoiceStateUpdate(channelId, 'leave')
        if (appStore.user) {
            voice.removeParticipant(channelId, appStore.user.id)
            voice.removeParticipantState(channelId, appStore.user.id)
        }
        voice.voiceChannel = null
        voice.voiceSelfState.micEnabled = true
        voice.voiceSelfState.deafened = false
        voice.setSpeakingIdentities([])
        appStore.showNotification(`已離開語音頻道 #${channelName}`, 'info')
    }

    async function toggleMicrophone() {
        if (!voice.voiceRoom || !voice.voiceChannel) return
        const next = !voice.voiceSelfState.micEnabled
        try {
            await voice.voiceRoom.localParticipant.setMicrophoneEnabled(next)
            voice.voiceSelfState.micEnabled = next
            if (appStore.user) {
                voice.upsertParticipantState(voice.voiceChannel.id, appStore.user.id, {
                    user_id: appStore.user.id,
                    username: appStore.user.nickname || appStore.user.username,
                    mic_enabled: next,
                    deafened: voice.voiceSelfState.deafened,
                })
            }
            broadcastSelfState(voice.voiceRoom, voice.voiceChannel.id, appStore.user)
        } catch (e) {
            console.error('[voice] toggleMic failed', e)
            appStore.showNotification('切換麥克風失敗', 'error')
        }
    }

    function toggleDeafen() {
        if (!voice.voiceRoom || !voice.voiceChannel) return
        voice.voiceSelfState.deafened = !voice.voiceSelfState.deafened
        voice.applyDeafenState()
        if (appStore.user) {
            voice.upsertParticipantState(voice.voiceChannel.id, appStore.user.id, {
                user_id: appStore.user.id,
                username: appStore.user.nickname || appStore.user.username,
                mic_enabled: voice.voiceSelfState.micEnabled,
                deafened: voice.voiceSelfState.deafened,
            })
        }
        broadcastSelfState(voice.voiceRoom, voice.voiceChannel.id, appStore.user)
    }

    function handleVoiceStateUpdate(data) {
        const { channel_id, user_id, username, action } = data
        if (action === 'join') {
            voice.upsertParticipant(channel_id, user_id, username || 'Unknown')
            voice.upsertParticipantState(channel_id, user_id, { user_id, username: username || 'Unknown', mic_enabled: true, deafened: false })
        } else if (action === 'leave') {
            voice.removeParticipant(channel_id, user_id)
            voice.removeParticipantState(channel_id, user_id)
        }
        const isSelf = appStore.user && user_id === appStore.user.id
        if (!isSelf && (action === 'join' || action === 'leave')) {
            voice.playNotificationSound(action)
        }
    }

    // ── Internal ─────────────────────────────────────────────────
    function broadcastSelfState(room, channelId, user) {
        if (!room || !channelId || !user) return
        const payload = {
            type: 'voice_user_state',
            channel_id: channelId,
            user_id: user.id,
            username: user.nickname || user.username || 'Unknown',
            mic_enabled: voice.voiceSelfState.micEnabled,
            deafened: voice.voiceSelfState.deafened,
            ts: Date.now(),
        }
        try {
            const encoded = new TextEncoder().encode(JSON.stringify(payload))
            const p = room.localParticipant.publishData(encoded, { reliable: true, topic: 'voice-user-state' })
            p?.catch(e => console.warn('[voice] publishData failed', e))
        } catch (e) {
            console.warn('[voice] broadcast state failed', e)
        }
    }

    function handleVoiceData(payload, participant, topic) {
        if (topic && topic !== 'voice-user-state') return
        try {
            const text = typeof payload === 'string' ? payload : new TextDecoder().decode(payload)
            const data = JSON.parse(text)
            if (data.type !== 'voice_user_state') return
            const { channel_id, user_id, username, mic_enabled, deafened } = data
            if (!channel_id || !user_id) return

            // Register identity→userId for speaking detection
            if (participant?.identity) {
                voice.identityToUserId.set(participant.identity, user_id)
            }

            // Show the participant in the list as soon as we know their identity
            const displayName = username || participant?.name || 'Unknown'
            voice.upsertParticipant(channel_id, user_id, displayName)

            voice.upsertParticipantState(channel_id, user_id, {
                user_id,
                username: username || participant?.name || 'Unknown',
                mic_enabled: mic_enabled !== false,
                deafened: deafened === true,
            })
        } catch { }
    }

    return { joinVoiceChannel, leaveVoiceChannel, toggleMicrophone, toggleDeafen, handleVoiceStateUpdate }
}
