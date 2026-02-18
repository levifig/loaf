# Next.js 14+

## App Router Conventions

```
app/
├── layout.tsx           # Root layout (required)
├── page.tsx             # Home page
├── loading.tsx          # Loading UI
├── error.tsx            # Error UI
├── not-found.tsx        # 404 page
├── (marketing)/         # Route group (no URL segment)
├── blog/[slug]/page.tsx # Dynamic routes
└── api/posts/route.ts   # API route handler
```

## Server vs Client Components

- **Default**: Server Components (no `"use client"` directive)
- Add `"use client"` only when using hooks, event handlers, or browser APIs
- Fetch data in Server Components, pass to Client Components as props
- Use `Suspense` boundaries for streaming

## Data Patterns

- **Server Actions** for mutations (`"use server"` + `revalidatePath`/`redirect`)
- **Parallel data fetching** with `Promise.all` in Server Components
- Type `params` and `searchParams` in page props

## Critical Rules

### Always
- Use Server Components by default
- Use Server Actions for mutations
- Implement `loading.tsx` and `error.tsx` per route
- Type route params

### Never
- Fetch in Client Components (use Server Components or React Query)
- Use `useState` for URL state (use `searchParams`)
- Import server code in client components
