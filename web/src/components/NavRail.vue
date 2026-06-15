<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useAppStore } from '@/stores/useAppStore.js'
import { useDMStore } from '@/stores/useDMStore.js'

const emit = defineEmits(['create-guild', 'join-guild'])
const store = useAppStore()
const dm = useDMStore()
const { t } = useI18n()
const router = useRouter()
const route  = useRoute()

const currentSection = computed(() => route.meta?.section || 'chat')

const sections = computed(() => [
  { key: 'chat',  icon: 'fa-comments',        label: t('nav.chat'),  to: '/'      },
  { key: 'feed',  icon: 'fa-rss',             label: t('nav.feed'),  to: '/feed'  },
  { key: 'learn', icon: 'fa-graduation-cap',  label: t('nav.learn'), to: '/learn' },
  { key: 'live',  icon: 'fa-tower-broadcast', label: t('nav.live'),  to: '/live'  },
])

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
        <button class="nav-guild-btn home" :title="t('nav.home')" @click="dm.exitDMMode(); store.currentGuild = null">
          <i class="fas fa-home"></i>
        </button>
        <button
          :class="['nav-guild-btn', 'home', { active: dm.isDMMode }]"
          :title="t('nav.privateMessages')"
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
          <!-- Mention badge (red @, highest priority) -->
          <span
            v-else-if="store.mentionGuildIds.has(guild.id)"
            class="guild-mention-badge"
          >@</span>
          <!-- Unread dot (white bar, lower priority) -->
          <span
            v-else-if="store.unreadGuildIds.has(guild.id)"
            class="guild-unread-dot"
          ></span>
        </button>
        <button class="nav-guild-btn add" :title="t('nav.createGuild')" @click="emit('create-guild')">
          <i class="fas fa-plus"></i>
        </button>
        <button class="nav-guild-btn add" :title="t('nav.joinGuild')" @click="emit('join-guild')">
          <i class="fas fa-link"></i>
        </button>
      </div>
    </Transition>
  </nav>
</template>
