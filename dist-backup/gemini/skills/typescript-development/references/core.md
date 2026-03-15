# TypeScript Core

## Stack

| Component | Default | Alternative |
|-----------|---------|-------------|
| Runtime | Node.js 20+ | Bun, Deno |
| Package Manager | pnpm | npm, yarn |
| Bundler | Vite | webpack, esbuild |
| Linter | ESLint | Biome |
| Formatter | Prettier | Biome |
| Test Runner | Vitest | Jest |

## Project Structure

```
src/
├── components/
├── lib/
│   ├── types/
│   ├── utils/
│   └── hooks/
├── app/              # Next.js App Router
└── index.ts
```

## TSConfig

Target ES2022, strict mode enabled. Key strict flags:
- `noUncheckedIndexedAccess: true`
- `noUnusedLocals: true`
- `noUnusedParameters: true`
- `noFallthroughCasesInSwitch: true`

Module resolution: `bundler`. Path aliases: `@/*` → `./src/*`.

## Conventions

- Use `satisfies` for type-checking without widening
- Use branded types for IDs: `type UserId = string & { __brand: "UserId" }`
- Use discriminated unions for state machines
- Use `unknown` over `any`, with type guards for narrowing
- Use `as const` for literal types
- Never disable strict checks or use `@ts-ignore`
