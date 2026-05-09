<script setup>
import { ref } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import ChannelSidebar from '@/components/ChannelSidebar.vue'
import ChatArea from '@/components/ChatArea.vue'
import MemberSidebar from '@/components/MemberSidebar.vue'

const store = useAppStore()
const showMemberSidebar = ref(true)
</script>

<template>
  <div class="chat-view">
    <ChannelSidebar />
    <ChatArea @toggle-members="showMemberSidebar = !showMemberSidebar" />
    <MemberSidebar v-if="showMemberSidebar && store.currentGuild" />
  </div>
</template>
