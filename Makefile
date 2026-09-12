# Loaf development entry points. Everything runs through Go; there is no npm.
#   make / make build  compile the native binary, regenerate the CLI reference, build content, verify
#   make build-cli compile the native executable only
#   make install   complete build and verification, then activate this checkout
#   make test      go test ./...
#   make verify-local  comprehensive local checks, without activating the development launcher
#   make ci-smoke      bounded remote spot checks, also runnable locally
.PHONY: build build-cli build-go install verify release package test typecheck vet capability-tests verify-local verify-generated cgo-free ci-check ci-smoke release-smoke vulncheck clean

GO ?= go

build:
	$(GO) run ./cmd/loafdev build

build-cli:
	$(GO) run ./cmd/loafdev build-cli

build-go:
	$(GO) run ./cmd/loafdev build-go

install:
	$(GO) run ./cmd/loafdev install

verify:
	$(GO) run ./cmd/loafdev verify-artifacts

release:
	$(GO) run ./cmd/loafdev release

package:
	$(GO) run ./cmd/loafdev package

test:
	$(GO) test -count=1 ./...

typecheck:
	$(GO) test ./... -run=^$$

vet:
	$(GO) vet ./...

cgo-free:
	CGO_ENABLED=0 $(GO) build ./...

# Run sequentially even under make -j: tests do not depend on a prior build,
# and generated-content checks must read the completed build.
verify-local:
	$(MAKE) test
	$(MAKE) typecheck
	$(MAKE) vet
	$(MAKE) cgo-free
	$(MAKE) capability-tests
	$(MAKE) verify-generated

# tsc is an existing optional build tool; comprehensive verification requires
# it rather than accepting the normal interactive build's skip warning.
verify-generated:
	@command -v tsc >/dev/null || { echo "verify-generated requires the existing TypeScript compiler (tsc) on PATH" >&2; exit 1; }
	LOAF_DEV_LINK=0 LOAF_VALIDATE_TYPESCRIPT=1 $(MAKE) build
	bin/loaf check --hook render-drift --json

# Exact names make additions deliberate. These spot checks cover native command
# dispatch, SQLite recovery, the clean-machine policy regression, and delivery
# integrity. The full suites and all adapter runner tests belong in verify-local.
ci-check:
	$(MAKE) ci-smoke
	$(MAKE) cgo-free
	$(MAKE) verify-generated
	git diff --exit-code -- dist/ plugins/ .claude-plugin/ content/skills/loaf-reference/

ci-smoke:
	$(GO) test -count=1 -timeout=2m ./cmd/loaf -run '^(TestPublicBinaryDispatchesStateVersionAndReleasePreflightNatively|TestReleaseWorkflowVerifiesEvidenceBeforeStampedBuild)$$'
	$(GO) test -count=1 -timeout=2m ./internal/state -run '^TestBackupReportsRecoveryMetadataAndJournalWatermark$$'
	$(GO) test -count=1 -timeout=2m ./internal/cli -run '^(TestScopedCodexPolicyFixtureWithoutInstalledLoaf|TestValidateCodexJournalExecutableRejectsMissingAndDisposablePaths|TestScopedCodexRuleUpgradeIgnoresSymlinkedGlobalGuidance|TestReviewScopedCodexPolicyRunner|TestScopedCodexPolicyApplyConverges|TestScopedCodexPolicyApplyRollsBackOnFault|TestScopedCodexPolicyPreservesLateEdits)$$'
	$(MAKE) release-smoke

release-smoke:
	$(GO) test -count=1 -timeout=2m ./internal/devtool -run '^(TestClassifyReleaseTag|TestPackageWritesArchivesAndChecksums|TestUpdateHomebrewFormulaGeneratesAuditSafeURLs|TestVerifyArtifactsRequiresPATHRuntimeAndForbidsPluginExecutables)$$'

# Network-backed local audit, separate from the deterministic verification gate.
# This preserves the former CI check without adding a project dependency.
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Harness capability runners still execute under Node; they drive external
# CLIs and read their JSON streams. They need no package install.
capability-tests:
	node --experimental-strip-types --test cli/scripts/smoke-claude-code-startup.test.mjs cli/scripts/smoke-codex-startup.test.mjs cli/scripts/smoke-opencode-request-context.test.mjs cli/scripts/preflight-cursor-agent-context.test.mjs internal/cli/amp_delegation.test.mjs

clean:
	rm -rf bin dist/release
