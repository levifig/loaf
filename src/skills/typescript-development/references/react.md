# React with TypeScript

## Component Conventions

- Functional components with explicit prop interfaces
- Default export for pages, named export for components
- Generic components for reusable lists/tables
- Context pattern: `createContext<T | undefined>(undefined)` + custom hook with runtime check

## Hooks Conventions

- Explicit types for `useState` when not inferable (e.g., `useState<User | null>(null)`)
- Always provide cleanup in `useEffect`
- Custom hooks return typed result objects

## Critical Rules

### Always
- Define explicit prop interfaces
- Use functional components with hooks
- Memoize expensive computations (`useMemo`, `React.memo`)
- Use Context for shared state within a subtree

### Never
- Use `any` for component props
- Mutate state directly
- Forget cleanup in `useEffect`
- Skip `key` prop in lists
