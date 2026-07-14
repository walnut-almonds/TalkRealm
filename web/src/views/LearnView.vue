<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLearnStore } from '@/stores/useLearnStore.js'
import WordFill from '@/components/learn/WordFill.vue'
import LetterWheel from '@/components/learn/LetterWheel.vue'
import Crossword from '@/components/learn/Crossword.vue'

const learn = useLearnStore()
const { t } = useI18n()

const tier = ref(2)
const playing = ref(false)
const tiers = [1, 2, 3, 4, 5]
// check-i18n-keys 只認字面 key，動態 tier key 走查表
const tierKeys = ['learn.tier1', 'learn.tier2', 'learn.tier3', 'learn.tier4', 'learn.tier5']

onMounted(() => { learn.loadStats(); learn.loadDaily(); learn.loadLeaderboard() })

async function start(mode) {
    await learn.startLevel(mode, tier.value)
    if (learn.level) playing.value = true
}

async function startDaily() {
    if (await learn.startDaily()) playing.value = true
}

async function startCrosswordMode() {
    await learn.startCrossword(tier.value)
    if (learn.crossword) playing.value = true
}

function exitGame() {
    playing.value = false
    learn.level = null
    learn.crossword = null
}
</script>

<template>
  <div class="learn-view">
    <div v-if="playing && (learn.level || learn.crossword)" class="learn-game">
      <div class="learn-game__bar">
        <button class="exit-btn" @click="exitGame">
          <i class="fas fa-arrow-left"></i>
          <span>{{ t('learn.backToHub') }}</span>
        </button>
        <button
          :class="['hard-toggle', { on: learn.hardMode }]"
          :title="t('learn.hardModeDesc')"
          @click="learn.setHardMode(!learn.hardMode)"
        >
          <i class="fas fa-eye-slash"></i>
          <span>{{ t('learn.hardMode') }}</span>
        </button>
      </div>
      <WordFill v-if="learn.level?.mode === 'fill'" @exit="exitGame" />
      <LetterWheel v-else-if="learn.level?.mode === 'wheel'" @exit="exitGame" />
      <Crossword v-else-if="learn.crossword" @exit="exitGame" />
    </div>

    <div v-else class="learn-hub">
      <header class="learn-hub__head">
        <h2>{{ t('learn.hubTitle') }}</h2>
        <div v-if="learn.stats" class="learn-stats">
          <span class="stat"><b>{{ learn.stats.xp }}</b> XP</span>
          <span class="stat"><b>{{ learn.stats.streak }}</b> {{ t('learn.streakDays') }}</span>
        </div>
      </header>

      <section class="learn-card daily-card">
        <div class="daily-head">
          <h3>{{ t('learn.daily') }}</h3>
          <span class="daily-date">{{ learn.daily?.date }}</span>
        </div>
        <button
          v-if="learn.daily && !learn.daily.played"
          class="mode-btn"
          @click="startDaily"
        >
          <i class="fas fa-calendar-day"></i>
          <span>{{ t('learn.dailyStart') }}</span>
        </button>
        <p v-else-if="learn.daily?.played" class="daily-done">
          {{ t('learn.dailyPlayed') }} — <b class="mono">{{ learn.daily.score }}</b> {{ t('learn.score') }}
        </p>

        <div v-if="learn.leaderboard?.top?.length" class="lb">
          <h4>{{ t('learn.leaderboard') }}</h4>
          <ol class="lb-list">
            <li v-for="e in learn.leaderboard.top" :key="e.user_id" class="lb-row">
              <span class="lb-rank mono">{{ e.rank }}</span>
              <img v-if="e.avatar" :src="e.avatar" class="lb-avatar" />
              <span class="lb-name">{{ e.username || `#${e.user_id}` }}</span>
              <span class="lb-score mono">{{ e.score }}</span>
            </li>
          </ol>
          <p v-if="learn.leaderboard.me" class="lb-me">
            {{ t('learn.rank') }} <b class="mono">{{ learn.leaderboard.me.rank }}</b>
            · <b class="mono">{{ learn.leaderboard.me.score }}</b> {{ t('learn.score') }}
          </p>
        </div>
      </section>

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
        <button
          :class="['hard-toggle', { on: learn.hardMode }]"
          @click="learn.setHardMode(!learn.hardMode)"
        >
          <i class="fas fa-eye-slash"></i>
          <span>{{ t('learn.hardMode') }}</span>
          <small>{{ t('learn.hardModeDesc') }}</small>
        </button>

        <h3>{{ t('learn.chooseMode') }}</h3>
        <div class="mode-row">
          <button class="mode-btn" :disabled="learn.loading" @click="start('fill')">
            <i class="fas fa-keyboard"></i>
            <span>{{ t('learn.modeFill') }}</span>
            <small>{{ t('learn.modeFillDesc') }}</small>
          </button>
          <button class="mode-btn" :disabled="learn.loading" @click="start('wheel')">
            <i class="fas fa-circle-nodes"></i>
            <span>{{ t('learn.modeWheel') }}</span>
            <small>{{ t('learn.modeWheelDesc') }}</small>
          </button>
          <button class="mode-btn" :disabled="learn.loading" @click="startCrosswordMode">
            <i class="fas fa-border-all"></i>
            <span>{{ t('learn.modeCrossword') }}</span>
            <small>{{ t('learn.modeCrosswordDesc') }}</small>
          </button>
        </div>

        <p v-if="learn.error" class="learn-error">{{ learn.error }}</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.learn-view { height: 100%; overflow-y: auto; }
.learn-game__bar { max-width: 560px; margin: 0 auto; padding: 16px 16px 0; display: flex; justify-content: space-between; gap: 8px; }
.exit-btn {
    display: inline-flex; align-items: center; gap: 8px;
    padding: 8px 14px; background: transparent; color: var(--text-muted);
    border: 1px solid var(--border); cursor: pointer;
    font-family: var(--font-mono); font-size: 12px;
}
.exit-btn:hover { border-color: var(--accent); color: var(--accent); }
.hard-toggle {
    display: inline-flex; align-items: center; gap: 8px;
    padding: 8px 14px; background: transparent; color: var(--text-muted);
    border: 1px solid var(--border); cursor: pointer;
    font-family: var(--font-mono); font-size: 12px;
    align-self: flex-start;
}
.hard-toggle.on { border-color: var(--accent); color: var(--accent); }
.hard-toggle small { color: var(--text-muted); font-size: 11px; }
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
.daily-head { display: flex; justify-content: space-between; align-items: baseline; }
.daily-date, .mono { font-family: var(--font-mono); font-size: 12px; color: var(--text-muted); }
.daily-done b, .lb-me b { color: var(--accent); }
.lb-list { list-style: none; padding: 0; margin: 8px 0 0; display: flex; flex-direction: column; gap: 6px; }
.lb-row { display: flex; align-items: center; gap: 10px; padding: 6px 8px; border: 1px solid var(--border); }
.lb-rank { width: 20px; text-align: right; }
.lb-avatar { width: 20px; height: 20px; border-radius: 50%; } /* 人=圓 */
.lb-name { flex: 1; font-size: 13px; }
.lb-me { margin-top: 8px; font-size: 13px; }
</style>
