# Skill Portability

## Current Model

Loaf authors skills as `SKILL.md` packages following the [Agent Skills specification](https://agentskills.io/specification). One shared instruction body carries the method across supported harnesses. Builders adapt packaging and native metadata, not the meaning or wording of portable skill prose.

Genuine product-specific facts remain explicit: native metadata belongs in authorized sidecars; instructions whose literal tools, paths, or invocation forms differ belong in labeled product sections or a compact harness table. Every target receives those same authored sections. Portability does not mean pretending all harnesses expose identical capabilities.

Skill descriptions are part of routing behavior: they are visible before the full instructions are loaded. Package validity and byte equality do not establish that a model will select or follow a skill correctly.

## Rationale and Tradeoffs

The common package format lets a methodological change be authored and reviewed once, while adapters preserve the native packaging and metadata a harness needs. Shared instructions and resources can be checked across outputs rather than reviewed as independent implementations of the same method.

Independently authored harness formats or a separate corpus per target would expose each product's full native format and reduce adapter logic, but duplicate methodology and multiply review and behavioral drift. Loaf accepts the cost of maintaining adapters and validating every affected output instead.

Per-target prose substitution is not an authoring mechanism. It can turn valid product-specific instructions into different or invalid instructions; labeled sections preserve the literal facts without silently changing shared behavior.

This model does not choose the CLI's implementation language or user-facing distribution channel. Node.js development tooling is compatible with shared skill authoring; runtime and native distribution decisions remain separate in the [architecture overview](../ARCHITECTURE.md#build-and-distribution).

## Constraints

- Keep one authored body for each skill, with its references and templates. Do not introduce a target identity or second-stage prose rewrite on the skill-copy path.
- Restrict target-specific skill frontmatter differences to the fields and exact values that the target's sidecar authorizes. Other frontmatter and skill resources remain comparable across targets; build-version stamps must agree.
- Validate both the common package and each target adaptation when shared content changes. Cross-target equality complements, rather than replaces, native artifact validation and routing evaluation.
- Treat the supported target roster as implementation detail, not a permanently fixed architectural choice. Adding a target must preserve the shared authoring contract.

## Implementation and Boundaries

Shared source packages live under [`content/skills/`](../../content/skills/). Tracker-native Flow and provider sources currently live under [`vnext/content/`](../../vnext/content/) and replace the older Flow copies in the common `dist/skills/` intermediate. The [shared intermediate builder](../../internal/cli/build_codex.go) also projects shared templates into consuming packages and adjusts their links before target packaging. That common path projection is not a harness-specific prose rewrite; the current source split is an integration boundary, not independent per-target authorship.

Target builders under [`internal/cli/`](../../internal/cli/) produce the outputs listed in [`config/targets.yaml`](../../config/targets.yaml). The [skill invariance checker](../../internal/cli/build_skill_invariance.go) compares skill paths, bodies, resources, retained frontmatter, authorized sidecar values, and version stamps. [Build tests](../../internal/cli/build_test.go) exercise cross-target equality, reject unauthorized metadata differences, preserve labeled sections, probe for prose substitution, and verify tracker-native content packaging.

The shared build path and its invariants are implemented. This is not a claim that every generated integration or model behavior is proven in a live harness: package checks, target runtime smoke tests, and description-routing evaluation establish different things.

For authoring layout and templates, see [Skill Architecture](../knowledge/skill-architecture.md). For build commands, output paths, adapter details, and verification tools, see [Build System](../knowledge/build-system.md). Those guides carry operational detail; this topic owns the portability model and rationale.
