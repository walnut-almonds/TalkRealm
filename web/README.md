# TalkRealm Web

TalkRealm's active web client is a Vue 3 single-page application built with Vite. The legacy files in `web/js` and `web/css` are retained for reference; product work belongs in `web/src`.

## Requirements

- Node.js and npm
- A configured TalkRealm backend listening on `http://localhost:8080` for local development

## Development

```bash
cd web
npm install
npm run dev
```

Vite starts on `http://localhost:3000`. Its development server proxies `/api` requests to `http://localhost:8080`, so the frontend uses relative API paths and does not need a separate API URL configuration for the default local setup.

## Validation

```bash
cd web
npm run build
npm run check:i18n
```

`npm run build` creates the production bundle in `web/dist`. `npm run check:i18n` verifies that every translation key used in `web/src` exists in the English base locale.

## Project Layout

```text
web/
├── src/
│   ├── api/          # REST API client
│   ├── components/   # Vue UI components
│   ├── composables/  # Reusable UI behavior
│   ├── i18n/         # Locale setup and translation dictionaries
│   ├── stores/       # Pinia application state
│   ├── styles/       # Kinetic Noir design tokens and global styles
│   └── views/        # Route-level views
├── scripts/           # Development validation scripts
├── index.html         # Vite entry document
└── vite.config.js     # Development server and build configuration
```

## Application Behavior

The client manages authentication, guilds, channels, direct messages, realtime WebSocket updates, file uploads, feeds, learning features, localization, and voice-related UI. REST requests are made through `src/api`, while realtime event handling is centralized in the relevant stores and composables.

Use the existing semantic design tokens from `src/styles/main.css` for new UI work. The interface follows the Kinetic Noir system documented in the repository-level `DESIGN.md`.