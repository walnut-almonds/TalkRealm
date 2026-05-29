# Social Galaxy Homepage Concept

## Overview

The homepage is not a traditional sidebar-based chat UI.

Instead, it is a living, floating social universe where the user is placed at the center of their relationships and communities.

The goal is to create a homepage that feels alive, emotional, and visually representative of a user's social world.

---

# Core Concept

## User-Centered Galaxy

The current user is positioned at the center of the universe.

Surrounding elements represent social relationships and communities.

### Mapping

- Friends → Planets
- Groups / Servers → Galaxies or Clusters
- Voice Channels → Glowing active regions
- Recently active friends → Move closer to the center
- Inactive friends → Drift farther away
- Blocked users → Near black hole edges or fragmented visuals

---

# Dynamic Relationship Visualization

The system visualizes interaction intensity in real time.

## Higher interaction frequency results in:

- Shorter distance between nodes
- Thicker connection lines
- Brighter planets/nodes
- Faster orbits or stronger movement

Interaction signals may include:

- Message frequency
- Voice activity
- Reactions
- Shared groups
- Recent interactions

---

# Navigation & Interaction

## Zooming Into Communities

Users can:

- Click groups to zoom into galaxy clusters
- Explore social structures visually
- Navigate relationships spatially

---

# Visual Hierarchy

## Center

- Current user avatar

## First Ring

- Most interacted friends

## Second Ring

- Group / server clusters

## Third Ring

- Friends of friends
- Peripheral social connections

---

# Realtime Effects

The homepage should feel alive through realtime animations and status updates.

## Examples

### Online Status

- Nodes glow when online

### Voice Activity

- Pulsing effects
- Animated connection edges

### Typing Indicators

- Ripple animations

### Unread Messages

- Small orbiting satellites
- Notification particles

---

# UI Philosophy

Minimal and immersive.

## UI Characteristics

- No traditional sidebar
- Scroll wheel zoom
- Drag-to-pan navigation
- Click nodes to open chats
- Ambient and spatial interface design

---

# Technical Direction

## Recommended Stack

### Frontend

- React
- PixiJS or Three.js

### Graph Layout

- Force-directed graph physics
- d3-force or custom physics simulation

### Realtime

- WebSocket realtime updates

---

# MVP Scope (Phase 1)

## Do NOT build initially

Avoid overengineering early on.

### Skip:

- Advanced shaders
- Full 3D universe
- AI relationship analysis
- Complex recommendation systems

---

# MVP Goal

Build a functional realtime social galaxy.

## Core Requirements

- User positioned at center
- Friends represented as nodes
- Groups represented as clusters
- Realtime online status
- Automatic force-graph positioning
- Clickable nodes that open chats

---

# Desired Feeling

The experience should feel like:

> "This is my social universe."

The homepage should be:

- Emotionally engaging
- Visually memorable
- Suitable for side project showcasing
- Potentially viral and highly shareable

---

# Long-Term Vision

This is not just a chat homepage.

It is:

## A realtime social relationship visualization system.