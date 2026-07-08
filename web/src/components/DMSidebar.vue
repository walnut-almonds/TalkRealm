<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDMStore } from '@/stores/useDMStore.js'
import { useAppStore } from '@/stores/useAppStore.js'
import { useFriendStore } from '@/stores/useFriendStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import { api } from '@/api/index.js'
import UserPanel from './UserPanel.vue'

const emit = defineEmits(['channel-selected'])

const dm = useDMStore()
const store = useAppStore()
const friendStore = useFriendStore()
const ws = useWebSocket()
const { t } = useI18n()

// ── Tabs ──
const activeTab = ref('messages') // 'messages' | 'friends' | 'requests'

// ── DM / Picker ──
const showPicker = ref(false)
const pickerQuery = ref('')

onMounted(async () => {
    await dm.loadDMChannels()
    dm.dmChannels.forEach(ch => ws.subscribeToChannel(ch.id))
    await friendStore.loadAll()
})

function getPartner(channel) {
    const selfId = store.user?.id
    if (channel.participants) {
        const p = channel.participants.find(p => p.user_id !== selfId)
        return p?.user || {}
    }
    if (channel.user1?.id !== selfId) return channel.user1 || {}
    return channel.user2 || {}
}

async function selectUserForDM(userId) {
    showPicker.value = false
    await dm.openDMWith(userId)
    if (dm.currentDMChannel) ws.subscribeToChannel(dm.currentDMChannel.id)
    emit('channel-selected')
}

// ── Friend search ──
const friendSearchQuery = ref('')
const friendSearchResults = ref([])
const friendSearchLoading = ref(false)
const friendSearchError = ref('')
const friendAddSuccess = ref('')

let searchTimeout = null
watch(friendSearchQuery, (val) => {
    friendSearchError.value = ''
    friendAddSuccess.value = ''
    clearTimeout(searchTimeout)
    if (!val.trim()) { friendSearchResults.value = []; return }
    searchTimeout = setTimeout(async () => {
        friendSearchLoading.value = true
        try {
            const data = await api.searchUsers(val.trim())
            friendSearchResults.value = data.users ?? []
        } finally {
            friendSearchLoading.value = false
        }
    }, 300)
})

async function sendFriendRequest(username) {
    friendSearchError.value = ''
    friendAddSuccess.value = ''
    try {
        await friendStore.sendRequest(username)
        friendAddSuccess.value = t('dm.requestSentTo', { username })
    } catch (e) {
        friendSearchError.value = e?.message || t('dm.requestSendFailed')
    }
}

async function acceptFriend(userId) {
    await friendStore.acceptRequest(userId)
}

async function rejectFriend(userId) {
    await friendStore.rejectRequest(userId)
}

async function removeFriend(userId, name) {
    if (!confirm(t('dm.removeFriendConfirm', { name }))) return
    await friendStore.unfriend(userId)
}

// ── Helpers ──
function friendDisplayName(f) {
    const selfId = store.user?.id
    const other = f.requester_id === selfId ? f.addressee : f.requester
    return other?.nickname || other?.username || '?'
}
function friendAvatar(f) {
    const selfId = store.user?.id
    return (f.requester_id === selfId ? f.addressee : f.requester)?.avatar
}
function friendUserId(f) {
    const selfId = store.user?.id
    return f.requester_id === selfId ? f.addressee_id : f.requester_id
}
function requesterName(f) { return f.requester?.nickname || f.requester?.username || '?' }

const pendingCount = computed(() => friendStore.incomingRequests.length)
</script>

