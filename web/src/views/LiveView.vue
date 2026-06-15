<script setup>
import { useI18n } from 'vue-i18n'
// Future: Live streaming / broadcast view
// Planned features:
//  - One-to-many video/audio broadcast (host → audience)
//  - Screen share with live annotation
//  - Chat overlay during stream (scrolling live chat)
//  - Audience reaction system (emoji burst)
//  - Recording & VOD playback
//  - Co-host / panelist raise-hand system
//  - Integration with voice channels (upgrade to broadcast mode)
//  - Stream scheduling & calendar
const { t } = useI18n()
</script>

<template>
  <div class="stub-view">
    <div class="stub-view__inner">
      <div class="stub-view__icon live">
        <i class="fas fa-tower-broadcast"></i>
      </div>
      <h2>{{ t('views.live.title') }}</h2>
      <p>{{ t('views.live.subtitle') }}</p>
      <div class="stub-tags">
        <span class="stub-tag">{{ t('views.live.tagStreaming') }}</span>
        <span class="stub-tag">{{ t('views.live.tagScreenShare') }}</span>
        <span class="stub-tag">{{ t('views.live.tagLiveChat') }}</span>
        <span class="stub-tag">{{ t('views.live.tagReactions') }}</span>
        <span class="stub-tag">{{ t('views.live.tagVod') }}</span>
      </div>
    </div>
  </div>
</template>
