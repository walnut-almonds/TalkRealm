<script setup>
import { ref, computed, onMounted } from 'vue'
import { useDMStore } from '@/stores/useDMStore.js'
import { useAppStore } from '@/stores/useAppStore.js'
import { useWebSocket } from '@/composables/useWebSocket.js'
import UserPanel from './UserPanel.vue'

const dm = useDMStore()
const store = useAppStore()
const ws = useWebSocket()

const showPicker = ref(false)
const searchQuery = ref('')

onMounted(async () => {
    await dm.loadDMChannels()
    dm.dmChannels.forEach(ch => ws.subscribeToChannel(ch.id))
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

// Members that can be DM'd (exclude self, filter by search)
const pickableMembers = computed(() => {
    const selfId = store.user?.id
    return store.members
        .filter(m => m.user_id !== selfId && m.user)
        .filter(m => {
            if (!searchQuery.value) return true
            const q = searchQuery.value.toLowerCase()
            const u = m.user
            return (u.username || '').toLowerCase().includes(q) ||
                   (u.display_name || '').toLowerCase().includes(q)
        })
})

function openPicker() {
    searchQuery.value = ''
    showPicker.value = true
}

async function selectUser(member) {
    showPicker.value = false
    await dm.openDMWith(member.user_id)
    // subscribe to the new DM channel if not already
    if (dm.currentDMChannel) ws.subscribeToChannel(dm.currentDMChannel.id)
}
</script>

<template>
    <div class="dm-sidebar">
        <div class="dm-header">
            <h2>私人訊息</h2>
            <button class="new-dm-btn" title="新增私人訊息" @click="openPicker">
                <i class="fas fa-plus"></i>
            </button>
        </div>

        <!-- New DM picker -->
        <div v-if="showPicker" class="dm-picker">
            <div class="dm-picker-header">
                <input
                    v-model="searchQuery"
                    class="dm-search-input"
                    placeholder="搜尋使用者..."
                    autofocus
                />
                <button class="dm-picker-close" @click="showPicker = false">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="dm-picker-list">
                <div
                    v-for="m in pickableMembers"
                    :key="m.user_id"
                    class="dm-picker-item"
                    @click="selectUser(m)"
                >
                    <div class="dm-avatar">
                        <img v-if="m.user.avatar" :src="m.user.avatar" :alt="m.user.username" />
                        <span v-else class="dm-avatar-initial">{{ (m.user.username || '?').charAt(0).toUpperCase() }}</span>
                    </div>
                    <span class="dm-username">{{ m.user.display_name || m.user.username }}</span>
                </div>
                <div v-if="pickableMembers.length === 0 && store.members.length === 0" class="dm-empty">
                    <p>請先切換至一個社群以載入成員</p>
                </div>
                <div v-else-if="pickableMembers.length === 0" class="dm-empty">
                    <p>找不到使用者</p>
                </div>
            </div>
        </div>

        <div class="dm-list">
            <div
                v-for="ch in dm.dmChannels"
                :key="ch.id"
                :class="['dm-item', { active: dm.currentDMChannel?.id === ch.id }]"
                @click="dm.openDMChannel(ch)"
            >
                <div class="dm-avatar">
                    <img v-if="getPartner(ch).avatar" :src="getPartner(ch).avatar" :alt="getPartner(ch).username" />
                    <span v-else class="dm-avatar-initial">{{ (getPartner(ch).username || '?').charAt(0).toUpperCase() }}</span>
                </div>
                <span class="dm-username">{{ getPartner(ch).display_name || getPartner(ch).username }}</span>
            </div>
            <div v-if="dm.dmChannels.length === 0 && !showPicker" class="dm-empty">
                <p>沒有私人訊息</p>
            </div>
        </div>
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

.dm-header {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color, #40444b);
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.dm-header h2 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #dcddde);
    text-transform: uppercase;
    letter-spacing: 0.02em;
}

.new-dm-btn {
    background: none;
    border: none;
    color: var(--text-muted, #8e9297);
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 14px;
    line-height: 1;
    transition: color 0.15s, background 0.15s;
}
.new-dm-btn:hover {
    color: var(--text-primary, #dcddde);
    background: var(--bg-hover, #393c43);
}

.dm-picker {
    border-bottom: 1px solid var(--border-color, #40444b);
    background: var(--bg-tertiary, #292b2f);
}

.dm-picker-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 10px;
}

.dm-search-input {
    flex: 1;
    background: var(--bg-primary, #36393f);
    border: 1px solid var(--border-color, #40444b);
    border-radius: 4px;
    padding: 5px 8px;
    color: var(--text-primary, #dcddde);
    font-size: 13px;
    outline: none;
}
.dm-search-input::placeholder { color: var(--text-muted, #8e9297); }
.dm-search-input:focus { border-color: var(--accent, #5865f2); }

.dm-picker-close {
    background: none;
    border: none;
    color: var(--text-muted, #8e9297);
    cursor: pointer;
    padding: 4px;
    font-size: 13px;
    border-radius: 4px;
    transition: color 0.15s;
}
.dm-picker-close:hover { color: var(--text-primary, #dcddde); }

.dm-picker-list {
    max-height: 200px;
    overflow-y: auto;
    padding: 4px 0 8px;
}

.dm-picker-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 16px;
    cursor: pointer;
    color: var(--text-muted, #8e9297);
    transition: background 0.1s, color 0.1s;
}
.dm-picker-item:hover {
    background: var(--bg-hover, #393c43);
    color: var(--text-primary, #dcddde);
}

.dm-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
}

.dm-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 16px;
    cursor: pointer;
    border-radius: 4px;
    margin: 2px 8px;
    color: var(--text-muted, #8e9297);
    transition: background 0.1s, color 0.1s;
}

.dm-item:hover, .dm-item.active {
    background: var(--bg-hover, #393c43);
    color: var(--text-primary, #dcddde);
}

.dm-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--accent, #5865f2);
}

.dm-avatar img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.dm-avatar-initial {
    color: #fff;
    font-size: 14px;
    font-weight: 600;
}

.dm-username {
    font-size: 14px;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.dm-empty {
    padding: 16px;
    text-align: center;
    color: var(--text-muted, #8e9297);
    font-size: 13px;
}
</style>
