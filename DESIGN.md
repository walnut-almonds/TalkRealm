---
name: Kinetic Noir — TalkRealm Edition
source: adapted from walnut-almonds.github.io DESIGN.md (Kinetic Noir)
colors:
  # Surfaces (dark → light = depth; left rail is deepest)
  surface-lowest: '#0e0e0e'   # nav rail
  surface-low: '#131313'      # sidebars (channels / members / DM)
  surface: '#1b1b1b'          # chat reading surface
  surface-high: '#1f1f1f'     # modals, popovers
  surface-hover: '#242424'
  surface-active: '#2a2a2a'
  surface-input: '#111111'
  on-surface: '#e2e2e2'
  on-surface-variant: '#b4b7b8'
  on-surface-muted: '#7c8082'
  outline: 'rgba(255,255,255,0.08)'
  outline-strong: 'rgba(255,255,255,0.16)'
  # Accent — the only decorative color. Slate blue, used sparingly.
  accent: '#b3c6f3'
  accent-hover: '#cdd9f6'
  accent-container: '#33466c'
  accent-glow: 'rgba(179,198,243,0.22)'
  # Functional — muted pastels, never saturated
  success: '#83c896'
  warning: '#e3c26d'
  danger: '#ffb4ab'
  danger-container: '#93000a'
  on-danger-container: '#ffdad6'
  # Presence
  online: '#83c896'
  idle: '#e3c26d'
  busy: '#ffb4ab'
  offline: '#5c6062'
typography:
  body:
    fontFamily: '"Hanken Grotesk", "Noto Sans TC", "Segoe UI", sans-serif'
    fontSize: 14px
    lineHeight: 1.55
  mono-label:
    fontFamily: '"Geist Mono", Consolas, monospace'
    fontSize: 11px
    fontWeight: '500'
    letterSpacing: 0.08em
    transform: uppercase
shapes:
  radius: 0px
  radius-lg: 2px
  exception: 'avatars and presence dots are perfect circles (people = nodes)'
spacing:
  unit: 4px
---

## Brand & Style

TalkRealm inherits the **Kinetic Noir** language of walnut-almonds.github.io — the personal
site is the front door to this project, and the app should feel like walking deeper into the
same building. High-tech minimalism: near-black surfaces, hairline borders instead of shadows,
one restrained slate-blue accent, and a monospace "technical voice" for metadata.

The emotional target is **quiet power**: a calm, focused room where conversation is the hero.
Flourishes are allowed — corner ticks, grid lines, instant terminal-style hovers, presence as
light — but every flourish must stay subdued. If a detail draws the eye away from the message
stream, it is wrong.

Explicit anti-goal: no Discord DNA. No blurple, no circle→squircle icon morph, no saturated
badge colors, no purple gradients.

## Colors

Monochrome first. The surface scale (`#0e0e0e → #2a2a2a`) is the Kinetic Noir container scale;
depth reads dark→light from the left rail into the chat surface.

- **On-surface white** (`#e2e2e2`) carries all primary content. Pure `#ffffff` is reserved for
  the highest-priority marks (active borders, unread pips, focus rings).
- **Slate accent** (`#b3c6f3`, container `#33466c`) is the only decorative color: links,
  mentions, active tabs, focused inputs, interactive glows. Used at word/line scale, never as
  large fills.
- **Functional pastels** (success/warning/danger) are desaturated Material-dark tones. Presence
  colors are functional, not decorative — a small dot or a faint glow, never a badge shout.
- Prefer white-at-low-opacity over mid-tone grays for borders and dividers.

## Typography

Two voices, same as the source system:

- **Hanken Grotesk** for messages, names, headings — with **Noto Sans TC** as the CJK
  companion (the app is zh-TW-first; Hanken has no CJK glyphs). Tight letter-spacing on large
  headings only.
- **Geist Mono** is the technical voice: category headers, timestamps, keyboard hints, badges,
  status labels, invite codes. Almost always uppercase with wide letter-spacing. When a piece
  of text is *about the system* rather than *from a person*, it is mono.

