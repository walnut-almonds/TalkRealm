import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { api as apiClient } from '@/api/index.js'
import { useAppStore } from '@/stores/useAppStore.js'

export const useDMStore = defineStore('dm', () => {
    const dmChannels = ref([])
    const currentDMChannel = ref(null)
    const dmMessages = ref([])
    const isDMMode = ref(false)
    const isLoadingMessages = ref(false)
    const hasMoreMessages = ref(false)
    const dmPendingFileIds = ref([])
    const dmTranslationCache = reactive(new Map())
    const dmTranslationLoadingSet = reactive(new Set())

    async function loadDMChannels() {
        const res = await apiClient.listDMChannels()
        dmChannels.value = res.channels || []
    }

    async function openDMWith(targetUserId) {
        const channel = await apiClient.openDMChannel(targetUserId)
        // Ensure it's in the list
        const idx = dmChannels.value.findIndex(c => c.id === channel.id)
        if (idx === -1) dmChannels.value.unshift(channel)
        await openDMChannel(channel)
    }

    async function openDMChannel(channel) {
        currentDMChannel.value = channel
        isDMMode.value = true
        dmMessages.value = []
        hasMoreMessages.value = false
        await loadDMMessages(channel.id)
        // ACK: 標記 DM 頻道已讀
        const appStore = useAppStore()
        const lastMsg = dmMessages.value[dmMessages.value.length - 1]
        if (lastMsg?.id) {
            appStore.channelUnreadMap.delete(channel.id)
            apiClient.ackChannel(channel.id, lastMsg.id).catch(() => { })
        }
    }

    async function loadDMMessages(channelId, before = null) {
        isLoadingMessages.value = true
        try {
            const res = await apiClient.getDMMessages(channelId, 50, before)
            const msgs = res.messages || []
            if (before) {
                dmMessages.value = [...msgs, ...dmMessages.value]
            } else {
                dmMessages.value = msgs
            }
            hasMoreMessages.value = msgs.length === 50
        } finally {
            isLoadingMessages.value = false
        }
    }

    async function sendDM(content, nonce = null, fileIds = []) {
        if (!currentDMChannel.value) return
        const msg = await apiClient.sendDMMessage(currentDMChannel.value.id, content, nonce, fileIds)
        pushIncomingDM(msg)
        return msg
    }

    function getMessageChannelId(message) {
        return message?.channel_id || message?.dm_channel_id || 0
    }

    function pushIncomingDM(message) {
        const messageChannelID = getMessageChannelId(message)
        const isCurrentChannel = currentDMChannel.value && messageChannelID === currentDMChannel.value.id
        if (isCurrentChannel) {
            // Avoid duplicates from optimistic/REST + WS echoes.
            if (message.id && dmMessages.value.some(m => m.id === message.id)) return
            if (message.nonce && dmMessages.value.some(m => m.nonce === message.nonce)) return
            dmMessages.value.push(message)
        } else {
            // Track DM unread in shared channelUnreadMap
            const appStore = useAppStore()
            const cur = appStore.channelUnreadMap.get(messageChannelID) || { unread: 0, mention: 0 }
            appStore.channelUnreadMap.set(messageChannelID, { ...cur, unread: cur.unread + 1 })
        }
        // Move channel to top of list
        const idx = dmChannels.value.findIndex(c => c.id === messageChannelID)
        if (idx > 0) {
            const [ch] = dmChannels.value.splice(idx, 1)
            dmChannels.value.unshift(ch)
        }
    }

    function handleDMMessageUpdate(data) {
        const idx = dmMessages.value.findIndex(m => m.id === data.id)
        if (idx !== -1) {
            dmMessages.value[idx] = { ...dmMessages.value[idx], ...data }
        }
    }

    function handleDMMessageDelete(data) {
        const msgId = data.message_id || data.id
        dmMessages.value = dmMessages.value.filter(m => m.id !== msgId)
    }

    function handleDMTranslationReady(data) {
        const msgId = data.message_id || data.dm_message_id
        if (msgId) {
            dmTranslationCache.set(msgId, data)
            dmTranslationLoadingSet.delete(msgId)
        }
    }

    function exitDMMode() {
        isDMMode.value = false
        currentDMChannel.value = null
        dmMessages.value = []
    }

    return {
        dmChannels,
        currentDMChannel,
        dmMessages,
        isDMMode,
        isLoadingMessages,
        hasMoreMessages,
        dmPendingFileIds,
        dmTranslationCache,
        dmTranslationLoadingSet,
        loadDMChannels,
        openDMWith,
        openDMChannel,
        loadDMMessages,
        sendDM,
        pushIncomingDM,
        handleDMMessageUpdate,
        handleDMMessageDelete,
        handleDMTranslationReady,
        exitDMMode,
    }
})
