# Spacing

## Scale (8-Point Grid)

| Token | Value | Use |
|-------|-------|-----|
| 0.5 | 2px | Hairline |
| 1 | 4px | Minimal |
| 2 | 8px | Base unit |
| 3 | 12px | Small |
| 4 | 16px | Standard |
| 5 | 20px | Medium |
| 6 | 24px | Comfortable |
| 8 | 32px | Large |
| 10 | 40px | Section |
| 12 | 48px | Large section |
| 16 | 64px | Hero |
| 20 | 80px | Massive |
| 24 | 96px | Layout |

## Component Spacing Tokens

| Context | sm | md | lg |
|---------|----|----|-----|
| Padding | 8px | 16px | 24px |
| Gap | 8px | 16px | 24px |
| Section | 32px | 48px | 64px |

## Rules

- **Padding**: Internal (component owns). **Margin**: External (parent controls). **Gap**: Prefer for flex/grid.
- Use logical properties (`padding-inline`, `margin-block-end`) for RTL support
- Touch targets: minimum 44x44px, minimum 8px gap between targets
