# Motion

## Timing Tokens

| Token | Duration | Use |
|-------|----------|-----|
| `fast` | 150ms | Hover, focus |
| `base` | 300ms | Standard transitions |
| `slow` | 500ms | Complex animations |

## Easing Tokens

| Token | Use |
|-------|-----|
| `easeOut` | Entrances (fast start, slow end) |
| `easeIn` | Exits (slow start, fast end) |
| `easeInOut` | Emphasis (symmetrical) |

## Rules

- Only animate `transform` and `opacity` (GPU-accelerated)
- Always respect `prefers-reduced-motion: reduce`
- No flashing > 3 times/second

## Checklist

- [ ] Animation serves a clear purpose
- [ ] Duration appropriate for element size
- [ ] Only animates transform/opacity
- [ ] Respects `prefers-reduced-motion`
- [ ] No flashing (> 3 times per second)
