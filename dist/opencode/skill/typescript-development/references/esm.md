# Modern JavaScript (ESM)

## JS vs TS Decision Guide

| Scenario | Recommendation |
|----------|----------------|
| New project, team knows TS | TypeScript |
| Quick script/automation | JavaScript |
| Library with public API | TypeScript |
| Legacy codebase | Gradual migration |
| Prototyping | JavaScript |
| Large team | TypeScript |

## ESM Conventions

- `"type": "module"` in `package.json`
- Always include `.js` file extensions in imports
- Use `exports` field for package entry points (with `import` + `types` conditions)
- Top-level `await` allowed in ESM
- `__dirname` equivalent: `dirname(fileURLToPath(import.meta.url))`

## JS-to-TS Migration Path

1. Add JSDoc types (`@param`, `@returns`, `@typedef`)
2. Enable `allowJs` + `checkJs` in tsconfig
3. Rename files incrementally `.js` → `.ts`
4. Enable strict flags gradually

## Critical Rules

- Use ESM (`import`/`export`), never CommonJS
- Always include `.js` extensions in ESM imports
- Use named exports for tree-shaking
- JSDoc for JavaScript, explicit types for TypeScript
