<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, reactive } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'
import { useDMStore } from '@/stores/useDMStore.js'
import { api } from '@/api/index.js'

const store = useAppStore()
const dm = useDMStore()

// ── Guild color palette (8 distinct colors) ──────────────────
const GUILD_PALETTE = [
    { hex: '#818cf8', rgb: '129,140,248' },  // indigo
    { hex: '#34d399', rgb: '52,211,153'  },  // emerald
    { hex: '#f472b6', rgb: '244,114,182' },  // pink
    { hex: '#fb923c', rgb: '251,146,60'  },  // orange
    { hex: '#22d3ee', rgb: '34,211,238'  },  // cyan
    { hex: '#fbbf24', rgb: '251,191,36'  },  // amber
    { hex: '#a78bfa', rgb: '167,139,250' },  // violet
    { hex: '#f87171', rgb: '248,113,113' },  // red
]

function guildPalette(guildId) {
    return GUILD_PALETTE[(guildId ?? 0) % GUILD_PALETTE.length]
}

function guildNodeRadius(guildId) {
    const count = interactionMap.get(guildId) || 0
    const t = Math.min(count / 80, 1)
    return Math.round(18 + t * 9) // 18..27
}

// ── Container size (responsive) ──────────────────────────────
const containerRef = ref(null)
const W = ref(800)
const H = ref(600)

function updateSize() {
    if (!containerRef.value) return
    const rect = containerRef.value.getBoundingClientRect()
    W.value = rect.width || 800
    H.value = rect.height || 600
    if (simNodes.length > 0) {
        simNodes[0].x = W.value / 2
        simNodes[0].y = H.value / 2
        simNodes[0].vx = 0
        simNodes[0].vy = 0
    }
}

onMounted(() => {
    updateSize()
    window.addEventListener('resize', updateSize)
})
onBeforeUnmount(() => {
    window.removeEventListener('resize', updateSize)
    stopSim()
    stopAnimLoop()
    if (hoverLeaveTimer) { clearTimeout(hoverLeaveTimer); hoverLeaveTimer = null }
})

// ── Stars – bright cross-stars + color-tinted regular stars ──
const STAR_COLORS = ['white', 'white', 'white', '#b3d4ff', '#b3d4ff', '#ffe9a0', '#ffd0d0']
const stars = ref(
    Array.from({ length: 220 }, (_, i) => ({
        x: (Math.random() * 100).toFixed(2),
        y: (Math.random() * 100).toFixed(2),
        r: (Math.random() * 1.9 + 0.3).toFixed(2),
        op: (Math.random() * 0.60 + 0.15).toFixed(2),
        dur: (Math.random() * 3 + 2).toFixed(1),
        del: (Math.random() * 8).toFixed(1),
        bright: i < 14,
        color: STAR_COLORS[Math.floor(Math.random() * STAR_COLORS.length)],
    })),
)

// ── Interaction stats ─────────────────────────────────────────
const interactionMap = reactive(new Map())

async function loadInteractionStats() {
    try {
        const res = await api.getInteractionStats(30)
        const stats = res.stats || []
        stats.forEach(s => interactionMap.set(s.guild_id, s.message_count))
    } catch {
        // Non-fatal
    }
}

// ── Guild members cache ───────────────────────────────────────
const guildMembersCache = reactive(new Map())

async function loadGuildMembers(guildId) {
    if (guildMembersCache.has(guildId)) return
    try {
        const res = await api.getGuildMembers(guildId)
        const members = Array.isArray(res) ? res : (res.members || [])
        guildMembersCache.set(guildId, members.slice(0, 5))
    } catch {
        guildMembersCache.set(guildId, [])
    }
}

// ── Friends ───────────────────────────────────────────────────
const friendsList = ref([])

async function loadFriends() {
    try {
        const res = await api.listFriends()
        const raw = Array.isArray(res) ? res : (res.friends || [])
        friendsList.value = raw
            .filter(f => f.status === 'accepted')
            .slice(0, 8)
            .map(f => {
                const friend = (f.requester_id === store.user?.id) ? f.addressee : f.requester
                return friend ? { ...f, friendUser: friend } : null
            })
            .filter(Boolean)
    } catch {
        friendsList.value = []
    }
}

// ── Force Simulation ──────────────────────────────────────────
const simNodes = []
const simLinks = []
const displayNodes = ref([])
const displayLinks = ref([])

let rafId = null
let alpha = 1
const ALPHA_DECAY = 0.025
const ALPHA_MIN = 0.001
const VELOCITY_DECAY = 0.4

