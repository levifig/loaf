# Accessibility Review Checklist

## Quick Check (Every UI Change)

- [ ] Color contrast meets 4.5:1 ratio
- [ ] Interactive elements keyboard accessible
- [ ] Images have alt text
- [ ] Focus indicator visible

## Testing Workflow

1. **Automated scan** — axe DevTools
2. **Keyboard navigation** — Tab through everything
3. **Screen reader** — Test with VoiceOver or NVDA
4. **Zoom** — Test at 200% zoom
5. **Color** — Check contrast, disable color

## Testing Tools

| Tool | Use |
|------|-----|
| axe DevTools | Automated testing |
| WAVE | Visual feedback |
| Lighthouse | General audit |
| VoiceOver/NVDA | Screen reader testing |
| Keyboard only | Tab through interface |
