---
id: TASK-083
title: Amp target
spec: SPEC-020
status: todo
priority: p1
dependencies: [TASK-081]
track: D
---

# TASK-083: Amp target

New build target for Amp (experimental).

## Scope

Create `cli/lib/build/targets/amp.ts`. Output:
```
dist/amp/
├── skills/           # from shared intermediate, no sidecar merge
│   └── {skill-name}/
│       └── SKILL.md
└── plugins/
    └── loaf.ts       # generated runtime plugin
```

**Skills:** Call shared `copySkills()` — trivial copy from intermediate, no sidecar merge needed (Amp reads standard SKILL.md).

**Runtime plugin:** Call shared `generateRuntimePlugin()` with Amp adapter:
- Experimental header: `// @i-know-the-amp-plugin-api-is-wip-and-very-experimental-right-now`
- Maps `tool.call` and `tool.result` events
- No agents (Amp has native agents)
- No command generation (Amp auto-discovers skills)

**Registration:** Add to `cli/commands/build.ts`, `config/targets.yaml`, `cli/lib/detect/tools.ts`, `cli/lib/install/installer.ts`.

## Verification

- [ ] `loaf build --target amp` produces `dist/amp/skills/` + `dist/amp/plugins/loaf.ts`
- [ ] Plugin includes experimental header
- [ ] Plugin maps `tool.call`/`tool.result` events correctly
- [ ] Skills copied from intermediate (no sidecar merge)
- [ ] Registered in build, detect, install
- [ ] `loaf build` (all targets) still succeeds
