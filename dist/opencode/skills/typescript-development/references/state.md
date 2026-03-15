# State Management

## Decision Guide

| Scenario | Solution |
|----------|----------|
| API/server data | React Query |
| Global client state | Zustand |
| Shareable state (URL) | `searchParams` |
| Component subtree | Context + `useReducer` |
| Single component | `useState` |
| Persistent | localStorage hook |

## Conventions

### Zustand
- Use `persist` middleware for auth tokens
- Use `partialize` to persist only specific fields
- Always use selectors for performance: `useStore((s) => s.field)`
- Use slices pattern for large stores

### React Query
- Query keys factory pattern: `userKeys.detail(id)` returns `["users", "detail", id]`
- `staleTime: 5 * 60 * 1000` for standard data
- Optimistic updates with `onMutate` → `onError` rollback → `onSettled` invalidate

### URL State
- Use `useSearchParams` for filters, sort, pagination
- Keep URL state serializable

## Critical Rules

- Use React Query for server state, Zustand for client state — never mix
- Keep state as local as possible
- Type all state and actions
- Never store API data in Zustand
