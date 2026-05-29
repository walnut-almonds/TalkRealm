# Social Galaxy Enhancement Plan

# Vision

Social Galaxy is not just a chat homepage.

It is:

> A living realtime social universe.

The goal is to transform a traditional messaging interface into a spatial, emotional, and immersive representation of human relationships.

Instead of:

* server lists
* channels
* sidebars
* unread counters

Users experience:

* social gravity
* relationship distance
* emotional presence
* living communities
* realtime interaction flow

The system should feel less like software and more like a living digital ecosystem.

---

# Implementation Status

## ✅ Implemented (2026-05-29)

### 1. Presence System
- **Online friend breathing glow** — `sg-friend-breathe` animation, 3.2 s breathing cycle for nodes where `onlineUserIds.has(friendId)`
- **Focus / Attention Gravity** — clicking a node triggers `focusOnNode()` with cubic ease-out smooth pan (`smoothPan()`); unrelated nodes fade to 35–40 % opacity
- **Reset focus** — clicking the background calls `resetFocus()` and animates back to origin

### 2. Typing Presence
- `loadGuildChannels()` builds `guildChannelMap` (guildId → Set\<channelId\>) on mount
- `typingGuildIds` computed: iterates `store.typingUsers`, maps channelId → guild, returns active set
- **Typing ripple ring** — `sg-typing-ripple` amber circle pulsing outward (1.4 s loop) when any user is typing in that guild
- Tooltip badge: ✍️ 有人正在輸入

### 3. Voice Activity Presence
- `voiceGuildIds` computed: reads `voiceStore.voiceParticipants`, maps channelId → guild
- **Voice pulse ring** — `sg-voice-pulse` cyan breathing circle (2 s) when voice channel is active
- Tooltip badge: 🎙️ 語音頻道活躍

### 4. Message Flash Events
- `flashGuildIds` reactive Set; watcher on `store.unreadGuildIds` detects newly added guilds
- **Flash ring** — `sg-msg-flash` (1.4 s forwards animation, expands + fades) when new message arrives on a guild

### 5. Noise-Based Organic Drift
- `harmonicNoise(seed, t, freq)` — three-harmonic approximation (7.31, 13.73, 5.07 seed multipliers, golden-ratio frequency scaling)
- Runs inside `startAnimLoop` tick after `alpha < ALPHA_MIN * 5`
- Online friends/members: 38 % speed, 30 px amplitude; offline: 18 %, 30 px; guild nodes: 55 px amplitude

### 6. Time-Based Atmosphere
- `updateAtmosphere()` called on mount, sets `timeAtmosphere` ref (night / dawn / dusk / day)
- `data-atmosphere` attribute on `.galaxy-view` drives CSS `filter` and background overrides
- Night: desaturated deep blue; Dawn: purple-orange; Dusk: rose-violet; Day: default

### 7. Enhanced Tooltip
- `isTyping` and `hasVoice` fields added to `tooltipGuild` computed
- Badge row shown only when active

---

# Current State

The current implementation already includes:

* User-centered galaxy structure
* Friend nodes
* Guild / group clusters
* Interaction-based node distance
* Basic force simulation
* Zoom / pan interaction
* Starfield background
* Realtime-ready architecture

This is already beyond a standard graph visualization.

However, the next stage is:

> Evolving from a graph UI into a living social organism.

---

# Core Enhancement Goals

The next evolution of Social Galaxy should focus on:

1. Presence
2. Emotional atmosphere
3. Spatial depth
4. Relationship memory
5. Cinematic interaction
6. Living universe behavior

---

# 1. Presence System

## Goal

Users should feel:

> "People are actually alive in this universe."

Presence should not be represented only by a green dot.

The entire universe should react to user activity.

---

## Online Presence

### Effects

* Soft glow
* Breathing animation
* Ambient particles
* Slight orbit stabilization

### Example

Online nodes:

* brighter
* more stable
* slightly animated

Offline nodes:

* colder
* dimmer
* drifting

---

## Typing Presence

When a user is typing:

### Effects

* Ripple around node
* Edge pulse animation
* Small particle emission
* Nearby nodes subtly react

The effect should resemble:

* signal propagation
* social energy transfer

---

## Voice Activity Presence

Voice channels should feel energetic and spatial.

### Effects

* Pulsing edges
* Resonating glow
* Low-frequency movement
* Shared voice clusters tightening together

Possible extension:

* spatial voice visualization
* wave distortion effects

---

# 2. Relationship Memory System

## Goal

Relationships should evolve over time.

The graph should not only show:

* current interaction

It should show:

* history
* momentum
* fading
* emotional weight

---

## Relationship Momentum

Current interaction score should become:

```ts
score =
  recentInteractions * 0.7 +
  weeklyTrend * 0.2 +
  longTermHistory * 0.1
```

This allows:

* rapidly growing friendships
* fading relationships
* long-term bonds

---

## Orbit Memory

Nodes should leave subtle relationship trails.

### Examples

Highly active relationships:

