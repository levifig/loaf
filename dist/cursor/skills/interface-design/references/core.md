# Core Design Principles

## Token Architecture

### Layers

1. **Primitives**: Raw color/spacing/radius values (e.g., `blue.600`, `spacing.4`)
2. **Semantic tokens**: Meaningful aliases (`brand.primary`, `text.primary`, `background.primary`)

### Token Categories

1. **Color**: brand, text, background, border, status
2. **Spacing**: padding, margin, gap (4px base)
3. **Typography**: font size, weight, line height, letter spacing
4. **Shadow**: elevation levels
5. **Border**: width, radius, style
6. **Timing**: animation duration
7. **Easing**: animation timing functions
8. **Z-index**: semantic layer names

## Tool Stack

| Purpose | Tools |
|---------|-------|
| Design | Figma, Stark, Coolors |
| Tokens | Style Dictionary |
| Components | React + TypeScript + Tailwind |
| Docs | Storybook |
| Testing | axe DevTools, WAVE, Lighthouse, NVDA, VoiceOver |