function initSim() {
    simNodes.length = 0
    simLinks.length = 0

    const cx = W.value / 2
    const cy = H.value / 2

    // User node (fixed at center)
    simNodes.push({
        id: 'user',
        type: 'user',
        x: cx, y: cy, vx: 0, vy: 0,
        fx: cx, fy: cy,
    })

    // Friends (inner ring ~0.20 radius)
    const friendRingR = Math.min(W.value, H.value) * 0.20
    friendsList.value.forEach((f, i) => {
        const angle = (i / (friendsList.value.length || 1)) * 2 * Math.PI
        simNodes.push({
            id: `friend-${f.friendUser.id}`,
            type: 'friend',
            friendData: f.friendUser,
            x: cx + friendRingR * Math.cos(angle) + (Math.random() - 0.5) * 15,
            y: cy + friendRingR * Math.sin(angle) + (Math.random() - 0.5) * 15,
            vx: 0, vy: 0,
        })
        simLinks.push({
            source: 'user',
            target: `friend-${f.friendUser.id}`,
            distance: friendRingR,
            strength: 0.4,
        })
    })

    store.guilds.forEach((guild, i) => {
        const angle = (i / store.guilds.length) * 2 * Math.PI
        const r = Math.min(W.value, H.value) * 0.32
        const gx = cx + r * Math.cos(angle)
        const gy = cy + r * Math.sin(angle)

        simNodes.push({
            id: `guild-${guild.id}`,
            type: 'guild',
            guildId: guild.id,
            x: gx + (Math.random() - 0.5) * 20,
            y: gy + (Math.random() - 0.5) * 20,
            vx: 0, vy: 0,
        })

        // Link: user ↔ guild; distance depends on interaction intensity
        const msgCount = interactionMap.get(guild.id) || 0
        const intensity = Math.min(msgCount / 50, 1)
        const minDist = Math.min(W.value, H.value) * 0.18
        const maxDist = Math.min(W.value, H.value) * 0.40
        const linkDist = maxDist - intensity * (maxDist - minDist)

        simLinks.push({
            source: 'user',
            target: `guild-${guild.id}`,
            distance: linkDist,
            strength: 0.3 + intensity * 0.3,
        })

        // Member nodes
        const members = guildMembersCache.get(guild.id) || []
        members.forEach((m, j) => {
            const mAngle = (j / (members.length || 1)) * 2 * Math.PI
            const mR = 55
            simNodes.push({
                id: `member-${guild.id}-${m.user_id}`,
                type: 'member',
                guildId: guild.id,
                memberId: m.user_id,
                memberData: m,
                x: gx + mR * Math.cos(mAngle),
                y: gy + mR * Math.sin(mAngle),
                vx: 0, vy: 0,
            })
            simLinks.push({
                source: `guild-${guild.id}`,
                target: `member-${guild.id}-${m.user_id}`,
                distance: 55,
                strength: 0.5,
            })
        })
    })

    alpha = 1
    syncDisplay()
    startSim()
}

function stopSim() {
    if (rafId) { cancelAnimationFrame(rafId); rafId = null }
}

function startSim() {
    stopSim()
    function tick() {
        if (alpha < ALPHA_MIN) { syncDisplay(); return }
        simulateStep()
        alpha *= (1 - ALPHA_DECAY)
        syncDisplay()
        rafId = requestAnimationFrame(tick)
    }
    rafId = requestAnimationFrame(tick)
}

function simulateStep() {
    const cx = W.value / 2
    const cy = H.value / 2
    const nodeMap = new Map(simNodes.map(n => [n.id, n]))

    // Build force accumulators
    const forces = new Map(simNodes.map(n => [n.id, { x: 0, y: 0 }]))

    // 1. Link forces
    simLinks.forEach(link => {
        const a = nodeMap.get(link.source)
        const b = nodeMap.get(link.target)
        if (!a || !b) return
        const dx = b.x - a.x
        const dy = b.y - a.y
        const dist = Math.sqrt(dx * dx + dy * dy) || 1
        const delta = (dist - link.distance) / dist * link.strength
        const fx = dx * delta
        const fy = dy * delta
        if (!a.fx) forces.get(a.id).x += fx
        if (!a.fy) forces.get(a.id).y += fy
        if (!b.fx) forces.get(b.id).x -= fx
        if (!b.fy) forces.get(b.id).y -= fy
    })

    // 2. Repulsion between guild nodes
    const guildNodes = simNodes.filter(n => n.type === 'guild')
    for (let i = 0; i < guildNodes.length; i++) {
        for (let j = i + 1; j < guildNodes.length; j++) {
            const a = guildNodes[i], b = guildNodes[j]
            const dx = b.x - a.x
            const dy = b.y - a.y
            const dist2 = dx * dx + dy * dy
            if (dist2 < 1) continue
            const strength = -3000 / dist2
            const d = Math.sqrt(dist2)
            const fx = dx / d * strength
            const fy = dy / d * strength
            forces.get(a.id).x += fx
            forces.get(a.id).y += fy
            forces.get(b.id).x -= fx
            forces.get(b.id).y -= fy
        }
    }

    // 3. Member repulsion from other members in same guild
    const byGuild = new Map()
    simNodes.filter(n => n.type === 'member').forEach(n => {
        if (!byGuild.has(n.guildId)) byGuild.set(n.guildId, [])
        byGuild.get(n.guildId).push(n)
    })
    byGuild.forEach(members => {
        for (let i = 0; i < members.length; i++) {
            for (let j = i + 1; j < members.length; j++) {
                const a = members[i], b = members[j]
                const dx = b.x - a.x
                const dy = b.y - a.y
                const dist2 = dx * dx + dy * dy
                if (dist2 < 1) continue
                const strength = -500 / dist2
                const d = Math.sqrt(dist2)
                forces.get(a.id).x += dx / d * strength
                forces.get(a.id).y += dy / d * strength
                forces.get(b.id).x -= dx / d * strength
                forces.get(b.id).y -= dy / d * strength
            }
        }
    })

    // 4. Friend ↔ friend repulsion
    const friendPhysNodes = simNodes.filter(n => n.type === 'friend')
    for (let i = 0; i < friendPhysNodes.length; i++) {
        for (let j = i + 1; j < friendPhysNodes.length; j++) {
            const a = friendPhysNodes[i], b = friendPhysNodes[j]
            const dx = b.x - a.x, dy = b.y - a.y
            const dist2 = dx * dx + dy * dy
            if (dist2 < 1) continue
            const strength = -900 / dist2
            const d = Math.sqrt(dist2)
            forces.get(a.id).x += dx / d * strength; forces.get(a.id).y += dy / d * strength
            forces.get(b.id).x -= dx / d * strength; forces.get(b.id).y -= dy / d * strength
        }
    }

    // 5. Gravity toward center for guild nodes (keeps layout from flying off)
    guildNodes.forEach(n => {
        forces.get(n.id).x += (cx - n.x) * 0.02 * alpha
        forces.get(n.id).y += (cy - n.y) * 0.02 * alpha
    })

    // 6. Integrate positions
    simNodes.forEach(n => {
        if (n.fx !== undefined) { n.x = n.fx; n.vx = 0; return }
        if (n.fy !== undefined) { n.y = n.fy; n.vy = 0; return }
        const f = forces.get(n.id)
        n.vx = (n.vx + f.x * alpha) * (1 - VELOCITY_DECAY)
        n.vy = (n.vy + f.y * alpha) * (1 - VELOCITY_DECAY)
        n.x += n.vx
        n.y += n.vy
    })
}

