# API Integration

## Stack

| Pattern | Use |
|---------|-----|
| Type-safe fetch wrapper | Custom `ApiClient` class |
| Server state | React Query |
| End-to-end types | tRPC (when applicable) |
| Real-time | TypedWebSocket wrapper |

## Conventions

- `ApiClient` class with typed `get<T>`, `post<T>`, `put<T>`, `delete<T>` methods
- Custom `ApiError` class with `status`, `statusText`, `data`
- React Query key factory pattern: `userKeys.detail(id)`
- Result type: `{ success: true; data: T } | { success: false; error: E }`

## Critical Rules

### Always
- Type all API requests/responses
- Handle errors explicitly with `ApiError`
- Use React Query for server state (not `useEffect` + `useState`)
- Implement request cancellation
- Add loading/error states

### Never
- Use `any` for API responses
- Store API data in local state (use React Query cache)
- Make requests in `useEffect`
