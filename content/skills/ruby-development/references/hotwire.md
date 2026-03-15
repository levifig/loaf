# Hotwire Guide

## Decision Guide

| Need | Solution |
|------|----------|
| Faster navigation | Turbo Drive (automatic) |
| Update one section | Turbo Frame |
| Update multiple elements | Turbo Stream |
| Real-time from server | Turbo Stream + broadcasts |
| Client-side interactivity | Stimulus |

## Key Conventions

- **Turbo Streams**: `append`, `prepend`, `replace`, `update`, `remove`
- **Real-time broadcasts**: `after_create_commit -> { broadcast_append_to target }` in models
- **Stimulus naming**: `clipboard_controller.js` → `data-controller="clipboard"`, `sourceTarget` → `data-clipboard-target="source"`
- **Lazy frames**: `turbo_frame_tag "comments", src: path, loading: :lazy`
- **Break out of frame**: `data: { turbo_frame: "_top" }`
