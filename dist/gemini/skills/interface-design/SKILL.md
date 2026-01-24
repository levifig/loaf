---
name: interface-design
description: >-
  Covers UI/UX design, accessibility, and design systems. Includes design
  principles, color theory with WCAG contrast requirements, typography scales,
  8pt spacing grid, responsive breakpoints, WCAG 2.1 accessibility compliance,
  motion design, and design system governance. Use when designing interfaces,
  choosing colors, or when the user asks "is this accessible?" or "what spacing
  should I use?"
version: 1.15.0
---

# Design Principles

**Philosophy**: Design is problem-solving through systematic thinking, with accessibility and user needs at the center. Every design decision must be intentional, accessible, and meaningful.

## Quick Reference

| Topic | Key Principle | Reference |
|-------|--------------|-----------|
| Core | User-first, accessible by default, consistent, minimal | [references/core.md](references/core.md) |
| Color | Function first, 4.5:1 contrast minimum, never color-only | [references/color.md](references/color.md) |
| Typography | Legible, accessible, reinforce hierarchy | [references/typography.md](references/typography.md) |
| Spacing | 8pt grid, consistent rhythm, logical properties | [references/spacing.md](references/spacing.md) |
| Responsive | Mobile-first, content-out, fluid layouts | [references/responsive.md](references/responsive.md) |
| Accessibility | WCAG 2.1 AA minimum, keyboard navigable, screen reader compatible | [references/a11y.md](references/a11y.md) |
| Accessibility Review | WCAG 2.1 AA checklist, testing tools, common issues | [references/accessibility-review.md](references/accessibility-review.md) |
| Motion | Purposeful, natural, respect reduced motion | [references/motion.md](references/motion.md) |
| Systems | Tokens, components, patterns, governance | [references/systems.md](references/systems.md) |

## Core Principles

### 1. User-First
Every design decision must serve user needs and goals.
- Understand user context: Who, what, why, when, where?
- Validate assumptions with real users
- Measure outcomes that matter to users

### 2. Accessible by Default
Accessibility is not a feature - it is a requirement.
- WCAG 2.1 AA minimum, AAA as aspiration
- Keyboard navigation for all functionality
- 4.5:1 contrast for text, 3:1 for UI components
- Never use color as the sole indicator

### 3. Consistent
Consistency reduces cognitive load and builds trust.
- Design tokens as single source of truth
- Component reuse over reinvention
- Predictable naming conventions

### 4. Minimal
Every element must justify its existence.
- Progressive disclosure
- Clear visual hierarchy
- Sufficient whitespace

### 5. Delightful
Good design feels effortless.
- Smooth, purposeful interactions
- Thoughtful details
- Performance as a feature

## Design Tokens

Use semantic tokens, never hardcoded values.

```typescript
// Pattern: [category]-[property]-[variant?]-[state?]
color-text-primary
color-background-secondary
spacing-component-padding-md
```

## Critical Rules

1. **Never use pixel values directly** - Always use design tokens
2. **Never use divs for buttons** - Use semantic HTML
3. **Never skip alt text** - All images need descriptions
4. **Never use color alone** - Combine with icons/text
5. **Never ignore reduced motion** - Respect user preferences
6. **Never animate layout properties** - Use transform/opacity only
7. **Never skip focus indicators** - Minimum 3:1 contrast
8. **Never exceed 75ch line length** - Optimal readability at 65ch

## Quality Checklist

Every design deliverable must meet:

- [ ] WCAG 2.1 AA compliant
- [ ] Keyboard navigable
- [ ] Screen reader compatible
- [ ] Color contrast verified (4.5:1 text, 3:1 UI)
- [ ] Focus indicators visible (3:1 contrast)
- [ ] Reduced motion respected
- [ ] Uses design tokens exclusively
- [ ] Touch targets minimum 44x44px
- [ ] Mobile responsive
- [ ] Documented in component library
