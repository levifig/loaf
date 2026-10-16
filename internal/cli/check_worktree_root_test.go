package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Harnesses can spawn hooks from the main checkout while the session's shell
// works in a linked worktree on another branch. These fixtures keep the hook
// process in the main checkout and name the worktree only in the payload cwd.

func linkedWorktree(t *testing.T, primary, branch string) string {
	t.Helper()
	worktree := filepath.Join(realpath(t, t.TempDir()), branch)
	gitCLI(t, primary, "worktree", "add", "-b", branch, worktree, "main")
	return worktree
}

func worktreeHookPayload(cwd, command string) string {
	body, err := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"cwd":             cwd,
		"tool_name":       "Bash",
		"tool_input":      map[string]string{"command": command},
	})
	if err != nil {
		panic(err)
	}
	return string(body)
}

func runWorktreeHook(t *testing.T, primary, cwd, hook, command string, args ...string) (checkJSONOutput, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := (Runner{WorkingDir: primary, StateHome: t.TempDir(), Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(worktreeHookPayload(cwd, command))}).Run(append([]string{"check", "--hook", hook}, args...))
	var result checkJSONOutput
	if stdout.Len() > 0 {
		_ = json.Unmarshal(stdout.Bytes(), &result)
	}
	return result, stderr.String(), err
}

func TestWorkflowPrePRChecksInvokingWorktreeCommittedHEAD(t *testing.T) {
	primary := initCLIGitRepo(t)
	trackLoafRemote(t, primary)
	mkdirAll(t, filepath.Join(primary, ".github"))
	writeFile(t, filepath.Join(primary, ".github", "PULL_REQUEST_TEMPLATE.md"), coverageDeclaration)
	writeFile(t, filepath.Join(primary, "CHANGELOG.md"), coverageChangelog("- Old unrelated note."))
	coverageCommit(t, primary)
	covered := coverageChangelog("- Old unrelated note.\n- Enforce coverage ([#284](https://github.com/levifig/loaf/issues/284)).")

	worktree := linkedWorktree(t, primary, "feature")
	writeFile(t, filepath.Join(worktree, "CHANGELOG.md"), covered)
	coverageCommit(t, worktree)
	// A relative --body-file belongs to the shell's directory, below the root.
	subdir := filepath.Join(worktree, "docs")
	mkdirAll(t, subdir)
	writeFile(t, filepath.Join(subdir, "pr.md"), "## Native work\n\n"+coverageDeclaration+"\n")

	// gh opens a relative body from the physical shell directory, so a symlink
	// followed by .. leaves where a lexical join would land. Decoys sit at the
	// lexical destinations and declare an Issue the worktree does not cover.
	decoy := "<!-- loaf:implemented-issues {\"issues\":[\"https://github.com/levifig/loaf/issues/999\"]} -->\n"
	writeFile(t, filepath.Join(worktree, "pr.md"), "## Native work\n\n"+coverageDeclaration+"\n")
	aliases := filepath.Join(realpath(t, t.TempDir()), "aliases")
	mkdirAll(t, aliases)
	writeFile(t, filepath.Join(aliases, "pr.md"), decoy)
	alias := filepath.Join(aliases, "session")
	if err := os.Symlink(subdir, alias); err != nil {
		t.Fatal(err)
	}
	mkdirAll(t, filepath.Join(worktree, "deep", "inner"))
	writeFile(t, filepath.Join(worktree, "deep", "pr.md"), decoy)
	if err := os.Symlink(filepath.Join(worktree, "deep", "inner"), filepath.Join(subdir, "nested")); err != nil {
		t.Fatal(err)
	}

	// Coverage only in the working tree is not committed HEAD.
	uncommitted := linkedWorktree(t, primary, "uncommitted")
	writeFile(t, filepath.Join(uncommitted, "CHANGELOG.md"), covered)

	literal := `gh pr create --repo levifig/loaf --base main --title "fix: enforce coverage" --body '` + coverageDeclaration + `'`
	bodyFile := func(path string) string {
		return `gh pr create --repo levifig/loaf --base main --title "fix: enforce coverage" --body-file ` + path
	}
	for _, tc := range []struct {
		name, cwd, command string
		blockedFor         string
	}{
		{name: "worktree root", cwd: worktree, command: literal},
		{name: "worktree subdirectory", cwd: subdir, command: bodyFile("pr.md")},
		{name: "symlinked shell directory", cwd: alias, command: bodyFile("../pr.md")},
		{name: "symlink then parent in body path", cwd: subdir, command: bodyFile("nested/../pr.md"), blockedFor: "999"},
		{name: "uncommitted coverage", cwd: uncommitted, command: literal, blockedFor: "284"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err := runWorktreeHook(t, primary, tc.cwd, "workflow-pre-pr", tc.command)
			var exitErr ExitError
			if tc.blockedFor != "" {
				if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr, "issues/"+tc.blockedFor) {
					t.Fatalf("want block for issue %s: error=%v stderr=%q", tc.blockedFor, err, stderr)
				}
				return
			}
			if err != nil {
				t.Fatalf("worktree HEAD with coverage blocked (main checkout read instead?): error=%v stderr=%q", err, stderr)
			}
		})
	}
}

