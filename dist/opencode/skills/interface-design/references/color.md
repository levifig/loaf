# Color

## Palette Structure

- **Primitives**: Full spectrum per hue (50–900 shades)
- **Semantic tokens**: `brand.primary`, `text.primary/secondary/inverse`, `status.success/warning/danger/info`

## Dark Mode

| Token | Light | Dark |
|-------|-------|------|
| `background.primary` | `#ffffff` | `#0a0a0a` |
| `background.secondary` | — | `#1a1a1a` |
| `background.tertiary` | — | `#2a2a2a` |
| `text.primary` | `#1a1a1a` | `#f5f5f5` (15:1) |
| `text.secondary` | — | `#a3a3a3` (7:1) |

Implementation: CSS custom properties with `[data-theme="dark"]` selector.

## Checklist

- [ ] Text meets 4.5:1 contrast (AA)
- [ ] Large text meets 3:1 contrast
- [ ] Interactive elements meet 3:1 contrast
- [ ] Focus indicators meet 3:1 contrast
- [ ] Color is not sole indicator of meaning (add icon/text)
- [ ] Works in light and dark modes
- [ ] All colors from design tokens