<template>
    <div class="dm-sidebar">
        <!-- Tab bar -->
        <div class="tab-bar">
            <button :class="['tab-btn', { active: activeTab === 'messages' }]" @click="activeTab = 'messages'">
                {{ t('dm.tabMessages') }}
            </button>
            <button :class="['tab-btn', { active: activeTab === 'friends' }]" @click="activeTab = 'friends'">
                {{ t('dm.tabFriends') }}
            </button>
            <button :class="['tab-btn requests-tab', { active: activeTab === 'requests' }]" @click="activeTab = 'requests'">
                {{ t('dm.tabRequests') }}
                <span v-if="pendingCount > 0" class="badge">{{ pendingCount }}</span>
            </button>
        </div>

        <!-- ══ Messages Tab ══ -->
        <template v-if="activeTab === 'messages'">
            <div class="section-header">
                <span>{{ t('dm.privateMessages') }}</span>
                <button class="icon-btn" :title="t('dm.newPrivateMessage')" @click="showPicker = !showPicker">
                    <i class="fas fa-plus"></i>
                </button>
            </div>

            <!-- DM picker: search from friends -->
            <div v-if="showPicker" class="picker">
                <input v-model="pickerQuery" class="search-input" :placeholder="t('dm.searchFriends')" autofocus />
                <div class="picker-list">
                    <div
                        v-for="f in friendStore.friends.filter(fr => {
                            const n = friendDisplayName(fr).toLowerCase()
                            return !pickerQuery || n.includes(pickerQuery.toLowerCase())
                        })"
                        :key="f.id"
                        class="picker-item"
                        @click="selectUserForDM(friendUserId(f))"
                    >
                        <div class="avatar sm">
                            <img v-if="friendAvatar(f)" :src="friendAvatar(f)" />
                            <span v-else>{{ friendDisplayName(f).charAt(0).toUpperCase() }}</span>
                        </div>
                        <span>{{ friendDisplayName(f) }}</span>
                    </div>
                    <div v-if="friendStore.friends.length === 0" class="empty-hint">{{ t('dm.noFriendsHint') }}</div>
                </div>
            </div>

            <div class="list">
                <div
                    v-for="ch in dm.dmChannels"
                    :key="ch.id"
                    :class="['list-item', { active: dm.currentDMChannel?.id === ch.id }]"
                    @click="dm.openDMChannel(ch); emit('channel-selected')"
                >
                    <div class="avatar sm">
                        <img v-if="getPartner(ch).avatar" :src="getPartner(ch).avatar" />
                        <span v-else>{{ (getPartner(ch).username || '?').charAt(0).toUpperCase() }}</span>
                    </div>
                    <span class="name">{{ getPartner(ch).nickname || getPartner(ch).username }}</span>
                    <span
                        v-if="store.channelUnreadMap.get(ch.id)?.mention > 0"
                        class="badge-mention"
                    >{{ store.channelUnreadMap.get(ch.id).mention }}</span>
                    <span
                        v-else-if="store.channelUnreadMap.get(ch.id)?.unread > 0"
                        class="badge-unread"
                    ></span>
                </div>
                <div v-if="dm.dmChannels.length === 0" class="empty-hint">{{ t('dm.noPrivateMessages') }}</div>
            </div>
        </template>

        <!-- ══ Friends Tab ══ -->
        <template v-else-if="activeTab === 'friends'">
            <div class="section-header">
                <span>{{ t('dm.searchUsers') }}</span>
            </div>
            <div class="friend-search-box">
                <input v-model="friendSearchQuery" class="search-input" :placeholder="t('dm.searchUsersPlaceholder')" />
                <p v-if="friendSearchError" class="error-text">{{ friendSearchError }}</p>
                <p v-if="friendAddSuccess" class="success-text">{{ friendAddSuccess }}</p>
                <div v-if="friendSearchResults.length > 0" class="picker-list">
                    <div v-for="u in friendSearchResults" :key="u.id" class="picker-item result-item">
                        <div class="avatar sm">
                            <img v-if="u.avatar" :src="u.avatar" />
                            <span v-else>{{ (u.username || '?').charAt(0).toUpperCase() }}</span>
                        </div>
                        <span class="flex-1">{{ u.nickname || u.username }}</span>
                        <button class="add-btn" @click="sendFriendRequest(u.username)">{{ t('dm.addFriend') }}</button>
                    </div>
                </div>
                <div v-else-if="friendSearchQuery && !friendSearchLoading" class="empty-hint">{{ t('dm.userNotFound') }}</div>
            </div>

            <div class="section-header" style="margin-top: 8px">
                <span>{{ t('dm.friendsCount', { count: friendStore.friends.length }) }}</span>
            </div>
            <div class="list">
                <div v-for="f in friendStore.friends" :key="f.id" class="list-item">
                    <div class="avatar sm">
                        <img v-if="friendAvatar(f)" :src="friendAvatar(f)" />
                        <span v-else>{{ friendDisplayName(f).charAt(0).toUpperCase() }}</span>
                    </div>
                    <span class="name flex-1">{{ friendDisplayName(f) }}</span>
                    <button class="icon-btn danger" :title="t('dm.unfriend')" @click="removeFriend(friendUserId(f), friendDisplayName(f))">
                        <i class="fas fa-user-minus"></i>
                    </button>
                    <button class="icon-btn" :title="t('dm.sendMessage')" @click="selectUserForDM(friendUserId(f)); activeTab = 'messages'">
                        <i class="fas fa-comment"></i>
                    </button>
                </div>
                <div v-if="friendStore.friends.length === 0 && !friendStore.loading" class="empty-hint">{{ t('dm.noFriends') }}</div>
            </div>
        </template>

        <!-- ══ Requests Tab ══ -->
        <template v-else-if="activeTab === 'requests'">
            <div class="section-header">
                <span>{{ t('dm.receivedRequestsCount', { count: friendStore.incomingRequests.length }) }}</span>
            </div>
            <div class="list">
                <div v-for="f in friendStore.incomingRequests" :key="f.id" class="list-item">
                    <div class="avatar sm">
                        <img v-if="f.requester?.avatar" :src="f.requester.avatar" />
                        <span v-else>{{ requesterName(f).charAt(0).toUpperCase() }}</span>
                    </div>
                    <span class="name flex-1">{{ requesterName(f) }}</span>
                    <button class="icon-btn accent" :title="t('dm.accept')" @click="acceptFriend(f.requester_id)">
                        <i class="fas fa-check"></i>
                    </button>
                    <button class="icon-btn danger" :title="t('dm.reject')" @click="rejectFriend(f.requester_id)">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div v-if="friendStore.incomingRequests.length === 0" class="empty-hint">{{ t('dm.noPendingRequests') }}</div>
            </div>

            <div class="section-header" style="margin-top: 8px">
                <span>{{ t('dm.sentRequestsCount', { count: friendStore.outgoingRequests.length }) }}</span>
            </div>
            <div class="list">
                <div v-for="f in friendStore.outgoingRequests" :key="f.id" class="list-item">
                    <div class="avatar sm">
                        <img v-if="f.addressee?.avatar" :src="f.addressee.avatar" />
                        <span v-else>{{ (f.addressee?.nickname || f.addressee?.username || '?').charAt(0).toUpperCase() }}</span>
                    </div>
                    <span class="name flex-1">{{ f.addressee?.nickname || f.addressee?.username }}</span>
                    <span class="muted-text">{{ t('dm.pending') }}</span>
                </div>
                <div v-if="friendStore.outgoingRequests.length === 0" class="empty-hint">{{ t('dm.noSentRequests') }}</div>
            </div>
        </template>

        <UserPanel />
    </div>
