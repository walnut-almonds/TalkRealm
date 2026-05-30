<script setup>
import { ref, watch } from 'vue'
import { api } from '@/api/index.js'

const props = defineProps({
  urls: {
    type: Array,
    default: () => [],
  },
})

// Show at most 1 preview per message to avoid clutter.
const MAX_PREVIEWS = 1

const previews = ref([])

watch(
  () => props.urls,
  async (urls) => {
    if (!urls?.length) { previews.value = []; return }
    const targets = urls.slice(0, MAX_PREVIEWS)
    const results = await Promise.all(
      targets.map(async (url) => {
        try {
          const data = await api.getOGPreview(url)
          if (!data?.title && !data?.description && !data?.image) return null
          return { ...data, _src: url }
        } catch {
          return null
        }
      }),
    )
    previews.value = results.filter(Boolean)
  },
  { immediate: true },
)
</script>

<template>
  <div v-if="previews.length" class="og-previews">
    <a
      v-for="og in previews"
      :key="og._src"
      :href="og._src"
      target="_blank"
      rel="noopener noreferrer"
      class="og-card"
    >
      <img v-if="og.image" :src="og.image" class="og-card__image" alt="" loading="lazy" />
      <div class="og-card__body">
        <div v-if="og.site_name" class="og-card__site">{{ og.site_name }}</div>
        <div v-if="og.title" class="og-card__title">{{ og.title }}</div>
        <div v-if="og.description" class="og-card__desc">{{ og.description }}</div>
      </div>
    </a>
  </div>
</template>
