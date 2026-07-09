<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import WordFill from '@/components/learn/WordFill.vue'

const learn = useLearnStore()
const { t } = useI18n()

const tier = ref(2)
const playing = ref(false)
const tiers = [1, 2, 3, 4, 5]
// check-i18n-keys 只認字面 key，動態 tier key 走查表
const tierKeys = ['learn.tier1', 'learn.tier2', 'learn.tier3', 'learn.tier4', 'learn.tier5']

onMounted(() => learn.loadStats())

async function start(mode) {
    await learn.startLevel(mode, tier.value)
    if (learn.level) playing.value = true
}

function exitGame() {
    playing.value = false
    learn.level = null
}
</script>

<template>
  <div class="learn-view">
    <WordFill v-if="playing && learn.level" @exit="exitGame" />

    <div v-else class="learn-hub">
      <header class="learn-hub__head">
        <h2>{{ t('learn.hubTitle') }}</h2>
        <div v-if="learn.stats" class="learn-stats">
          <span class="stat"><b>{{ learn.stats.xp }}</b> XP</span>
          <span class="stat"><b>{{ learn.stats.streak }}</b> {{ t('learn.streakDays') }}</span>
        </div>
      </header>

      <section class="learn-card">
        <h3>{{ t('learn.difficulty') }}</h3>
        <div class="tier-row">
          <button
            v-for="tv in tiers"
            :key="tv"
            :class="['tier-btn', { active: tier === tv }]"
            @click="tier = tv"
          >{{ t(tierKeys[tv - 1]) }}</button>
        </div>

        <h3>{{ t('learn.chooseMode') }}</h3>
        <div class="mode-row">
          <button class="mode-btn" :disabled="learn.loading" @click="start('fill')">
            <i class="fas fa-keyboard"></i>
            <span>{{ t('learn.modeFill') }}</span>
            <small>{{ t('learn.modeFillDesc') }}</small>
          </button>
          <!-- Phase 2 解鎖：字母盤模式按鈕 -->
        </div>

        <p v-if="learn.error" class="learn-error">{{ learn.error }}</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.learn-view { height: 100%; overflow-y: auto; }
.learn-hub { max-width: 640px; margin: 0 auto; padding: 32px 16px; display: flex; flex-direction: column; gap: 24px; }
.learn-hub__head { display: flex; justify-content: space-between; align-items: baseline; }
.learn-stats { display: flex; gap: 16px; font-family: var(--font-mono); font-size: 13px; color: var(--text-muted); }
.learn-stats b { color: var(--accent); }
.learn-card {
    padding: 20px; background: var(--bg-secondary);
    border: 1px solid var(--border);
    display: flex; flex-direction: column; gap: 12px;
}
.tier-row, .mode-row { display: flex; gap: 8px; flex-wrap: wrap; }
.tier-btn {
    padding: 8px 14px; background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
    font-family: var(--font-mono); font-size: 12px;
}
.tier-btn.active { border-color: var(--accent); color: var(--accent); }
.mode-btn {
    flex: 1; min-width: 200px; padding: 16px; text-align: left;
    background: transparent; color: inherit;
    border: 1px solid var(--border); cursor: pointer;
    display: flex; flex-direction: column; gap: 6px;
}
.mode-btn:hover { border-color: var(--accent); }
.mode-btn small { color: var(--text-muted); }
.learn-error { color: var(--danger); font-size: 13px; }
</style>
