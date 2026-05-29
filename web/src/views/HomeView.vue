<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'

const store = useAppStore()

// ── Container size (responsive) ──────────────────────────────
const containerRef = ref(null)
const W = ref(800)
const H = ref(600)

function updateSize() {
    if (!containerRef.value) return
    W.value = containerRef.value.clientWidth || 800
    H.value = containerRef.value.clientHeight || 600
}

onMounted(() => {
    updateSize()
    window.addEventListener('resize', updateSize)
})
onBeforeUnmount(() => window.removeEventListener('resize', updateSize))

// ── Layout helpers ────────────────────────────────────────────
const cx = computed(() => W.value / 2)
const cy = computed(() => H.value / 2)
const orbitR = computed(() => Math.min(W.value, H.value) * 0.30)

const displayName = computed(() => store.user?.nickname || store.user?.username || '')

// ── Stars – generated once, never regenerated ─────────────────
const stars = ref(
    Array.from({ length: 160 }, () => ({
        x: (Math.random() * 100).toFixed(2),
        y: (Math.random() * 100).toFixed(2),
        r: (Math.random() * 1.6 + 0.4).toFixed(2),
        op: (Math.random() * 0.55 + 0.15).toFixed(2),
        dur: (Math.random() * 3 + 2).toFixed(1),
        del: (Math.random() * 6).toFixed(1),
    })),
)

// ── Per-guild random float animation params (stable) ─────────
const floatProps = new Map() // guildId → { delay, dur }
watch(
    () => store.guilds,
    (guilds) => {
        guilds.forEach((g) => {
            if (!floatProps.has(g.id)) {
                floatProps.set(g.id, {
                    delay: (Math.random() * 5).toFixed(1),
                    dur: (3.5 + Math.random() * 2.5).toFixed(1),
                })
            }
        })
    },
    { immediate: true },
)

// ── Guild nodes positioned on a circle ───────────────────────
const guildNodes = computed(() => {
    const count = store.guilds.length
    if (count === 0) return []
    return store.guilds.map((guild, i) => {
        const angle = (i / count) * 2 * Math.PI - Math.PI / 2
        const fp = floatProps.get(guild.id) || { delay: '0', dur: '4' }
        return {
            ...guild,
            x: cx.value + orbitR.value * Math.cos(angle),
            y: cy.value + orbitR.value * Math.sin(angle),
            floatDelay: fp.delay,
            floatDur: fp.dur,
            isOnline: store.onlineUserIds?.has(guild.id), // may be undefined; handled gracefully
        }
    })
})

function goToGuild(guildId) {
    store.selectGuild(guildId)
}
</script>

