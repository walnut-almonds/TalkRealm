<script setup>
import { useI18n } from 'vue-i18n'
// Future: Language Learning integration
// Planned features:
//  - Lesson browser (vocabulary, grammar, listening, speaking exercises)
//  - SRS flashcard system (spaced repetition)
//  - Voice practice with pronunciation scoring
//  - Language exchange rooms — pair users for text/voice conversation practice
//  - Progress tracking (streaks, XP, skill tree)
//  - Integrated with chat: "Practice mode" inside channels
const { t } = useI18n()
</script>

<template>
  <div class="stub-view">
    <div class="stub-view__inner">
      <div class="stub-view__icon learn">
        <i class="fas fa-graduation-cap"></i>
      </div>
      <h2>{{ t('views.learn.title') }}</h2>
      <p>{{ t('views.learn.subtitle') }}</p>
      <div class="stub-tags">
        <span class="stub-tag">{{ t('views.learn.tagFlashcards') }}</span>
        <span class="stub-tag">{{ t('views.learn.tagListening') }}</span>
        <span class="stub-tag">{{ t('views.learn.tagSpeakingScore') }}</span>
        <span class="stub-tag">{{ t('views.learn.tagExchange') }}</span>
        <span class="stub-tag">{{ t('views.learn.tagProgress') }}</span>
      </div>
    </div>
  </div>
</template>
