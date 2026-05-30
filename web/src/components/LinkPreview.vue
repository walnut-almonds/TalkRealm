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

// Module-level cache — persists for the lifetime of the page so the same URL
// is never fetched more than once per session.
// null  = fetch returned no usable data (don't retry)
// false = fetch in-flight (deduplicate concurrent requests)
// OGData = cached result
const ogCache = new Map()
const inflight = new Map()

async function fetchOG(url) {
  if (ogCache.has(url)) return ogCache.get(url)

  // Deduplicate concurrent requests for the same URL.
  if (inflight.has(url)) return inflight.get(url)

  const promise = api.getOGPreview(url)
    .then((data) => {
      const result = (data?.title || data?.description || data?.image)
        ? { ...data, _src: url }
        : null
      ogCache.set(url, result)
      inflight.delete(url)
      return result
    })
    .catch(() => {
      // Cache failures too so we don't hammer a server that keeps returning 4xx.
      ogCache.set(url, null)
      inflight.delete(url)
      return null
    })

  inflight.set(url, promise)
  return promise
}

const previews = ref([])

watch(
  () => props.urls,
  async (urls) => {
    if (!urls?.length) { previews.value = []; return }
    const targets = urls.slice(0, MAX_PREVIEWS)
    const results = await Promise.all(targets.map(fetchOG))
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
