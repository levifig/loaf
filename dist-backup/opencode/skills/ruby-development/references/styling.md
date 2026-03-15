# Rails Styling with TailwindCSS

## Rails 8 Integration

TailwindCSS runs via standalone CLI (no Node.js required).

```bash
bin/rails tailwindcss:install
bin/dev  # Watches with Procfile.dev
```

## Conventions

- Utility-first — compose from utility classes, avoid custom CSS
- Mobile-first with breakpoint prefixes: `sm:`, `md:`, `lg:`, `xl:`
- Dark mode: `darkMode: 'class'` with `dark:` prefix utilities
- Extract only when patterns repeat 3+ times — prefer partials over `@apply`
- Use `focus:ring` for keyboard accessibility
- Use Tailwind's spacing scale (`p-4`, `mt-6`, not arbitrary values)
- Style helpers for dynamic variants (e.g., `status_badge` helper method)