func TestWorkflowPrePRAdvisoryReadsInvokingWorktreeCommittedHEAD(t *testing.T) {
	// A repository that has not opted into issue coverage, whose main branch
	// carries no [Unreleased] section between releases.
	primary := initCLIGitRepo(t)
	trackLoafRemote(t, primary)
	writeFile(t, filepath.Join(primary, "CHANGELOG.md"), "# Changelog\n\n## [1.0.0] - 2026-01-01\n\n- Initial release.\n")
	coverageCommit(t, primary)
	unreleased := coverageChangelog("- Patch a vulnerable dependency.")

	worktree := linkedWorktree(t, primary, "feature")
	writeFile(t, filepath.Join(worktree, "CHANGELOG.md"), unreleased)
	coverageCommit(t, worktree)
	uncommitted := linkedWorktree(t, primary, "uncommitted")
	writeFile(t, filepath.Join(uncommitted, "CHANGELOG.md"), unreleased)

	command := `gh pr create --repo levifig/loaf --base main --title "fix: patch dependency" --body "Patches a dependency."`
	for _, tc := range []struct {
		name, cwd   string
		wantWarning bool
	}{
		{name: "committed section", cwd: worktree},
		{name: "uncommitted section", cwd: uncommitted, wantWarning: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, stderr, err := runWorktreeHook(t, primary, tc.cwd, "workflow-pre-pr", command, "--json")
			if err != nil || result.Blocked {
				t.Fatalf("advisory pre-PR blocked: error=%v result=%+v stderr=%q", err, result, stderr)
			}
			warned := strings.Contains(strings.Join(result.Warnings, "\n"), "[Unreleased]")
			if warned != tc.wantWarning {
				t.Fatalf("[Unreleased] warning = %v, want %v: %+v", warned, tc.wantWarning, result.Warnings)
			}
		})
	}
}

func TestArtifactNamesScansInvokingWorktree(t *testing.T) {
	primary := initCLIGitRepo(t)
	worktree := linkedWorktree(t, primary, "feature")
	mkdirAll(t, filepath.Join(worktree, ".agents", "reports"))
	writeFile(t, filepath.Join(worktree, ".agents", "reports", "TASK-074-notes.md"), "body\n")
	gitCLI(t, worktree, "add", ".agents")

	result, stderr, err := runWorktreeHook(t, primary, worktree, "artifact-names", "git commit -m notes", "--json")
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(strings.Join(result.Errors, "\n"), "TASK-074-notes.md") {
		t.Fatalf("worktree artifact name not scanned: error=%v result=%+v stderr=%q", err, result, stderr)
	}
}

func TestCheckHookRejectsInaccessiblePayloadCwd(t *testing.T) {
	primary := initCLIGitRepo(t)
	for _, cwd := range []string{"relative/dir", filepath.Join(primary, "missing")} {
		_, stderr, err := runWorktreeHook(t, primary, cwd, "workflow-pre-pr", `gh pr create --repo levifig/loaf --base main --title "fix: x" --body x`)
		var exitErr ExitError
		if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr, "cwd") {
			t.Fatalf("cwd %q was not rejected: error=%v stderr=%q", cwd, err, stderr)
		}
	}
}

