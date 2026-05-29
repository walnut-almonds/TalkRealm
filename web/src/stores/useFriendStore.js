import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/index.js'

export const useFriendStore = defineStore('friend', () => {
    const friends = ref([])
    const incomingRequests = ref([])
    const outgoingRequests = ref([])
    const loading = ref(false)

    async function loadAll() {
        loading.value = true
        try {
            const [f, inc, out] = await Promise.all([
                api.listFriends(),
                api.listIncomingRequests(),
                api.listOutgoingRequests(),
            ])
            friends.value = f.friends ?? []
            incomingRequests.value = inc.requests ?? []
            outgoingRequests.value = out.requests ?? []
        } finally {
            loading.value = false
        }
    }

    async function sendRequest(username) {
        const f = await api.sendFriendRequest(username)
        outgoingRequests.value.push(f)
        return f
    }

    async function acceptRequest(userId) {
        const f = await api.acceptFriendRequest(userId)
        incomingRequests.value = incomingRequests.value.filter(
            r => r.requester_id !== userId && r.addressee_id !== userId
        )
        friends.value.push(f)
        return f
    }

    async function rejectRequest(userId) {
        await api.removeFriend(userId)
        incomingRequests.value = incomingRequests.value.filter(
            r => r.requester_id !== userId
        )
    }

    async function unfriend(userId) {
        await api.removeFriend(userId)
        friends.value = friends.value.filter(f => {
            return f.requester_id !== userId && f.addressee_id !== userId
        })
    }

    // ── WS event handlers ──
    function onFriendRequest(data) {
        const dup = incomingRequests.value.find(r => r.id === data.id)
        if (!dup) incomingRequests.value.push(data)
    }

    function onFriendAccept(data) {
        outgoingRequests.value = outgoingRequests.value.filter(r => r.id !== data.id)
        const dup = friends.value.find(f => f.id === data.id)
        if (!dup) friends.value.push(data)
    }

    function onFriendReject(data) {
        outgoingRequests.value = outgoingRequests.value.filter(
            r => r.addressee_id !== data.user_id
        )
    }

    function onFriendRemove(data) {
        friends.value = friends.value.filter(
            f => f.requester_id !== data.user_id && f.addressee_id !== data.user_id
        )
    }

    return {
        friends,
        incomingRequests,
        outgoingRequests,
        loading,
        loadAll,
        sendRequest,
        acceptRequest,
        rejectRequest,
        unfriend,
        onFriendRequest,
        onFriendAccept,
        onFriendReject,
        onFriendRemove,
    }
})