function syncDisplay() {
    displayNodes.value = simNodes.map(n => ({ ...n }))
    displayLinks.value = simLinks.map(l => {
        const nm = new Map(simNodes.map(n => [n.id, n]))
        const a = nm.get(l.source), b = nm.get(l.target)
        return a && b ? { ...l, x1: a.x, y1: a.y, x2: b.x, y2: b.y } : null
    }).filter(Boolean)
}

// ── Zoom / Pan ────────────────────────────────────────────────
const transform = reactive({ x: 0, y: 0, k: 1 })
let isPanning = false
let panStart = { x: 0, y: 0 }

function onWheel(e) {
    const scaleFactor = e.deltaY < 0 ? 1.1 : 0.91
    const newK = Math.max(0.3, Math.min(3, transform.k * scaleFactor))
    const rect = containerRef.value.getBoundingClientRect()
    const mx = e.clientX - rect.left
    const my = e.clientY - rect.top
    transform.x = mx - (mx - transform.x) * (newK / transform.k)
    transform.y = my - (my - transform.y) * (newK / transform.k)
    transform.k = newK
}

function onPointerDown(e) {
    if (e.target.closest('.sg-guild-node') || e.target.closest('.sg-user-node') || e.target.closest('.sg-friend-node')) return
    isPanning = true
    panStart = { x: e.clientX - transform.x, y: e.clientY - transform.y }
    containerRef.value.setPointerCapture?.(e.pointerId)
}

function onPointerMove(e) {
    if (!isPanning) return
    transform.x = e.clientX - panStart.x
    transform.y = e.clientY - panStart.y
}

function onPointerUp() { isPanning = false }

