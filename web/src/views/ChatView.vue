<script setup>
import { ref } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useDMStore } from '@/stores/useDMStore.js'
import ChannelSidebar from '@/components/ChannelSidebar.vue'
import ChatArea from '@/components/ChatArea.vue'
import MemberSidebar from '@/components/MemberSidebar.vue'
import DMSidebar from '@/components/DMSidebar.vue'
import DMChatArea from '@/components/DMChatArea.vue'

const store = useAppStore()
const dm = useDMStore()
const showMemberSidebar = ref(true)
const mobileChannelSidebarOpen = ref(false)
</script>

<template>
  <div class="chat-view">
    <!-- Mobile backdrop – closes sidebar on tap -->
    <Teleport to="body">
      <div
        v-if="mobileChannelSidebarOpen"
        class="mobile-sidebar-backdrop"
        @click="mobileChannelSidebarOpen = false"
      ></div>
    </Teleport>

    <template v-if="dm.isDMMode">
      <DMSidebar />
      <DMChatArea />
    </template>
    <template v-else>
      <ChannelSidebar
        :class="{ 'mobile-open': mobileChannelSidebarOpen }"
        @channel-selected="mobileChannelSidebarOpen = false"
      />
      <ChatArea
        @toggle-members="showMemberSidebar = !showMemberSidebar"
        @open-sidebar="mobileChannelSidebarOpen = true"
      />
      <MemberSidebar v-if="showMemberSidebar && store.currentGuild" />
    </template>
  </div>
</template>
