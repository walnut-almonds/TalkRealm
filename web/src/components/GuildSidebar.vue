<script setup>
import { useAppStore } from '@/stores/useAppStore.js'

const emit = defineEmits(['create-guild', 'join-guild'])
const store = useAppStore()

async function selectGuild(guildId) {
  await store.selectGuild(guildId)
}

function goHome() {
  store.currentGuild = null
  store.currentChannel = null
  store.messages = []
}
</script>

<template>
  <div class="guilds-sidebar">
    <div class="guild-item home" title="首頁" @click="goHome">
      <i class="fas fa-home"></i>
    </div>
    <div class="guilds-separator"></div>
    <div
      v-for="guild in store.guilds"
      :key="guild.id"
      :class="['guild-item', { active: store.currentGuild?.id === guild.id }]"
      :title="guild.name"
      @click="selectGuild(guild.id)"
    >
      <img v-if="guild.icon" :src="guild.icon" :alt="guild.name" />
      <span v-else>{{ guild.name.charAt(0).toUpperCase() }}</span>
    </div>
    <div class="guild-item add-guild" title="建立社群" @click="emit('create-guild')">
      <i class="fas fa-plus"></i>
    </div>
    <div class="guild-item join-guild" title="透過邀請碼加入" @click="emit('join-guild')">
      <i class="fas fa-link"></i>
    </div>
  </div>
</template>
