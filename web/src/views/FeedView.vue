<script setup>
import { useI18n } from 'vue-i18n'
// Future: SNS / social feed view
// Planned features:
//  - Global/friend activity feed (posts, reactions, comments)
//  - Stories / short media clips
//  - Follow system (per-user follow, not guild-based)
//  - Trending topics / hashtags
//  - Cross-guild public posts
const { t } = useI18n()
</script>

<template>
  <div class="stub-view">
    <div class="stub-view__inner">
      <div class="stub-view__icon">
        <i class="fas fa-rss"></i>
      </div>
      <h2>{{ t('views.feed.title') }}</h2>
      <p>{{ t('views.feed.subtitle') }}</p>
      <div class="stub-tags">
        <span class="stub-tag">{{ t('views.feed.tagPosts') }}</span>
        <span class="stub-tag">{{ t('views.feed.tagStories') }}</span>
        <span class="stub-tag">{{ t('views.feed.tagFollow') }}</span>
        <span class="stub-tag">{{ t('views.feed.tagHashtags') }}</span>
        <span class="stub-tag">{{ t('views.feed.tagGuildActivity') }}</span>
      </div>
    </div>
  </div>
</template>
