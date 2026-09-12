# Amp Native Delegation

## Adapter Contract

The managed Amp plugin exposes `loaf_delegate` for one bounded implementation turn against a native tracker contract. This is an Amp-specific local adapter, not a generic delegation framework. Native Amp Low, Medium, High, and Ultra remain the user-facing modes. Native Oracle and Subagents Auto stay Amp-owned.

- The main agent investigates, plans, assesses, coordinates, uses the tracker, and runs tests. The child does not read or mutate the provider.
- All implementation, including fixes and test writing, goes to a fresh Grok 4.6 Fast child. The requested configuration is `model: 'xai/grok-4.6'` with `features: ['fast']` and no `reasoningEffort`. Parent settings are not proof of the backend model, and failure does not authorize another model.
- The caller supplies the current canonical local Git worktree root. The adapter verifies it against both Amp's workspace root and `git rev-parse --show-toplevel` before creating the child. Other worktrees and remote Orbs are not supported.
- Only a currently parentless builtin native mode may call `loaf_delegate`. Currently parented, custom, and unknown callers fail preflight without spawning a child. The adapter checks the current reported parent and builtin definition; it does not attest lifetime ancestry. An orphaned builtin indistinguishable from a parentless builtin can qualify. Our Grok child remains custom with Read and apply_patch only, including if its parent is deleted.
- A missing or incompatible delegate does not authorize local implementation.
- Reload or restart loads changed plugin bytes, but merely restarting does not rewrite an existing custom-mode definition. Select builtin Low, Medium, High, or Ultra, or start a new builtin thread.

## Implementation Boundary

An implementation child receives only `Read` and `apply_patch`. It has no shell, provider, or delegation tools. Read paths and every `apply_patch` file header and `Move to` header must name an absolute path. The adapter rejects paths outside the canonical worktree, paths through escaping symlinks, and Git metadata paths.

The caller coordinates one writer and performs no concurrent parent writes. A plugin-runtime writer guard rejects another implementation child while ownership is active or uncertain, and the child's guard is sealed after its turn.

This is a trusted-host, single-plugin-runtime guard. It is not an OS sandbox or a cross-process lock, so coordination outside that runtime remains the caller's responsibility.

If the parent becomes idle or errors while delegation is pending, the adapter seals the child and requests cancellation. An unresolved submission or unconfirmed stop returns `uncertain` and retains writer ownership. Inspect the returned child before restarting Amp or attempting other writes; a cancellation request is not proof the child stopped. Plugin crashes and reloads lose these in-memory guards.

## Review and Acceptance

Native Oracle is the only read-only review and advisory path. `loaf_delegate` rejects `role: 'review'` without spawning a child. The Review button's existing user request, including merge-base and untracked files, remains unchanged. Review-only requests do not authorize fixes; authorized findings go to a fresh Grok worker, not the reviewer.

The main agent owns shell commands, tests, provider operations, authoritative result inspection, and acceptance against the same live tracker contract. Match the returned child identity and packet SHA-256 against the packet, then inspect the actual result; the receipt is evidence, not acceptance or independent attestation.

The only successful completion signal is Amp's `waitForResponse` observation of the child turn's running-to-idle transition, reported as `turn-complete`. That signal proves the turn completed, not that its output is correct or accepted. Repeated Ship should reuse valid contract, diff, test, and review evidence, rerun only stale or missing checks, avoid duplicate comments, and never infer commit, push, or merge authority.

## Invocation

After reading the live contract, call the exposed `loaf_delegate` tool with `role: 'implementation'`, `native_ref`, the absolute `worktree`, and the complete `packet`. Use the delegation contract for packet contents; include owned paths and acceptance criteria. Keep `role: 'implementation'` for compatibility; do not send review through this tool.

If preflight returns `incompatible`, report the missing capability or unsupported caller. Do not silently choose another mode, broaden tools, implement locally, or route to an unrestricted child. After installing or upgrading the managed plugin, reload plugins or restart Amp so the new bytes are loaded. Plugin crashes, reloads, and in-memory writer guards remain lifecycle limitations of this adapter.
