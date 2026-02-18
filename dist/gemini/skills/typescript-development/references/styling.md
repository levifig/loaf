# Styling with Tailwind CSS

## Stack

- **Tailwind CSS** with `darkMode: "class"`
- **CVA** (Class Variance Authority) for type-safe component variants
- **clsx + tailwind-merge** via `cn()` utility
- Plugins: `@tailwindcss/forms`, `@tailwindcss/typography`

## Conventions

- `cn()` utility: `twMerge(clsx(inputs))` for conditional + merged classes
- CVA for all variant components (Button, Badge, etc.)
  - `variants` for visual styles, `defaultVariants` for defaults
  - `compoundVariants` for cross-variant combinations
- Dark mode: `[data-theme="dark"]` class on root, `dark:` prefix utilities
- Responsive: Mobile-first (`sm:`, `md:`, `lg:`, `xl:`, `2xl:`)

## Critical Rules

### Always
- Use utility classes (not inline styles)
- Create variants with CVA
- Support dark mode
- Design mobile-first

### Never
- Hardcode colors/spacing (use design tokens)
- Skip responsive breakpoints
- Forget focus states