Body stays 14px/1.55 for long reading sessions.

## Layout & Spacing

The Discord-learned ergonomics stay (rail / channel list / message stream / member list) —
differentiation lives in the skin and signature moments, not in relearning navigation.

- 4px baseline grid. Hairline (1px, white @ 8%) separators between the four columns.
- Grid lines may be *visualized* on empty/ambient surfaces (auth page, empty states) as
  1px white @ ~4% — the blueprint feel — never behind message text.

## Elevation & Depth

No drop shadows on a black canvas — **light defines edges**.

1. **Surfaces:** flat fills from the surface scale, separated by hairlines.
2. **Containers (cards, inputs):** 1px `outline` border; input fields sit darker than their
   surface (`#111`) like a recessed slot.
3. **Overlays (modals, popovers, pickers):** dark translucent fill (`rgba(18,18,18,0.85)`)
   with `backdrop-filter: blur(20px)` and a sharper border (white @ 16%).

Active/interactive states use a faint slate **glow** (`0 0 0 1px accent` or a soft
`accent-glow` halo), never a bigger element.

## Shapes

**Sharp (0px).** Panels, buttons, inputs, modals, attachments, tooltips: right angles.
`--radius-lg` is 2px and reserved for tiny chips where 0px looks broken.

The one deliberate exception, inherited from the "floating nodes" concept: **people are
nodes**. Avatars and presence dots are perfect circles. Guilds are *places*, so guild icons
are sharp squares — this circle/square split is the signature that replaces Discord's
squircle morph.

## Components

- **Buttons:** transparent fill, 1px white @ 16% border. Hover inverts instantly (white
  background, near-black text, `transition: none`) — terminal responsiveness. Danger buttons
  use `danger-container` fill with `on-danger-container` text.
- **Inputs:** recessed dark slot, 1px hairline border; focus swaps the border to accent, no
  glow ring. Placeholder in muted mono where the field is technical (search, invite codes).
- **Nav rail:** deepest surface. Guild squares with hairline borders; active = 1px white
  border + 2px white pip line on the left edge. Hover is an instant border lift — no morph,
  no translate.
- **Channel list:** active channel gets a `2px` accent left line + `surface-active` fill.
  Category headers are mono uppercase.
- **Messages:** flat rows (no bubbles). Hover paints a faint fill plus a 1px accent hairline
  on the left. Mentions of you: accent-tinted row, not amber. Mention chips: accent container.
- **Presence as light:** online = soft green dot; speaking = a slow pastel pulse on the
  avatar ring; channel/voice activity may warm a surface by one step. Light is the metaphor,
  badges are the fallback.
- **Translation block (signature feature):** treated as a system annotation — mono language
  badge, hairline top border, accent reveal interaction for the guess game.
- **Toasts:** sharp glass panels with a 2px functional-color left edge.
- **Auth page:** the front door and the strongest Kinetic Noir moment — void background,
  faint blueprint grid, slate radial glow, a sharp bordered card with corner ticks. This page
  may quote the personal site almost directly.

## Motion

Motion is rare and purposeful:

- Hovers on buttons: instant (0ms). Hovers on rows/list items: ≤100ms fades.
- One well-orchestrated moment beats scattered micro-animations — the auth page and empty
  states carry the ambience (drifting node dots, slow glow breathing); the chat surface
  stays still.
- Presence pulses are slow (≥1.2s) and low-amplitude.

## Implementation Notes

- Tokens live in `web/src/styles/main.css` `:root`. Legacy names (`--primary`, `--bg-*`,
  `--text-*`) are kept as aliases; new code should prefer the semantic names. `--accent`,
  `--accent-hover`, `--brand` are now defined (components previously relied on their
  Discord-hex `var()` fallbacks).
- Fonts are loaded in `web/index.html` via Google Fonts (Hanken Grotesk, Geist Mono,
  Noto Sans TC).
- `web/css/styles.css` + `web/js/` are the pre-Vue legacy app and are **not** themed; the
  Vue app under `web/src/` is the only styled target.