<template>
    <div class="galaxy-view" ref="containerRef">
        <!-- ── Starfield ───────────────────────────────────── -->
        <svg class="sg-stars" aria-hidden="true">
            <circle
                v-for="(s, i) in stars"
                :key="i"
                :cx="`${s.x}%`"
                :cy="`${s.y}%`"
                :r="s.r"
                fill="white"
                :style="`--op:${s.op}; animation: sg-twinkle ${s.dur}s ease-in-out ${s.del}s infinite alternate`"
            />
        </svg>

        <!-- ── Main galaxy SVG ─────────────────────────────── -->
        <svg
            class="sg-canvas"
            :viewBox="`0 0 ${W} ${H}`"
            :width="W"
            :height="H"
            aria-label="Social Galaxy"
        >
            <defs>
                <!-- User avatar clip -->
                <clipPath id="sg-user-clip">
                    <circle :cx="cx" :cy="cy" r="26" />
                </clipPath>

                <!-- Guild avatar clips -->
                <clipPath
                    v-for="node in guildNodes"
                    :key="`clip-${node.id}`"
                    :id="`sg-guild-clip-${node.id}`"
                >
                    <circle :cx="node.x" :cy="node.y" r="18" />
                </clipPath>

                <!-- User glow radial gradient -->
                <radialGradient id="sg-user-glow" cx="50%" cy="50%" r="50%">
                    <stop offset="0%" stop-color="#818cf8" stop-opacity="0.4" />
                    <stop offset="100%" stop-color="#818cf8" stop-opacity="0" />
                </radialGradient>

                <!-- Per-guild connection line gradients -->
                <linearGradient
                    v-for="node in guildNodes"
                    :key="`grad-${node.id}`"
                    :id="`sg-line-grad-${node.id}`"
                    :x1="cx"
                    :y1="cy"
                    :x2="node.x"
                    :y2="node.y"
                    gradientUnits="userSpaceOnUse"
                >
                    <stop offset="0%" stop-color="#818cf8" stop-opacity="0.45" />
                    <stop offset="100%" stop-color="#a78bfa" stop-opacity="0.08" />
                </linearGradient>
            </defs>

            <!-- Orbit ring -->
            <circle
                :cx="cx"
                :cy="cy"
                :r="orbitR"
                fill="none"
                stroke="rgba(255,255,255,0.06)"
                stroke-width="1"
                stroke-dasharray="6 14"
                class="sg-orbit-ring"
            />

            <!-- Connection lines from center to each guild -->
            <line
                v-for="node in guildNodes"
                :key="`line-${node.id}`"
                :x1="cx"
                :y1="cy"
                :x2="node.x"
                :y2="node.y"
                :stroke="`url(#sg-line-grad-${node.id})`"
                stroke-width="1.5"
            />

            <!-- Guild nodes -->
            <g
                v-for="node in guildNodes"
                :key="node.id"
                class="sg-guild-node"
                :style="`--float-delay:${node.floatDelay}s; --float-dur:${node.floatDur}s`"
                @click="goToGuild(node.id)"
                role="button"
                :aria-label="node.name"
                tabindex="0"
                @keydown.enter="goToGuild(node.id)"
            >
                <!-- Halo glow -->
                <circle
                    :cx="node.x"
                    :cy="node.y"
                    r="32"
                    fill="rgba(139,92,246,0.1)"
                    class="sg-guild-halo"
                />
                <!-- Node circle -->
                <circle
                    :cx="node.x"
                    :cy="node.y"
                    r="22"
                    fill="#1a1740"
                    stroke="rgba(139,92,246,0.6)"
                    stroke-width="1.5"
                    class="sg-guild-bg"
                />
                <!-- Avatar image -->
                <image
                    v-if="node.icon"
                    :href="node.icon"
                    :x="node.x - 18"
                    :y="node.y - 18"
                    width="36"
                    height="36"
                    :clip-path="`url(#sg-guild-clip-${node.id})`"
                    preserveAspectRatio="xMidYMid slice"
                />
                <!-- Initial text fallback -->
                <text
                    v-else
                    :x="node.x"
                    :y="node.y"
                    text-anchor="middle"
                    dominant-baseline="middle"
                    fill="white"
                    font-size="14"
                    font-weight="600"
                    font-family="'Segoe UI', sans-serif"
                >{{ node.name?.charAt(0)?.toUpperCase() }}</text>
                <!-- Guild name label -->
                <text
                    :x="node.x"
                    :y="node.y + 37"
                    text-anchor="middle"
                    dominant-baseline="middle"
                    fill="rgba(255,255,255,0.6)"
                    font-size="11"
                    font-family="'Segoe UI', sans-serif"
                >{{ node.name?.length > 12 ? node.name.slice(0, 11) + '…' : node.name }}</text>
            </g>

            <!-- Center: current user node -->
            <g class="sg-user-node">
                <!-- Expanding pulse rings -->
                <circle :cx="cx" :cy="cy" r="52" fill="url(#sg-user-glow)" class="sg-pulse-outer" />
                <circle :cx="cx" :cy="cy" r="38" fill="rgba(99,102,241,0.18)" class="sg-pulse-inner" />
                <!-- User circle -->
                <circle
                    :cx="cx"
                    :cy="cy"
                    r="28"
                    fill="#1a1740"
                    stroke="#818cf8"
                    stroke-width="2"
                />
                <!-- User avatar -->
                <image
                    v-if="store.user?.avatar"
                    :href="store.user.avatar"
                    :x="cx - 26"
                    :y="cy - 26"
                    width="52"
                    height="52"
                    clip-path="url(#sg-user-clip)"
                    preserveAspectRatio="xMidYMid slice"
                />
                <!-- User initial fallback -->
                <text
                    v-else
                    :x="cx"
                    :y="cy"
                    text-anchor="middle"
                    dominant-baseline="middle"
                    fill="white"
                    font-size="20"
                    font-weight="700"
                    font-family="'Segoe UI', sans-serif"
                >{{ displayName?.charAt(0)?.toUpperCase() }}</text>
            </g>
        </svg>

        <!-- ── Welcome overlay ────────────────────────────── -->
        <div class="sg-welcome">
            <p class="sg-username">{{ displayName }}</p>
            <p v-if="store.guilds.length > 0" class="sg-hint">點擊節點進入社群</p>
            <p v-else class="sg-hint">在左側建立或加入一個社群</p>
        </div>
    </div>
