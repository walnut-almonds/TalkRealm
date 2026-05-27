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
const mobileMemberSidebarOpen = ref(false)

function toggleMembers() {
  if (window.innerWidth <= 768) {
    mobileMemberSidebarOpen.value = !mobileMemberSidebarOpen.value
  } else {
    showMemberSidebar.value = !showMemberSidebar.value
  }
}

// ── Swipe gesture (horizontal) ────────────────────────────
let touchStartX = 0
let touchStartY = 0

function onTouchStart(e) {
  touchStartX = e.touches[0].clientX
  touchStartY = e.touches[0].clientY
}

function onTouchEnd(e) {
  const dx = e.changedTouches[0].clientX - touchStartX
  const dy = e.changedTouches[0].clientY - touchStartY
  // Only handle mostly-horizontal swipes of 50px+
  if (Math.abs(dx) < 50 || Math.abs(dy) > Math.abs(dx) * 0.75) return

  if (dx < 0) {
    // Swipe left → close channel sidebar, OR open members sidebar
    if (mobileChannelSidebarOpen.value) {
      mobileChannelSidebarOpen.value = false
    } else if (!mobileMemberSidebarOpen.value && store.currentGuild) {
      mobileMemberSidebarOpen.value = true
    }
  } else {
    // Swipe right → close members sidebar, OR open channel sidebar
    if (mobileMemberSidebarOpen.value) {
      mobileMemberSidebarOpen.value = false
    } else if (!mobileChannelSidebarOpen.value) {
      mobileChannelSidebarOpen.value = true
    }
  }
}
</script>

<template>
  <div
    class="chat-view"
    @touchstart.passive="onTouchStart"
    @touchend.passive="onTouchEnd"
  >
    <!-- Mobile backdrop – closes channel sidebar on tap -->
    <Teleport to="body">
      <div
        v-if="mobileChannelSidebarOpen"
        class="mobile-sidebar-backdrop"
        @click="mobileChannelSidebarOpen = false"
      ></div>
    </Teleport>

    <!-- Mobile backdrop – closes members sidebar on tap -->
    <Teleport to="body">
      <div
        v-if="mobileMemberSidebarOpen"
        class="mobile-members-backdrop"
        @click="mobileMemberSidebarOpen = false"
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
        @toggle-members="toggleMembers"
        @open-sidebar="mobileChannelSidebarOpen = true"
      />
      <MemberSidebar
        v-if="store.currentGuild && (showMemberSidebar || mobileMemberSidebarOpen)"
        :class="{ 'mobile-open': mobileMemberSidebarOpen }"
        @close-mobile="mobileMemberSidebarOpen = false"
      />
    </template>
  </div>
</template>