func TestKbStalenessNudgeReadsInvokingWorktree(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("TOOL_INPUT", "")
	primary := initCLIGitRepo(t)
	worktree := linkedWorktree(t, primary, "feature")
	mkdirAll(t, filepath.Join(worktree, "src"))
	writeFile(t, filepath.Join(worktree, "src", "main.go"), "package main\n")
	gitCLI(t, worktree, "add", "src/main.go")
	gitCLI(t, worktree, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "add go file")
	mkdirAll(t, filepath.Join(worktree, "docs", "knowledge"))
	writeFile(t, filepath.Join(worktree, "docs", "knowledge", "go.md"), "---\ntopics:\n  - go\nlast_reviewed: 2000-01-01\ncovers:\n  - src/**/*.go\n---\n# Go\n")

	payload, err := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse",
		"session_id":      "worktree",
		"cwd":             worktree,
		"tool_name":       "Edit",
		"tool_input":      map[string]string{"file_path": filepath.Join(worktree, "src", "main.go")},
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = (Runner{WorkingDir: primary, Stdout: &stdout, Stderr: &stderr, Stdin: bytes.NewReader(payload)}).Run([]string{"check", "--hook", "kb-staleness-nudge"})
	if err != nil || !strings.Contains(stdout.String(), "docs/knowledge/go.md") {
		t.Fatalf("worktree edit missed its knowledge nudge: error=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
}

func TestWorkflowPrePRBlocksWhenOnlyMainCheckoutHasCoverage(t *testing.T) {
	primary := initCLIGitRepo(t)
	trackLoafRemote(t, primary)
	mkdirAll(t, filepath.Join(primary, ".github"))
	writeFile(t, filepath.Join(primary, ".github", "PULL_REQUEST_TEMPLATE.md"), coverageDeclaration)
	writeFile(t, filepath.Join(primary, "CHANGELOG.md"), coverageChangelog("- Old unrelated note."))
	coverageCommit(t, primary)
	worktree := linkedWorktree(t, primary, "feature")
	gitCLI(t, primary, "switch", "-c", "covered")
	writeFile(t, filepath.Join(primary, "CHANGELOG.md"), coverageChangelog("- Old unrelated note.\n- Enforce coverage ([#284](https://github.com/levifig/loaf/issues/284))."))
	coverageCommit(t, primary)

	_, stderr, err := runWorktreeHook(t, primary, worktree, "workflow-pre-pr", `gh pr create --repo levifig/loaf --base main --title "fix: enforce coverage" --body '`+coverageDeclaration+`'`)
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr, "284") {
		t.Fatalf("main checkout's coverage satisfied the worktree: error=%v stderr=%q", err, stderr)
	}
}

func TestCheckHookOutsideGitKeepsHookProcessCheckout(t *testing.T) {
	// An opted-in project whose branch lacks coverage must not fall to the
	// advisory path because the shell sits in a directory outside Git.
	primary := coverageRepo(t)
	outside := realpath(t, t.TempDir())
	for _, tc := range []struct{ command, wantErr string }{
		{`gh pr create --repo levifig/loaf --base main --title "fix: enforce coverage" --body '` + coverageDeclaration + `'`, "issues/284"},
		{`gh pr create --repo levifig/loaf --base main --head feature --title "fix: enforce coverage" --body '` + coverageDeclaration + `'`, "--head"},
		// Without --repo, only the fallback can select the opted-in checkout.
		{`gh pr create --base main --title "fix: enforce coverage" --body '` + coverageDeclaration + `'`, "explicit --repo"},
	} {
		_, stderr, err := runWorktreeHook(t, primary, outside, "workflow-pre-pr", tc.command)
		var exitErr ExitError
		if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr, tc.wantErr) {
			t.Fatalf("outside-Git cwd disabled enforcement for %q: error=%v stderr=%q", tc.command, err, stderr)
		}
	}
}

