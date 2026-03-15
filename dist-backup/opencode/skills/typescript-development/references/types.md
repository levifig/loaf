# Advanced TypeScript Types

## Project Type Conventions

- **State machines**: Discriminated unions with `status` field
- **API responses**: `{ success: true; data: T } | { success: false; error: string }`
- **IDs**: Branded types (`type UserId = string & { __brand: "UserId" }`)
- **Deep partial**: `type DeepPartial<T> = T extends object ? { [K in keyof T]?: DeepPartial<T[K]> } : T`
- **JSON**: Recursive `JSONValue` type for untyped data

## Type Guard Patterns

- Type predicates: `function isUser(value: unknown): value is User`
- Assertion functions: `function assertIsUser(value: unknown): asserts value is User`
- Prefer `in` operator and discriminated union narrowing over type assertions

## Critical Rules

### Always
- Use discriminated unions for state
- Prefer type inference when obvious
- Create branded types for IDs
- Document complex types

### Never
- Use `any` (use `unknown` + type guards)
- Create overly complex nested types
- Use type assertions without guards
