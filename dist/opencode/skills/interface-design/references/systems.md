# Design Systems

## Architecture Layers

| Layer | Purpose | Example |
|-------|---------|---------|
| 1. Tokens | Atomic values | Colors, spacing, radius, z-index |
| 2. Primitives | Basic layout components | `Box`, `Stack`, `Grid` |
| 3. Components | UI building blocks | `Button`, `Input`, `Card` |
| 4. Patterns | Complex compositions | `ProfileCard`, `DataTable` |

## Component API Conventions

- **Variants**: Visual styles only (`primary`, `secondary`, `danger`, `ghost`)
- **Sizes**: Consistent scale (`sm`, `md`, `lg`)
- **Composition over monolith**: Use `Card > CardHeader + CardContent + CardFooter`, not `<Card title="" content="" buttons={[]}>`

## Z-Index Scale

| Token | Value | Use |
|-------|-------|-----|
| `dropdown` | 1000 | Dropdowns, popovers |
| `modal` | 1200 | Modal dialogs |
| `tooltip` | 1500 | Tooltips |
