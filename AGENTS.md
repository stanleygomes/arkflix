# Arkflix Agent Guidelines

Arkflix is an Apple iOS / tvOS HIG-styled Web Client for Jellyfin with Google Cast (Chromecast) support.

## 1. Architecture & Principles
- **Code Language:** All code, comments, variables, types, and files MUST be written in English.
- **Default Locale:** Portuguese (`pt-BR`) is the primary UI translation language via `src/locales/pt-BR.ts`.
- **Decoupled Data Layer:** Components NEVER call APIs directly. Always use custom hooks (`src/hooks/`) with TanStack Query and Zustand.
- **Agnostic UI:** Presentational components and pages only assemble reusable micro-components (`src/components/ui/`).
- **Design System:** Apple HIG aesthetic (Inter font, squircles, subtle glassmorphism `backdrop-blur`, dark `#000000` base, AppleSpinner, fluid spring physics with Framer Motion).
- **Zero Extra UI Libraries:** Only use existing dependencies (Tailwind CSS, Framer Motion, Lucide React, HLS.js, Zustand, TanStack Query).

## 2. Directory Structure
```text
src/
├── components/
│   ├── layout/       # Navbar, MainLayout, PageTransition
│   ├── media/        # HeroBanner, MediaCard, MediaRow, DetailModal
│   ├── player/       # VideoPlayer, VideoControls, CastRemoteView
│   └── ui/           # Button, Input, Select, Tabs, Badge, Modal, Slider, Skeleton, AppleSpinner
├── hooks/            # useMedia, useAuth, useChromecast, useTheme, useTranslation
├── locales/          # pt-BR.ts (Typed dictionary)
├── pages/            # HomePage, LoginPage, WatchPage, ProfilePage, MoviesPage, SeriesPage, MyListPage, SearchPage
├── services/         # api.ts (Axios + dynamic baseURL), jellyfin.ts (Pure API methods)
├── stores/           # authStore, castStore, modalStore, themeStore, i18nStore
└── types/            # TypeScript models (jellyfin.ts, cast.ts)
```

## 3. Key Conventions & Rules
- **Server Persistence:** The Jellyfin server URL is dynamic and stored in `localStorage ('arkflix_server_url')`. Default: `https://ark-flix.duckdns.org`.
- **Authentication & Headers:** All API requests must include `X-Emby-Authorization` and `X-Emby-Token` via Axios interceptors.
- **Profiles:** Support multi-user accounts saved in `localStorage ('arkflix_profiles')` with instant profile switching.
- **Themes:** Support `dark`, `light`, and `auto` (OS sync via `prefers-color-scheme`) in `themeStore.ts`.
- **Video Player:** Dual mode (local HLS/HTML5 video and remote Chromecast controller `CastRemoteView`). Must report playback progress (`/Sessions/Playing/Progress`) every 10s.
- **Git Workflow:** Always commit changes atomically in dedicated feature branches before opening PRs to `master`.