</template>

<style scoped>
/* ── Container ─────────────────────────────────────────────── */
.galaxy-view {
    position: relative;
    flex: 1;
    overflow: hidden;
    background: #060918;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 0;
}

/* ── Starfield ─────────────────────────────────────────────── */
.sg-stars {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
}

@keyframes sg-twinkle {
    from { opacity: var(--op, 0.3); }
    to   { opacity: calc(var(--op, 0.3) * 0.15); }
}

/* ── Main canvas ───────────────────────────────────────────── */
.sg-canvas {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none; /* re-enabled per-element below */
    overflow: visible;
}

/* ── Orbit ring slow spin ──────────────────────────────────── */
.sg-orbit-ring {
    transform-box: fill-box;
    transform-origin: center;
    animation: sg-orbit-spin 120s linear infinite;
}

@keyframes sg-orbit-spin {
    from { transform: rotate(0deg); }
    to   { transform: rotate(360deg); }
}

/* ── Guild nodes ───────────────────────────────────────────── */
.sg-guild-node {
    cursor: pointer;
    pointer-events: all;
    transform-box: fill-box;
    transform-origin: center;
    animation: sg-float var(--float-dur, 4s) ease-in-out var(--float-delay, 0s) infinite alternate;
    transition: filter 0.2s ease;
    outline: none;
}

.sg-guild-node:hover,
.sg-guild-node:focus-visible {
    filter: brightness(1.35) drop-shadow(0 0 14px rgba(139, 92, 246, 0.65));
}

.sg-guild-node:hover .sg-guild-bg {
    stroke: rgba(167, 139, 250, 0.95);
}

@keyframes sg-float {
    from { transform: translateY(0px); }
    to   { transform: translateY(-9px); }
}

/* ── User center node ──────────────────────────────────────── */
.sg-user-node {
    pointer-events: none;
}

.sg-pulse-outer {
    transform-box: fill-box;
    transform-origin: center;
    animation: sg-pulse-out 3s ease-out infinite;
}

.sg-pulse-inner {
    transform-box: fill-box;
    transform-origin: center;
    animation: sg-pulse-out 3s ease-out 0.7s infinite;
}

@keyframes sg-pulse-out {
    0%   { transform: scale(1);   opacity: 0.35; }
    100% { transform: scale(1.5); opacity: 0; }
}

/* ── Welcome overlay ───────────────────────────────────────── */
.sg-welcome {
    position: absolute;
    bottom: 40px;
    left: 50%;
    transform: translateX(-50%);
    text-align: center;
    pointer-events: none;
    user-select: none;
}

.sg-username {
    font-size: 15px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.85);
    letter-spacing: 0.03em;
    margin-bottom: 4px;
}

.sg-hint {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.35);
    letter-spacing: 0.02em;
}
</style>