</template>

<style scoped>
.dm-sidebar {
    width: 240px;
    background: var(--bg-secondary, #2f3136);
    display: flex;
    flex-direction: column;
    height: 100%;
}

/* ── Tabs ── */
.tab-bar {
    display: flex;
    border-bottom: 1px solid var(--border-color, #40444b);
    flex-shrink: 0;
}
.tab-btn {
    flex: 1;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    padding: 10px 4px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted, #8e9297);
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    position: relative;
    transition: color 0.15s, border-color 0.15s;
}
.tab-btn:hover { color: var(--text-primary, #dcddde); }
.tab-btn.active { color: var(--text-primary, #dcddde); border-bottom-color: var(--accent, #5865f2); }
.badge {
    background: var(--accent-container);
    color: #fff;
    border-radius: var(--radius-lg);
    font-family: var(--font-mono);
    font-size: 10px;
    padding: 1px 5px;
    margin-left: 4px;
    font-weight: 500;
}

/* ── Section header ── */
.section-header {
    padding: 10px 16px 4px;
    font-size: 11px;
    font-weight: 700;
    color: var(--text-muted, #8e9297);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

/* ── Search inputs ── */
.search-input {
    width: 100%;
    background: var(--bg-primary, #36393f);
    border: 1px solid var(--border-color, #40444b);
    border-radius: var(--radius);
    padding: 6px 8px;
    color: var(--text-primary, #dcddde);
    font-size: 13px;
    outline: none;
    box-sizing: border-box;
}
.search-input::placeholder { color: var(--text-muted, #8e9297); }
.search-input:focus { border-color: var(--accent, #5865f2); }

.friend-search-box {
    padding: 0 10px 8px;
}

/* ── Picker ── */
.picker {
    border-bottom: 1px solid var(--border-color, #40444b);
    padding: 8px 10px;
    background: var(--bg-tertiary, #292b2f);
}
.picker-list {
    max-height: 180px;
    overflow-y: auto;
    margin-top: 4px;
}
.picker-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 4px;
    border-radius: var(--radius);
    cursor: pointer;
    color: var(--text-muted, #8e9297);
    transition: background 0.1s, color 0.1s;
}
.picker-item:hover { background: var(--bg-hover, #393c43); color: var(--text-primary, #dcddde); }
.result-item { cursor: default; }
.result-item:hover { background: transparent; color: var(--text-muted, #8e9297); }

/* ── List ── */
.list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
}
.list-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 12px;
    border-radius: var(--radius);
    margin: 1px 6px;
    cursor: pointer;
    color: var(--text-muted, #8e9297);
    transition: background 0.1s, color 0.1s;
}
.list-item:hover, .list-item.active {
    background: var(--bg-hover, #393c43);
    color: var(--text-primary, #dcddde);
}

/* ── Avatar ── */
.avatar {
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--accent, #5865f2);
    color: #fff;
    font-weight: 600;
}
.avatar.sm { width: 30px; height: 30px; font-size: 13px; }
.avatar img { width: 100%; height: 100%; object-fit: cover; }

/* ── Misc ── */
.name { font-size: 14px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.flex-1 { flex: 1; min-width: 0; }
.muted-text { font-size: 11px; color: var(--text-muted, #8e9297); }
.empty-hint { padding: 10px 16px; font-size: 12px; color: var(--text-muted, #8e9297); text-align: center; }
.error-text { color: var(--danger); font-size: 12px; margin: 4px 0; }
.success-text { color: var(--success); font-size: 12px; margin: 4px 0; }

.icon-btn {
    background: none;
    border: none;
    color: var(--text-muted, #8e9297);
    cursor: pointer;
    padding: 3px 5px;
    border-radius: var(--radius);
    font-size: 13px;
    transition: color 0.15s, background 0.15s;
}
.icon-btn:hover { color: var(--text-primary, #dcddde); background: var(--bg-hover, #393c43); }
.icon-btn.danger:hover { color: var(--danger); }
.icon-btn.accent:hover { color: var(--success); }
.add-btn {
    background: var(--accent, #5865f2);
    color: #fff;
    border: none;
    border-radius: var(--radius);
    padding: 3px 8px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
    transition: background 0.15s;
}
.add-btn:hover { background: var(--accent-hover); }

/* ── Mobile ── */
@media (max-width: 768px) {
    /* Sits beside the nav rail, which slides in with it (see main.css) */
    .dm-sidebar {
        position: fixed;
        left: 56px;
        top: 0;
        bottom: 0;
        width: min(300px, 85vw - 56px);
        height: 100%;
        transform: translateX(calc(-100% - 56px));
        transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
        z-index: 501;
        box-shadow: 4px 0 24px rgba(0, 0, 0, 0.45);
    }
    .dm-sidebar.mobile-open { transform: translateX(0); }
}
</style>
