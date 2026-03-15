# TypeScript Debugging

## Setup

- Enable `sourceMap: true` and `inlineSources: true` in tsconfig
- Node.js: `node --inspect` (or `--inspect-brk` to break on first line)
- With ts-node: `node --inspect -r ts-node/register src/server.ts`

## Tools

| Tool | Use |
|------|-----|
| `console.table()` | Tabular data |
| `console.dir(obj, { depth: null })` | Deep object inspection |
| `console.time()`/`timeEnd()` | Performance timing |
| `console.trace()` | Stack trace without error |
| Chrome DevTools conditional breakpoints | Break on specific conditions |
| Chrome DevTools logpoints | Log without pausing |

## Async Debugging

- Set `Error.stackTraceLimit = 50` for deeper async traces
- Label promises in `Promise.race` to identify which resolved
