<script setup>
import { onMounted } from 'vue'
import { useDMStore } from '@/stores/useDMStore.js'
import UserPanel from './UserPanel.vue'

const dm = useDMStore()

onMounted(() => {
    dm.loadDMChannels()
})

function getPartner(channel) {
    // Returns the "other" user relative to the current user
    // channel.user1 and channel.user2 are populated by the API
    return channel.user1 || channel.user2 || {}
}
</script>

<template>
    <div class="dm-sidebar">
        <div class="dm-header">
            <h2>私人訊息</h2>
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
            <div v-if="dm.dmChannels.length === 0" class="dm-empty">
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
    padding: 16px;
    border-bottom: 1px solid var(--border-color, #40444b);
}

.dm-header h2 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #dcddde);
    text-transform: uppercase;
    letter-spacing: 0.02em;
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
