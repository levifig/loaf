# Typography

## Type Scale

| Token | Size | Use |
|-------|------|-----|
| `xs` | 12px | Captions |
| `sm` | 14px | Secondary text |
| `base` | 16px | Body default |
| `lg` | 18px | Lead text |
| `xl` | 20px | Small headings |
| `2xl` | 24px | Section headings |
| `3xl` | 30px | Page headings |
| `4xl` | 36px | Hero headings |

Base 16px, ratio 1.25 (Major Third).

## Font Stacks

| Purpose | Stack |
|---------|-------|
| Sans | Inter var, system-ui, sans-serif |
| Serif | Merriweather, Georgia, serif |
| Mono | JetBrains Mono, Consolas, monospace |

Load with `font-display: swap`. Use `woff2` format.

## Line Height

| Token | Value | Use |
|-------|-------|-----|
| `tight` | 1.25 | Headings |
| `normal` | 1.5 | Body text |
| `relaxed` | 1.625 | Long-form content |

## Rules

- Minimum body font size: 16px (14px for captions)
- Max content width: 65ch
- Use `rem` units for resizable text
- Use `font-variant-numeric: tabular-nums` in tables
