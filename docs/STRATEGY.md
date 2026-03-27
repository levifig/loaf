# Loaf Strategy

How we get from build-time framework to agentic CLI.

## Phased Approach

### Phase 1: Foundation (Current)
Establish the knowledge system and Shape Up structure on Loaf itself. Eat our own dog food.

- Set up `docs/knowledge/` and `docs/decisions/`
- Create VISION, STRATEGY, ARCHITECTURE docs
- Shape specs for CLI, knowledge management, task CLI
- Build the `loaf` CLI skeleton
- Create the `knowledge-base` skill

### Phase 2: Knowledge Management
Ship `loaf kb` — the core knowledge management feature.

- `loaf kb init`, `loaf kb check`, `loaf kb validate`, `loaf kb status`, `loaf kb review`
- Knowledge-base skill with hooks (SessionStart, PostToolUse, SessionEnd)
- QMD integration (wrapping collections/context for cross-project)
- `loaf kb import` with fuzzy search
- Staleness detection (`covers:` + git)

### Phase 3: CLI Maturity
Surface existing systems and add cross-harness distribution.

- `loaf task`, `loaf spec` — surface existing task/spec system
- `loaf install --to <target>` — multi-harness distribution
- `loaf sync` — sync config across tools
- TASKS.json as programmatic task index
- `loaf skill add/list/validate`

### Phase 4: Cross-Project Intelligence
Domain knowledge sharing and personal knowledge.

- Cross-project imports via QMD collections
- Personal KB auto-import
- Knowledge repos for cross-cutting domains
- Expertise tracking

### Phase 5: Autonomous Execution
The overnight implementation loop.

- Claude Code SDK + Codex SDK integration
- `loaf implement SPEC-XXX --overnight`
- Implement → review → iterate → commit → PR loop
- TUI / macOS GUI control surface
- Configurable models/settings per project via `.agents/loaf.json`

## Priorities

1. Knowledge system foundation (Phase 1) — we need this to build everything else well
2. `loaf kb check` (Phase 2) — the core innovation, highest value
3. CLI skeleton (Phase 2-3) — enables everything else
4. Cross-project sharing (Phase 4) — when we need it for GridSight modules
5. Overnight loop (Phase 5) — the ultimate vision
