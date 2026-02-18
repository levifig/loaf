# Performance Optimization

## Stack

| Tool | Purpose |
|------|---------|
| `@next/bundle-analyzer` | Visualize bundle size |
| `React.lazy` / `next/dynamic` | Code splitting |
| `next/image` | Optimized images |
| `web-vitals` | Performance metrics |
| `react-window` | List virtualization |

## Web Vitals Targets

| Metric | Target |
|--------|--------|
| LCP (Largest Contentful Paint) | < 2.5s |
| FID (First Input Delay) | < 100ms |
| CLS (Cumulative Layout Shift) | < 0.1 |

## Conventions

- **Server Components** by default (zero JS to client)
- `"use client"` only when needed (hooks, event handlers)
- Tree-shake imports: `import { Search } from "lucide-react"` not `import * as Icons`
- `next/image` with `sizes` prop and `priority` for above-the-fold
- Reserve space for dynamic content to prevent CLS
- Virtualize lists > 100 items with `react-window`
- Memoize only expensive computations (not `count * 2`)

## Critical Rules

- Measure before optimizing (bundle analyzer, Web Vitals)
- Never premature-optimize
- Never over-memoize simple components
