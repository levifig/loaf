# Mobile Apps with Hotwire Native

## Architecture

Web content in native shells (iOS/Android) via WebView with native-feeling transitions.

## Key Conventions

- **Path configuration**: JSON files at `/configurations/ios_v1.json` defining navigation rules (modal, refresh, external)
- **Bridge components**: Stimulus controllers with native counterparts for native UI (top bar buttons, camera, push notifications)
- **Every screen needs a URL**: All screens must be URL-accessible for proper native navigation
- **Progressive enhancement**: Start web-only → add bridge components → native screens only where required
- **Minimum tap targets**: 44x44pt (iOS), 48x48dp (Android)