// ── Per-guild float params ────────────────────────────────────
const floatProps = new Map()
watch(
    () => store.guilds,
    (guilds) => {
        guilds.forEach(g => {
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

// ── Unread satellite angles ───────────────────────────────────
const satelliteAngles = reactive(new Map())

// ── Particles (flowing dots along guild connection lines) ─────
const particles = ref([])

function updateParticles(now) {
    const pts = []
    guildLinks.value.forEach(link => {
        const gid = linkGuildId(link.target)
        const col = guildPalette(gid)
        for (let i = 0; i < 3; i++) {
            const phase = i / 3
            const t = (now * 0.22 + phase) % 1
            pts.push({
                x: link.x1 + (link.x2 - link.x1) * t,
                y: link.y1 + (link.y2 - link.y1) * t,
                opacity: Math.sin(t * Math.PI) * 0.90,
                r: 2.2 + (gid % 2) * 0.4,
                color: col.hex,
            })
        }
    })
    particles.value = pts
}

// ── Combined animation loop (satellites + particles) ─────────
let animRafId = null

function startAnimLoop() {
    if (animRafId) return
    const tick = () => {
        const now = Date.now() / 1000
        store.unreadGuildIds.forEach(gid => {
            satelliteAngles.set(gid, (now * 1.8) % (2 * Math.PI))
        })
        updateParticles(now)
        animRafId = requestAnimationFrame(tick)
    }
    animRafId = requestAnimationFrame(tick)
}

function stopAnimLoop() {
    if (animRafId) { cancelAnimationFrame(animRafId); animRafId = null }
}

// ── Hover tooltip ─────────────────────────────────────────────
const hoverGuildId = ref(null)
const hoverPos = ref({ x: 0, y: 0 })

const tooltipGuild = computed(() => {
    if (!hoverGuildId.value) return null
    const guild = store.guilds.find(g => g.id === hoverGuildId.value)
    if (!guild) return null
    const members = guildMembersCache.get(hoverGuildId.value) || []
    const onlineCount = members.filter(m => store.onlineUserIds.has(m.user_id)).length
    const msgCount = interactionMap.get(hoverGuildId.value) || 0
    return { guild, members, onlineCount, msgCount, palette: guildPalette(hoverGuildId.value) }
})

let hoverLeaveTimer = null

function onGuildEnter(e, guildId) {
    if (hoverLeaveTimer) { clearTimeout(hoverLeaveTimer); hoverLeaveTimer = null }
    hoverGuildId.value = guildId
    updateHoverPos(e)
}

function onGuildMove(e) {
    if (hoverGuildId.value) updateHoverPos(e)
}

function onGuildLeave() {
    hoverLeaveTimer = setTimeout(() => {
        hoverGuildId.value = null
        hoverLeaveTimer = null
    }, 120)
}

function onTooltipEnter() {
    if (hoverLeaveTimer) { clearTimeout(hoverLeaveTimer); hoverLeaveTimer = null }
}

function updateHoverPos(e) {
    hoverPos.value = {
        x: Math.min(e.clientX + 18, window.innerWidth - 240),
        y: Math.max(e.clientY - 65, 8),
    }
}

// ── Derived data from displayNodes ───────────────────────────
const userNode = computed(() => displayNodes.value.find(n => n.type === 'user'))

const guildNodeMap = computed(() => {
    const m = new Map()
    displayNodes.value.filter(n => n.type === 'guild').forEach(n => m.set(n.guildId, n))
    return m
})

const memberNodes = computed(() => displayNodes.value.filter(n => n.type === 'member'))
const friendNodesList = computed(() => displayNodes.value.filter(n => n.type === 'friend'))

const guildLinks = computed(() =>
    displayLinks.value.filter(l => l.source === 'user' && l.target.startsWith('guild-')),
)
const friendLinksComp = computed(() =>
    displayLinks.value.filter(l => l.source === 'user' && l.target.startsWith('friend-')),
)
const memberLinks = computed(() =>
    displayLinks.value.filter(l => l.source.startsWith('guild-') && l.target.startsWith('member-')),
)

function guildById(id) { return store.guilds.find(g => g.id === id) }

function linkGuildId(target) {
    return Number(target.replace('guild-', ''))
}

function interactionWidth(guildId) {
    const count = interactionMap.get(guildId) || 0
    const t = Math.min(count / 50, 1)
    return (1.0 + t * 4.0).toFixed(2)
}

// Orbit ring radii (3 rings)
const orbitRadii = computed(() => {
    const minDim = Math.min(W.value, H.value)
    return [0.17, 0.30, 0.46].map(r => r * minDim)
})

const displayName = computed(() => store.user?.nickname || store.user?.username || '')

// ── Bootstrap ─────────────────────────────────────────────────
onMounted(async () => {
    startAnimLoop()
    await Promise.all([
        loadInteractionStats(),
        loadFriends(),
        ...store.guilds.map(g => loadGuildMembers(g.id)),
    ])
    initSim()
})

watch(
    () => [store.guilds.length, interactionMap.size, guildMembersCache.size],
    () => { if (store.guilds.length > 0) initSim() },
)

function goToGuild(guildId) {
    store.selectGuild(guildId)
}

function goToFriend(friendUserId) {
    dm.openDMWith(friendUserId)
}
</script>

<template>
    <div
        class="galaxy-view"
        ref="containerRef"
        @wheel.prevent="onWheel"
        @pointerdown="onPointerDown"
        @pointermove="onPointerMove"
        @pointerup="onPointerUp"
        @pointerleave="onPointerUp"
    >
        <!-- ── Starfield (Milky Way band + cross-stars + tinted stars) ── -->
        <svg class="sg-stars" aria-hidden="true">
            <defs>
                <!-- Milky Way diagonal band -->
                <linearGradient id="sg-milkyway" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%"   stop-color="transparent" />
                    <stop offset="30%"  stop-color="#7c6fff" stop-opacity="0.025" />
                    <stop offset="45%"  stop-color="#9f8fff" stop-opacity="0.055" />
                    <stop offset="55%"  stop-color="#7c6fff" stop-opacity="0.040" />
                    <stop offset="70%"  stop-color="#5b4fcf" stop-opacity="0.025" />
                    <stop offset="100%" stop-color="transparent" />
                </linearGradient>
                <!-- Deep nebula glow – top-left -->
                <radialGradient id="sg-nebula-tl" cx="20%" cy="18%" r="40%">
                    <stop offset="0%"   stop-color="#4f46e5" stop-opacity="0.14" />
                    <stop offset="100%" stop-color="#4f46e5" stop-opacity="0" />
                </radialGradient>
                <!-- Deep nebula glow – bottom-right -->
                <radialGradient id="sg-nebula-br" cx="82%" cy="80%" r="38%">
                    <stop offset="0%"   stop-color="#0e7490" stop-opacity="0.12" />
                    <stop offset="100%" stop-color="#0e7490" stop-opacity="0" />
                </radialGradient>
                <!-- Accent nebula – mid-right -->
                <radialGradient id="sg-nebula-mr" cx="88%" cy="40%" r="28%">
                    <stop offset="0%"   stop-color="#be185d" stop-opacity="0.10" />
                    <stop offset="100%" stop-color="#be185d" stop-opacity="0" />
                </radialGradient>
            </defs>

            <!-- Milky Way band -->
            <rect width="100%" height="100%" fill="url(#sg-milkyway)" />
            <!-- Nebula glows -->
            <rect width="100%" height="100%" fill="url(#sg-nebula-tl)" />
            <rect width="100%" height="100%" fill="url(#sg-nebula-br)" />
            <rect width="100%" height="100%" fill="url(#sg-nebula-mr)" />

            <!-- Bright cross-stars -->
            <g v-for="(s, i) in stars.filter(s => s.bright)" :key="`bs-${i}`">
                <circle
                    :cx="`${s.x}%`" :cy="`${s.y}%`"
                    :r="Number(s.r) * 2.2"
                    :fill="s.color"
                    :style="`--op:${s.op}; animation: sg-twinkle ${s.dur}s ease-in-out ${s.del}s infinite alternate`"
                />
                <line
                    :x1="`calc(${s.x}% - 7px)`" :y1="`${s.y}%`"
                    :x2="`calc(${s.x}% + 7px)`" :y2="`${s.y}%`"
                    :stroke="s.color" stroke-width="0.5" :opacity="Number(s.op) * 0.55"
                />
                <line
                    :x1="`${s.x}%`" :y1="`calc(${s.y}% - 7px)`"
                    :x2="`${s.x}%`" :y2="`calc(${s.y}% + 7px)`"
                    :stroke="s.color" stroke-width="0.5" :opacity="Number(s.op) * 0.55"
                />
            </g>
            <!-- Regular stars (color-tinted) -->
            <circle
                v-for="(s, i) in stars.filter(s => !s.bright)"
                :key="`rs-${i}`"
                :cx="`${s.x}%`"
                :cy="`${s.y}%`"
                :r="s.r"
                :fill="s.color"
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
            <g :transform="`translate(${transform.x},${transform.y}) scale(${transform.k})`">
                <defs>
                    <!-- User avatar clip -->
                    <clipPath id="sg-user-clip">
                        <circle v-if="userNode" :cx="userNode.x" :cy="userNode.y" r="26" />
                    </clipPath>
                    <!-- Guild avatar clips (dynamic radius) -->
                    <clipPath
                        v-for="[gid, node] in guildNodeMap"
                        :key="`clip-${gid}`"
                        :id="`sg-guild-clip-${gid}`"
                    >
                        <circle :cx="node.x" :cy="node.y" :r="guildNodeRadius(gid)" />
                    </clipPath>
                    <!-- Member avatar clips -->
                    <clipPath
                        v-for="n in memberNodes"
                        :key="`clip-${n.id}`"
                        :id="`sg-member-clip-${n.id}`"
                    >
                        <circle :cx="n.x" :cy="n.y" r="9" />
                    </clipPath>
                    <!-- Friend avatar clips -->
                    <clipPath
                        v-for="n in friendNodesList"
                        :key="`clip-${n.id}`"
                        :id="`sg-friend-clip-${n.id}`"
                    >
                        <circle :cx="n.x" :cy="n.y" r="12" />
                    </clipPath>
                    <!-- Galaxy core glow -->
                    <radialGradient v-if="userNode" id="sg-core-glow" cx="50%" cy="50%" r="50%">
                        <stop offset="0%"   stop-color="#818cf8" stop-opacity="0.55" />
                        <stop offset="40%"  stop-color="#6366f1" stop-opacity="0.20" />
                        <stop offset="100%" stop-color="#6366f1" stop-opacity="0" />
                    </radialGradient>
                    <!-- User glow -->
                    <radialGradient v-if="userNode" id="sg-user-glow" cx="50%" cy="50%" r="50%">
                        <stop offset="0%"   stop-color="#818cf8" stop-opacity="0.4" />
                        <stop offset="100%" stop-color="#818cf8" stop-opacity="0" />
                    </radialGradient>
                    <!-- Per-guild line gradients (palette-colored) -->
                    <linearGradient
                        v-for="link in guildLinks"
                        :key="`grad-${link.target}`"
                        :id="`sg-line-grad-${link.target}`"
                        :x1="link.x1" :y1="link.y1"
                        :x2="link.x2" :y2="link.y2"
                        gradientUnits="userSpaceOnUse"
                    >
                        <stop offset="0%"   :stop-color="guildPalette(linkGuildId(link.target)).hex" stop-opacity="0.65" />
                        <stop offset="100%" :stop-color="guildPalette(linkGuildId(link.target)).hex" stop-opacity="0.06" />
                    </linearGradient>
                    <!-- Per-guild halo gradients -->
                    <radialGradient
                        v-for="[gid] in guildNodeMap"
                        :key="`halo-grad-${gid}`"
                        :id="`sg-guild-halo-grad-${gid}`"
                        cx="50%" cy="50%" r="50%"
                    >
                        <stop offset="0%"   :stop-color="guildPalette(gid).hex" stop-opacity="0.22" />
                        <stop offset="100%" :stop-color="guildPalette(gid).hex" stop-opacity="0" />
                    </radialGradient>
                </defs>

                <!-- Galaxy core glow (behind everything) -->
                <circle
                    v-if="userNode"
                    :cx="userNode.x" :cy="userNode.y"
                    :r="Math.min(W, H) * 0.38"
                    fill="url(#sg-core-glow)"
                />

                <!-- Nebula blobs -->
                <ellipse
                    v-if="userNode"
                    :cx="userNode.x * 0.48" :cy="userNode.y * 1.55"
                    :rx="Math.min(W, H) * 0.18" :ry="Math.min(W, H) * 0.11"
                    fill="#7c3aed" opacity="0.07"
                    style="filter: blur(18px)"
                />
                <ellipse
                    v-if="userNode"
                    :cx="userNode.x * 1.55" :cy="userNode.y * 0.45"
                    :rx="Math.min(W, H) * 0.14" :ry="Math.min(W, H) * 0.09"
                    fill="#06b6d4" opacity="0.08"
                    style="filter: blur(16px)"
                />
                <ellipse
                    v-if="userNode"
                    :cx="userNode.x * 1.4" :cy="userNode.y * 1.5"
                    :rx="Math.min(W, H) * 0.12" :ry="Math.min(W, H) * 0.08"
                    fill="#f472b6" opacity="0.06"
                    style="filter: blur(14px)"
                />

                <!-- Orbit guide rings -->
                <circle
                    v-if="userNode"
                    v-for="(r, i) in orbitRadii"
                    :key="`orbit-${i}`"
                    :cx="userNode.x" :cy="userNode.y"
                    :r="r"
                    fill="none"
                    stroke="rgba(255,255,255,0.04)"
                    :stroke-width="i === 1 ? 1.5 : 1"
                    stroke-dasharray="4 8"
                />

                <!-- Connection lines: user → guild (palette-colored) -->
                <line
                    v-for="link in guildLinks"
                    :key="`line-${link.target}`"
                    :x1="link.x1" :y1="link.y1"
                    :x2="link.x2" :y2="link.y2"
                    :stroke="`url(#sg-line-grad-${link.target})`"
                    :stroke-width="interactionWidth(linkGuildId(link.target))"
                />

                <!-- Connection lines: user → friend (teal dashed) -->
                <line
                    v-for="link in friendLinksComp"
                    :key="`fline-${link.target}`"
                    :x1="link.x1" :y1="link.y1"
                    :x2="link.x2" :y2="link.y2"
                    stroke="rgba(34,211,238,0.28)"
                    stroke-width="1"
                    stroke-dasharray="5 6"
                />

                <!-- Member connection lines -->
                <line
                    v-for="link in memberLinks"
                    :key="`mline-${link.target}`"
                    :x1="link.x1" :y1="link.y1"
                    :x2="link.x2" :y2="link.y2"
                    stroke="rgba(139,92,246,0.18)"
                    stroke-width="1"
                    stroke-dasharray="3 5"
                />

                <!-- Flowing particles along guild lines -->
                <circle
                    v-for="(p, i) in particles"
                    :key="`p-${i}`"
                    :cx="p.x" :cy="p.y"
                    :r="p.r"
                    :fill="p.color"
                    :opacity="p.opacity"
                />

                <!-- Member nodes -->
                <g v-for="n in memberNodes" :key="n.id" class="sg-member-node">
                    <circle :cx="n.x" :cy="n.y" r="11" fill="#12102a" stroke="rgba(139,92,246,0.35)" stroke-width="1" />
                    <image
                        v-if="n.memberData?.user?.avatar"
                        :href="n.memberData.user.avatar"
                        :x="n.x - 9" :y="n.y - 9"
                        width="18" height="18"
                        :clip-path="`url(#sg-member-clip-${n.id})`"
                        preserveAspectRatio="xMidYMid slice"
                    />
                    <text
                        v-else
                        :x="n.x" :y="n.y"
                        text-anchor="middle" dominant-baseline="middle"
                        fill="rgba(255,255,255,0.5)" font-size="8" font-family="'Segoe UI', sans-serif"
                    >{{ (n.memberData?.user?.nickname || n.memberData?.user?.username || '?').charAt(0).toUpperCase() }}</text>
                    <circle
                        v-if="store.onlineUserIds.has(n.memberId)"
                        :cx="n.x + 7" :cy="n.y + 7"
                        r="3.5" fill="#22c55e" stroke="#060918" stroke-width="1.5"
                    />
                </g>

                <!-- Friend nodes (inner ring) -->
                <g
                    v-for="n in friendNodesList"
                    :key="n.id"
                    class="sg-friend-node"
                    @click="goToFriend(n.friendData.id)"
                    role="button"
                    :aria-label="n.friendData?.nickname || n.friendData?.username"
                    tabindex="0"
                    @keydown.enter="goToFriend(n.friendData.id)"
                >
                    <circle :cx="n.x" :cy="n.y" r="22" fill="rgba(6,183,210,0.06)" class="sg-friend-pulse" />
                    <circle :cx="n.x" :cy="n.y" r="14" fill="#0d1424" stroke="rgba(34,211,238,0.55)" stroke-width="1.5" />
                    <image
                        v-if="n.friendData?.avatar"
                        :href="n.friendData.avatar"
                        :x="n.x - 12" :y="n.y - 12"
                        width="24" height="24"
                        :clip-path="`url(#sg-friend-clip-${n.id})`"
                        preserveAspectRatio="xMidYMid slice"
                    />
                    <text
                        v-else
                        :x="n.x" :y="n.y"
                        text-anchor="middle" dominant-baseline="middle"
                        fill="rgba(255,255,255,0.8)" font-size="10" font-family="'Segoe UI', sans-serif"
                    >{{ (n.friendData?.nickname || n.friendData?.username || '?').charAt(0).toUpperCase() }}</text>
                    <circle
                        v-if="store.onlineUserIds.has(n.friendData?.id)"
                        :cx="n.x + 10" :cy="n.y + 10"
                        r="4" fill="#22c55e" stroke="#060918" stroke-width="1.5"
                    />
                    <text
                        :x="n.x" :y="n.y + 24"
                        text-anchor="middle" dominant-baseline="middle"
                        fill="rgba(34,211,238,0.7)" font-size="9" font-family="'Segoe UI', sans-serif"
                    >{{ (n.friendData?.nickname || n.friendData?.username || '').slice(0, 9) }}</text>
                </g>

                <!-- Guild nodes -->
                <g
                    v-for="[gid, node] in guildNodeMap"
                    :key="gid"
                    class="sg-guild-node"
                    :style="`--float-delay:${floatProps.get(gid)?.delay ?? 0}s; --float-dur:${floatProps.get(gid)?.dur ?? 4}s; --guild-hex:${guildPalette(gid).hex}`"
                    @click="goToGuild(gid)"
                    @pointerenter="onGuildEnter($event, gid)"
                    @pointermove="onGuildMove"
                    @pointerleave="onGuildLeave"
                    role="button"
                    :aria-label="guildById(gid)?.name"
                    tabindex="0"
                    @keydown.enter="goToGuild(gid)"
                >
                    <circle
                        :cx="node.x" :cy="node.y"
                        :r="guildNodeRadius(gid) + 14"
                        :fill="`url(#sg-guild-halo-grad-${gid})`"
                        class="sg-guild-halo"
                    />
                    <circle
                        :cx="node.x" :cy="node.y"
                        :r="guildNodeRadius(gid)"
                        fill="#1a1740"
                        :stroke="guildPalette(gid).hex"
                        stroke-opacity="0.7"
                        stroke-width="1.5"
                        class="sg-guild-bg"
                    />
                    <image
                        v-if="guildById(gid)?.icon"
                        :href="guildById(gid).icon"
                        :x="node.x - guildNodeRadius(gid)"
                        :y="node.y - guildNodeRadius(gid)"
                        :width="guildNodeRadius(gid) * 2"
                        :height="guildNodeRadius(gid) * 2"
                        :clip-path="`url(#sg-guild-clip-${gid})`"
                        preserveAspectRatio="xMidYMid slice"
                    />
                    <text
                        v-else
                        :x="node.x" :y="node.y"
                        text-anchor="middle" dominant-baseline="middle"
                        fill="white" :font-size="guildNodeRadius(gid) * 0.65"
                        font-weight="600" font-family="'Segoe UI', sans-serif"
                    >{{ guildById(gid)?.name?.charAt(0)?.toUpperCase() }}</text>
                    <text
                        :x="node.x" :y="node.y + guildNodeRadius(gid) + 15"
                        text-anchor="middle" dominant-baseline="middle"
                        fill="rgba(255,255,255,0.6)" font-size="11"
                        font-family="'Segoe UI', sans-serif"
                    >{{ guildById(gid)?.name?.length > 12 ? guildById(gid).name.slice(0, 11) + '…' : guildById(gid)?.name }}</text>
                    <g v-if="store.unreadGuildIds.has(gid)" class="sg-unread-satellite">
                        <circle
                            :cx="node.x + (guildNodeRadius(gid) + 8) * Math.cos(satelliteAngles.get(gid) || 0)"
                            :cy="node.y + (guildNodeRadius(gid) + 8) * Math.sin(satelliteAngles.get(gid) || 0)"
                            r="5" fill="#f59e0b" stroke="#060918" stroke-width="1.5" class="sg-satellite-dot"
                        />
                        <circle
                            :cx="node.x + (guildNodeRadius(gid) + 8) * Math.cos((satelliteAngles.get(gid) || 0) + Math.PI)"
                            :cy="node.y + (guildNodeRadius(gid) + 8) * Math.sin((satelliteAngles.get(gid) || 0) + Math.PI)"
                            r="3" fill="#f59e0b" opacity="0.5"
                        />
                    </g>
                </g>

                <!-- Center: user node -->
                <g class="sg-user-node" v-if="userNode">
                    <circle :cx="userNode.x" :cy="userNode.y" r="52" fill="url(#sg-user-glow)" class="sg-pulse-outer" />
                    <circle :cx="userNode.x" :cy="userNode.y" r="38" fill="rgba(99,102,241,0.18)" class="sg-pulse-inner" />
                    <circle :cx="userNode.x" :cy="userNode.y" r="28" fill="#1a1740" stroke="#818cf8" stroke-width="2" />
                    <image
                        v-if="store.user?.avatar"
                        :href="store.user.avatar"
                        :x="userNode.x - 26" :y="userNode.y - 26"
                        width="52" height="52"
                        clip-path="url(#sg-user-clip)"
                        preserveAspectRatio="xMidYMid slice"
                    />
                    <text
                        v-else
                        :x="userNode.x" :y="userNode.y"
                        text-anchor="middle" dominant-baseline="middle"
                        fill="white" font-size="20" font-weight="700" font-family="'Segoe UI', sans-serif"
                    >{{ displayName?.charAt(0)?.toUpperCase() }}</text>
                </g>
            </g>
        </svg>

        <!-- ── Hover tooltip card ──────────────────────────── -->
        <Transition name="sg-tip">
            <div
                v-if="tooltipGuild"
                class="sg-tooltip-card"
                :style="`left:${hoverPos.x}px; top:${hoverPos.y}px; --tip-color:${tooltipGuild.palette.hex}`"
                @pointerenter="onTooltipEnter"
                @pointerleave="onGuildLeave"
            >
                <div class="sg-tip-header">
                    <div
                        class="sg-tip-icon"
                        :style="tooltipGuild.guild.icon ? `background-image:url('${tooltipGuild.guild.icon}'); background-size: cover; background-position: center` : `background:${tooltipGuild.palette.hex}33`"
                    >{{ !tooltipGuild.guild.icon ? tooltipGuild.guild.name?.charAt(0)?.toUpperCase() : '' }}</div>
                    <div class="sg-tip-title-col">
                        <span class="sg-tip-name">{{ tooltipGuild.guild.name }}</span>
                        <span class="sg-tip-online">
                            <span class="sg-tip-indicator"></span>
                            {{ tooltipGuild.onlineCount }} 在線
                        </span>
                    </div>
                </div>
                <div class="sg-tip-body">
                    <div class="sg-tip-stat">
                        <span class="sg-tip-stat-label">近30天訊息</span>
                        <span class="sg-tip-stat-val">{{ tooltipGuild.msgCount }}</span>
                    </div>
                    <div class="sg-tip-stat">
                        <span class="sg-tip-stat-label">成員數</span>
                        <span class="sg-tip-stat-val">{{ tooltipGuild.members.length }}</span>
                    </div>
                </div>
                <div class="sg-tip-action">點擊進入社群 →</div>
            </div>
        </Transition>

        <!-- ── Welcome overlay ────────────────────────────── -->
        <div class="sg-welcome">
            <p class="sg-username">{{ displayName }}</p>
            <p v-if="store.guilds.length > 0" class="sg-hint">點擊節點進入社群 &nbsp;·&nbsp; 滾輪縮放 &nbsp;·&nbsp; 拖曳平移</p>
            <p v-else class="sg-hint">在左側建立或加入一個社群</p>
        </div>

        <!-- Zoom controls -->
        <div class="sg-zoom-controls">
            <button class="sg-zoom-btn" @click="transform.k = Math.min(3, transform.k * 1.2)" aria-label="放大">+</button>
            <button class="sg-zoom-btn" @click="transform.k = Math.max(0.3, transform.k * 0.83)" aria-label="縮小">−</button>
            <button class="sg-zoom-btn" @click="transform.x=0; transform.y=0; transform.k=1" aria-label="重置">⌂</button>
        </div>
    </div>
</template>

<style scoped>
/* ── Container ─────────────────────────────────────────────── */
.galaxy-view {
    position: relative;
    flex: 1;
    overflow: hidden;
    background:
        radial-gradient(ellipse 80% 55% at 20% 15%, rgba(79,70,229,0.18) 0%, transparent 65%),
        radial-gradient(ellipse 60% 45% at 85% 78%, rgba(14,116,144,0.15) 0%, transparent 60%),
        radial-gradient(ellipse 50% 40% at 90% 35%, rgba(190,24,93,0.10) 0%, transparent 55%),
        radial-gradient(ellipse 70% 60% at 50% 55%, #0d0a2e 0%, #060918 55%, #020510 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 0;
    cursor: grab;
    touch-action: none;
}
.galaxy-view:active { cursor: grabbing; }

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
    to   { opacity: calc(var(--op, 0.3) * 0.12); }
}

/* ── Main canvas ───────────────────────────────────────────── */
.sg-canvas {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
    overflow: visible;
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
    filter: brightness(1.35) drop-shadow(0 0 16px var(--guild-hex, rgba(139,92,246,0.65)));
}

.sg-guild-node:hover .sg-guild-bg {
    stroke-opacity: 1 !important;
}

@keyframes sg-float {
    from { transform: translateY(0px); }
    to   { transform: translateY(-8px); }
}

/* ── Friend nodes ──────────────────────────────────────────── */
.sg-friend-node {
    cursor: pointer;
    pointer-events: all;
    transform-box: fill-box;
    transform-origin: center;
    outline: none;
    transition: filter 0.2s ease;
}

.sg-friend-node:hover,
.sg-friend-node:focus-visible {
    filter: brightness(1.4) drop-shadow(0 0 10px rgba(34,211,238,0.7));
}

.sg-friend-pulse {
    transform-box: fill-box;
    transform-origin: center;
    animation: sg-friend-pulse-anim 2.8s ease-out infinite;
}

@keyframes sg-friend-pulse-anim {
    0%   { transform: scale(0.85); opacity: 0.5; }
    60%  { transform: scale(1.25); opacity: 0.15; }
    100% { transform: scale(1.25); opacity: 0; }
}

/* ── Member nodes ──────────────────────────────────────────── */
.sg-member-node {
    pointer-events: none;
    opacity: 0.85;
}

/* ── Unread satellite ──────────────────────────────────────── */
.sg-satellite-dot {
    filter: drop-shadow(0 0 4px #f59e0b);
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

/* ── Tooltip card ──────────────────────────────────────────── */
.sg-tooltip-card {
    position: fixed;
    z-index: 100;
    min-width: 200px;
    max-width: 240px;
    background: rgba(13, 10, 36, 0.92);
    border: 1px solid color-mix(in srgb, var(--tip-color, #818cf8) 40%, transparent);
    border-radius: 12px;
    padding: 12px 14px;
    pointer-events: auto;
    backdrop-filter: blur(12px);
    box-shadow: 0 8px 32px rgba(0,0,0,0.45), 0 0 0 1px rgba(255,255,255,0.04);
}

.sg-tip-header {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 10px;
}

.sg-tip-icon {
    width: 36px;
    height: 36px;
    border-radius: 8px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: 15px;
    font-weight: 700;
    border: 1px solid rgba(255,255,255,0.1);
}

.sg-tip-title-col {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
}

.sg-tip-name {
    font-size: 13px;
    font-weight: 600;
    color: rgba(255,255,255,0.92);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.sg-tip-online {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 11px;
    color: rgba(255,255,255,0.45);
}

.sg-tip-indicator {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #22c55e;
    box-shadow: 0 0 5px #22c55e;
    flex-shrink: 0;
}

.sg-tip-body {
    display: flex;
    gap: 12px;
    margin-bottom: 10px;
}

.sg-tip-stat {
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.sg-tip-stat-label {
    font-size: 10px;
    color: rgba(255,255,255,0.38);
}

.sg-tip-stat-val {
    font-size: 17px;
    font-weight: 700;
    color: var(--tip-color, #818cf8);
}

.sg-tip-action {
    font-size: 11px;
    color: rgba(255,255,255,0.35);
    text-align: right;
}

/* Tooltip transitions */
.sg-tip-enter-active,
.sg-tip-leave-active {
    transition: opacity 0.15s ease, transform 0.15s ease;
}

.sg-tip-enter-from,
.sg-tip-leave-to {
    opacity: 0;
    transform: translateY(6px) scale(0.97);
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
    color: rgba(255, 255, 255, 0.32);
    letter-spacing: 0.02em;
}

/* ── Zoom controls ─────────────────────────────────────────── */
.sg-zoom-controls {
    position: absolute;
    bottom: 80px;
    right: 20px;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.sg-zoom-btn {
    width: 32px;
    height: 32px;
    background: rgba(13, 10, 36, 0.85);
    border: 1px solid rgba(139, 92, 246, 0.35);
    border-radius: 6px;
    color: rgba(255, 255, 255, 0.65);
    font-size: 16px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
    line-height: 1;
}

.sg-zoom-btn:hover {
    background: rgba(99, 102, 241, 0.28);
    border-color: rgba(139, 92, 246, 0.8);
    color: white;
}
</style>
