<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAppStore } from '@/stores/useAppStore.js'
import { useDMStore } from '@/stores/useDMStore.js'

const emit = defineEmits(['create-guild', 'join-guild'])
const store = useAppStore()
const dm = useDMStore()
const router = useRouter()
const route  = useRoute()

const currentSection = computed(() => route.meta?.section || 'chat')

const sections = [
  { key: 'chat',  icon: 'fa-comments',   label: '聊天',     to: '/'       },
  { key: 'feed',  icon: 'fa-rss',        label: '動態',     to: '/feed'   },
  { key: 'learn', icon: 'fa-graduation-cap', label: '學習', to: '/learn'  },
  { key: 'live',  icon: 'fa-tower-broadcast', label: '直播', to: '/live'  },
]

function goTo(path) { router.push(path) }
function selectGuild(id) {
  dm.exitDMMode()
  store.selectGuild(id)
}

function openDM() {
  router.push('/')
  dm.isDMMode = true
  store.currentGuild = null
}
</script>

<template>
  <nav class="nav-rail">
    <!-- App sections -->
    <div class="nav-sections">
      <button
        v-for="s in sections"
        :key="s.key"
        :class="['nav-section-btn', { active: currentSection === s.key }]"
        :title="s.label"
        @click="goTo(s.to)"
      >
        <i :class="['fas', s.icon]"></i>
        <span class="nav-section-label">{{ s.label }}</span>
      </button>
    </div>

    <!-- Separator -->
    <div class="nav-separator"></div>

    <!-- Guild list (only in chat section) -->
    <Transition name="fade">
      <div v-if="currentSection === 'chat'" class="nav-guilds">
        <button class="nav-guild-btn home" title="首頁" @click="dm.exitDMMode(); store.currentGuild = null">
          <i class="fas fa-home"></i>
        </button>
        <button
          :class="['nav-guild-btn', 'home', { active: dm.isDMMode }]"
          title="私人訊息"
          @click="openDM()"
        >
          <i class="fas fa-envelope"></i>
        </button>
        <div class="nav-guilds-separator"></div>
        <button
          v-for="guild in store.guilds"
          :key="guild.id"
          :class="['nav-guild-btn', { active: store.currentGuild?.id === guild.id }]"
          :title="guild.name"
          @click="selectGuild(guild.id)"
        >
          <img v-if="guild.icon" :src="guild.icon" :alt="guild.name" />
          <span v-else class="guild-initial">{{ guild.name.charAt(0).toUpperCase() }}</span>
          <!-- Active indicator pip -->
          <span v-if="store.currentGuild?.id === guild.id" class="guild-pip"></span>
          <!-- Unread badge (dot when unread; not shown when guild is active) -->
          <span
            v-else-if="store.unreadGuildIds.has(guild.id)"
            class="guild-unread-dot"
          ></span>
        </button>
        <button class="nav-guild-btn add" title="建立社群" @click="emit('create-guild')">
          <i class="fas fa-plus"></i>
        </button>
        <button class="nav-guild-btn add" title="加入社群" @click="emit('join-guild')">
          <i class="fas fa-link"></i>
        </button>
      </div>
    </Transition>
  </nav>
</template>