func TestWorkflowPrePRBindsTargetRepository(t *testing.T) {
	// Policy belongs to the repository the PR targets: the --repo value as
	// the strict parser reads it, else the command's checkout. A target that
	// cannot be read or bound must not fall to advisory while either the
	// command's or the session's checkout opts in.
	opted := coverageRepo(t)
	unrelated := initCLIGitRepo(t)
	other := initCLIGitRepo(t)
	gitCLI(t, other, "remote", "add", "origin", "git@github.com:example/other.git")
	untracked := initCLIGitRepo(t)
	sshOpted := coverageRepo(t)
	gitCLI(t, sshOpted, "remote", "set-url", "loaf", "ssh://git@ssh.github.com:443/levifig/loaf.git")
	enterpriseOpted := coverageRepo(t)
	gitCLI(t, enterpriseOpted, "remote", "set-url", "loaf", "https://ghe.example.com/levifig/loaf.git")
	// Filesystem remotes whose path ends in levifig/loaf are not that hosted
	// repository, however a permissive normalizer reads them.
	absolutePath := initCLIGitRepo(t)
	gitCLI(t, absolutePath, "remote", "add", "origin", "/srv/levifig/loaf")
	relativePath := initCLIGitRepo(t)
	gitCLI(t, relativePath, "remote", "add", "origin", "srv/levifig/loaf")
	pr := func(flags string) string {
		return `gh pr create ` + flags + ` --base main --title "fix: enforce coverage" --body '` + coverageDeclaration + `'`
	}
	for _, tc := range []struct {
		name, process, cwd, command, wantErr string
	}{
		{name: "unrelated checkout cannot stand in for the target", process: opted, cwd: unrelated, command: pr("--repo levifig/loaf"), wantErr: "issues/284"},
		{name: "unsupported flags still block from an unrelated checkout", process: opted, cwd: unrelated, command: pr("--repo levifig/loaf --head feature"), wantErr: "--head"},
		{name: "target tracked by the command's checkout", process: untracked, cwd: opted, command: pr("--repo levifig/loaf"), wantErr: "issues/284"},
		{name: "separate repository's own PR follows its policy", process: opted, cwd: other, command: pr("--repo example/other")},
		{name: "implicit target is the command's checkout", process: opted, cwd: other, command: `gh pr create --base main --title "fix: x" --body x`},
		{name: "flag value naming a repository is not a target", process: other, cwd: opted, command: `gh pr create --base main --title '--repo=example/other' --body x`, wantErr: "explicit --repo"},
		{name: "bundled short flags are unreadable", process: opted, cwd: unrelated, command: `gh pr create -dRlevifig/loaf --base main --title "fix: x" --body x`, wantErr: "unsupported"},
		{name: "GH_REPO prefix is unreadable", process: opted, cwd: unrelated, command: `GH_REPO=levifig/loaf gh pr create --base main --title "fix: x" --body x`, wantErr: "ambiguous context"},
		{name: "dynamic target is unreadable", process: opted, cwd: unrelated, command: pr(`--repo "$REPO"`), wantErr: "literal"},
		{name: "untracked target with an opted-in checkout", process: opted, cwd: unrelated, command: pr("--repo example/nowhere"), wantErr: "neither"},
		{name: "untracked target without any policy", process: untracked, cwd: unrelated, command: pr("--repo example/nowhere")},
		{name: "ssh.github.com remote binds", process: untracked, cwd: sshOpted, command: pr("--repo levifig/loaf"), wantErr: "issues/284"},
		{name: "owner/repo binds on any host", process: untracked, cwd: enterpriseOpted, command: pr("--repo levifig/loaf"), wantErr: "issues/284"},
		{name: "repeated target is unreadable", process: opted, cwd: other, command: pr("--repo example/other --repo levifig/loaf"), wantErr: "repeated"},
		{name: "unquoted expansion in a skipped value", process: opted, cwd: other, command: `gh pr create --title $TITLE --base main --body x`, wantErr: "shell expansion"},
		{name: "quoted multi-word expansion in a skipped value", process: opted, cwd: other, command: `gh pr create --title "$@" --base main --body x`, wantErr: "shell expansion"},
		{name: "zsh glob qualifier in a skipped value", process: opted, cwd: other, command: `gh pr create --title /dev/fd/<1-2>(:s@/dev/fd/@--repo=levifig/loaf@) --base main --body x`, wantErr: "shell expansion"},
		{name: "quoted zsh splitting expansion in a skipped value", process: opted, cwd: other, command: `gh pr create --title "${(@s:|:)${:-fix: x|--repo|levifig/loaf}}" --base main --body x`, wantErr: "shell expansion"},
		{name: "absolute filesystem remote does not bind", process: opted, cwd: absolutePath, command: pr("--repo levifig/loaf"), wantErr: "issues/284"},
		{name: "relative filesystem remote does not bind", process: opted, cwd: relativePath, command: pr("--repo levifig/loaf"), wantErr: "issues/284"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err := runWorktreeHook(t, tc.process, tc.cwd, "workflow-pre-pr", tc.command)
			var exitErr ExitError
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("blocked: error=%v stderr=%q", err, stderr)
				}
				return
			}
			if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("want block mentioning %q: error=%v stderr=%q", tc.wantErr, err, stderr)
			}
		})
	}
}

func TestWorkflowPrePRAdvisoryWarnsOnUnbornHEAD(t *testing.T) {
	primary := initCLIGitRepo(t)
	unborn := realpath(t, t.TempDir())
	gitCLI(t, unborn, "init", "-b", "main")
	trackLoafRemote(t, unborn)
	writeFile(t, filepath.Join(unborn, "CHANGELOG.md"), coverageChangelog("- Uncommitted note."))

	result, stderr, err := runWorktreeHook(t, primary, unborn, "workflow-pre-pr", `gh pr create --repo levifig/loaf --base main --title "fix: x" --body "x"`, "--json")
	if err != nil || result.Blocked || !strings.Contains(strings.Join(result.Warnings, "\n"), "not found in committed HEAD") {
		t.Fatalf("unborn HEAD: error=%v result=%+v stderr=%q", err, result, stderr)
	}
}
