import { ref } from 'vue'
import { apiClient } from '@/api/index.js'

const dmChannels = ref([])
const currentDMChannel = ref(null)
const dmMessages = ref([])
const isDMMode = ref(false)
const isLoadingMessages = ref(false)
const hasMoreMessages = ref(false)

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

async function sendDM(content, nonce = null) {
    if (!currentDMChannel.value) return
    const msg = await apiClient.sendDMMessage(currentDMChannel.value.id, content, nonce)
    // Optimistic: server will broadcast via WS, but also push locally
    pushIncomingDM(msg)
    return msg
}

function pushIncomingDM(message) {
    if (currentDMChannel.value && message.dm_channel_id === currentDMChannel.value.id) {
        // Avoid duplicates (nonce match)
        if (message.nonce && dmMessages.value.some(m => m.nonce === message.nonce)) return
        dmMessages.value.push(message)
    }
    // Move channel to top of list
    const idx = dmChannels.value.findIndex(c => c.id === message.dm_channel_id)
    if (idx > 0) {
        const [ch] = dmChannels.value.splice(idx, 1)
        dmChannels.value.unshift(ch)
    }
}

function exitDMMode() {
    isDMMode.value = false
    currentDMChannel.value = null
    dmMessages.value = []
}

export function useDMStore() {
    return {
        dmChannels,
        currentDMChannel,
        dmMessages,
        isDMMode,
        isLoadingMessages,
        hasMoreMessages,
        loadDMChannels,
        openDMWith,
        openDMChannel,
        loadDMMessages,
        sendDM,
        pushIncomingDM,
        exitDMMode,
    }
}
