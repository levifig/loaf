# Go Idioms Reference

Project conventions for idiomatic Go patterns.

## Key Idioms

| Idiom | Convention |
|-------|-----------|
| Guard clauses | Return early; keep happy path unindented; omit `else` after `return` |
| Comma-ok | Always use for type assertions, map access, channel receive |
| Defer | Defer cleanup immediately after acquiring resource; LIFO order |
| Interface checks | Compile-time: `var _ Interface = (*Type)(nil)` |
| Embedding | Composition only, never as inheritance |
| Functional options | `type Option func(*Config)` for flexible constructors |

## Method Receivers

| Use Pointer | Use Value |
|-------------|-----------|
| Modifies receiver | Doesn't modify |
| Large struct | Small type |
| Any method needs pointer | Maps, channels |

**Consistency:** If any method needs a pointer receiver, use pointer for all methods on that type.

## Always/Never

| Always | Never |
|--------|-------|
| Return early, omit else | Nest deeply with else chains |
| Use comma-ok for safe access | Assume map key exists |
| Defer cleanup immediately after acquiring | Defer in loops (accumulates) |
| Check interface satisfaction at compile time | Discover interface bugs at runtime |
| Be consistent with receiver types | Mix pointer/value arbitrarily |
| Use embedding for composition | Use embedding as inheritance |
