---
name: typescript-development
description: >-
  Covers TypeScript 5+, React 18+, Next.js 14+ App Router, Zustand, Tailwind
  CSS, and Vitest testing.
---

# TypeScript Development

Modern TypeScript development with React ecosystem.

## Stack Overview

| Layer | Default | Alternatives |
|-------|---------|--------------|
| Language | TypeScript 5+ | JavaScript (ESM) |
| Runtime | Node.js 20+ | Bun, Deno |
| Framework | Next.js 14+ | Vite, Remix |
| UI Library | React 18+ | - |
| State (Client) | Zustand | Context + Reducer |
| State (Server) | React Query | SWR |
| Forms | React Hook Form + Zod | - |
| Styling | Tailwind CSS + CVA | CSS Modules |
| Testing | Vitest + RTL | Jest |
| E2E Testing | Playwright | Cypress |
| Package Manager | pnpm | npm, yarn |

## Topics

| Topic | Use For |
|-------|---------|
| [Core](references/core.md) | Project setup, tsconfig, modern TS features, type utilities |
| [React](references/react.md) | Components, hooks, Context API, performance patterns |
| [Next.js](references/nextjs.md) | App Router, Server/Client Components, Server Actions, routing |
| [Types](references/types.md) | Advanced types, generics, conditional types, type guards |
| [State](references/state.md) | Zustand, React Query, Context + Reducer, URL state |
| [Forms](references/forms.md) | React Hook Form, Zod validation, Server Actions integration |
| [API](references/api.md) | Fetch wrappers, React Query, tRPC, GraphQL, WebSockets |
| [Testing](references/testing.md) | Vitest, React Testing Library, MSW, Playwright E2E |
| [Styling](references/styling.md) | Tailwind CSS, CVA variants, dark mode, responsive design |
| [Performance](references/performance.md) | Bundle analysis, code splitting, memoization, Web Vitals |
| [Accessibility](references/a11y.md) | WCAG compliance, ARIA, keyboard navigation, screen readers |
| [Mobile](references/mobile.md) | React Native, Expo, navigation, platform-specific code |
| [ESM](references/esm.md) | ESM patterns, JSDoc types, JS vs TS decision guide |
| [Debugging](references/debugging.md) | Console methods, DevTools, source maps, async debugging |

## Critical Rules

### Always

- Use strict mode in tsconfig
- Type all function parameters and returns
- Handle null/undefined explicitly
- Use Server Components by default
- Validate on both client and server
- Test with screen readers
- Measure before optimizing

### Never

- Use `any` (use `unknown` with type guards)
- Use `!` (non-null assertion) without justification
- Store server data in client state (use React Query)
- Rely on color alone for information
- Create new functions in render
- Skip error handling for API calls