* stable orbit
* bright trails

Inactive relationships:

* faded trails
* slow drifting

Former close friends:

* ghost orbit paths
* distant residual glow

---

## Blocked Users

Instead of disappearing:

* pulled toward black hole regions
* fragmented rendering
* distorted connection lines

This creates symbolic emotional visualization.

---

# 3. Spatial Depth System

## Goal

The universe should feel physically large.

Currently the graph exists mostly on a flat plane.

The next step is:

* layered depth
* atmospheric perspective
* spatial immersion

---

# Multi-Layer Rendering

## Foreground Layer

Core friends:

* large
* sharp
* bright
* animated

---

## Midground Layer

Guild clusters:

* moderate glow
* medium opacity

---

## Background Layer

Peripheral users:

* blurred
* smaller
* dimmer
* slow drifting

---

# Parallax Camera

Different layers move at different speeds.

Dragging the universe should create:

* depth illusion
* cinematic motion
* immersive navigation

---

# 4. Ambient Universe System

## Goal

The universe should have moods.

It should react to:

* time
* activity
* social intensity
* emotional density

---

# Social Weather

## Late Night

* dark blue palette
* slow particles
* soft ambient glow

---

## High Activity

* brighter nebula
* flowing particles
* active edges
* stronger color saturation

---

## Quiet Periods

* colder tones
* sparse movement
* reduced glow
* drifting particles

---

# Dynamic Atmosphere

The universe should continuously evolve based on:

* realtime activity
* group density
* active conversations
* voice participation

---

# 5. Camera Language System

## Goal

Navigation should feel cinematic.

Not:

* graph viewer

But:

* interactive sci-fi interface

---

# Focus System

Clicking a node should:

* smoothly focus camera
* fade unrelated nodes
* highlight important edges
* expand nearby cluster

---

# Guild Zoom

Clicking a guild:

* camera zooms inward
* cluster expands
* local social structure revealed

Possible future:

* nested galaxy exploration

---

# Attention Gravity

Opening a conversation:

* slightly pulls that node closer
* increases brightness
* stabilizes orbit

This creates:

* emotional focus
* attention visualization

---

# 6. Physics Evolution

## Goal

The universe should move naturally.

Current force simulation should evolve into:

* organic motion
* emotional movement
* living behavior

---

# Noise-Based Motion

Introduce:

* simplex noise
* ambient drift
* velocity randomness

This prevents:

* mechanical movement
* robotic motion

---

# Personality-Based Movement

Different node types should behave differently.

## Examples

Highly active friends:

* energetic orbit
* stable movement

Inactive users:

* unstable drifting
* slower movement

Groups:

* gravitational clustering

---

# 7. Realtime Event System

## Goal

Every social action should affect the universe.

---

# Supported Realtime Events

## Messaging

* edge pulse
* node flicker
* message particles

---

## Reactions

* micro bursts
* emoji particles
* color ripple

---

## Typing

* expanding wave
* edge transmission effect

---

## Voice Join

* cluster tightening
* pulse synchronization

---

# 8. Technical Architecture Improvements

## Goal

The system should scale cleanly.

Current implementation should eventually separate:

* simulation
* rendering
* presence
* camera
* realtime events

---

# Recommended Architecture

## useGalaxySimulation()

Responsible for:

* force simulation
* layout solving
* orbit updates
* movement logic

---

## useGalaxyPresence()

Responsible for:

* online state
* typing state
* voice activity
* unread status

---

## useGalaxyCamera()

Responsible for:

* zoom
* focus transitions
* cinematic movement
* parallax

---

## useGalaxyRenderer()

Responsible for:

* PixiJS rendering
* particles
* shaders
* node visuals

---

# 9. Performance Improvements

## Recommended Future Optimizations

### Web Worker Simulation

Move:

* force calculations
* clustering
* layout solving

off the main thread.

---

## Object Pooling

Avoid excessive:

* particle creation
* node recreation
* garbage collection spikes

---

## Texture Atlas

Batch:

* avatars
* icons
* effects

for efficient rendering.

---

# 10. Long-Term Vision

Social Galaxy should eventually become:

> A realtime emotional operating system for online relationships.

Not:

* a chat app
* a graph visualization
* a server list

But:

* a living social world

The universe should feel:

* emotional
* ambient
* reactive
* personal
* memorable

---

# Immediate Next Priorities

## Priority 1 — Presence Layer

Implement:

* breathing animations
* typing ripple
* voice pulse
* online glow

This alone will dramatically increase perceived liveliness.

---

## Priority 2 — Camera System

Implement:

* smooth focus
* cinematic transitions
* zoom-to-cluster
* node highlighting

---

## Priority 3 — Spatial Depth

Implement:

* multi-layer rendering
* parallax movement
* atmospheric fading
* depth separation

---

# Final Product Feeling

The desired emotional response is:

> "This feels like my social universe."

Not:

* a UI
* a dashboard
* a graph

But:

* a living digital cosmos shaped by human relationships.
