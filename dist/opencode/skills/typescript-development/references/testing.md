# Testing TypeScript Applications

## Stack

| Type | Tool |
|------|------|
| Unit/Integration | Vitest |
| Component | React Testing Library |
| API Mocking | MSW (Mock Service Worker) |
| E2E | Playwright |
| Coverage | v8/istanbul |

## Vitest Setup

- Environment: `jsdom`, globals enabled
- Setup file: `@testing-library/jest-dom` + `cleanup` in `afterEach`
- Mock `next/navigation` in setup file
- Coverage: v8 provider, text + html reporters
- Path alias: `@` → `/src`

## Conventions

- Use `userEvent` over `fireEvent` for realistic interactions
- Use `screen` queries (accessibility-first: `getByRole`, `getByLabelText`)
- Use MSW for API mocking (not `fetch` stubs)
- Wrap React Query hooks with `QueryClientProvider` (retry: false) in tests
- Use `waitFor` for async assertions
- Playwright for E2E: `baseURL` config, `webServer` auto-start, trace on retry

## Critical Rules

- Query by accessibility role first, `data-testid` as last resort
- No arbitrary `setTimeout` waits — use `waitFor`
- Clean up after each test
- No mocking React internals
