package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildRegistersNoPromptHook pins the retired per-prompt injection: no
// target's generated hook surface registers the prompt command, and the unused
// PR-format instruction files no longer ship.
func TestBuildRegistersNoPromptHook(t *testing.T) {
	root := setupIsolatedRepositoryBuildRoot(t)
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatalf("build error = %v\n%s", err, stdout.String())
	}
	surfaces := map[string]string{
		"Claude Code hooks": filepath.Join(root, "plugins", "loaf", "hooks", "hooks.json"),
		"Cursor hooks":      filepath.Join(root, "dist", "cursor", "hooks.json"),
		"Codex hooks":       filepath.Join(root, "dist", "codex", ".codex", "hooks.json"),
		"OpenCode plugin":   filepath.Join(root, "dist", "opencode", "plugins", "hooks.js"),
		"Amp plugin":        filepath.Join(root, "dist", "amp", ".amp", "plugins", "loaf.js"),
	}
	for surface, path := range surfaces {
		body := readBuildFileString(t, path)
		for _, token := range []string{"for-prompt", "session-context-inject", "UserPromptSubmit", "userpromptsubmit"} {
			if strings.Contains(body, token) {
				t.Errorf("generated %s registers retired prompt hook token %q", surface, token)
			}
		}
	}
	for _, dir := range []string{
		filepath.Join(root, "plugins", "loaf", "hooks", "instructions"),
		filepath.Join(root, "dist", "cursor", "hooks", "instructions"),
		filepath.Join(root, "dist", "opencode", "plugins", "hooks", "instructions"),
	} {
		for _, name := range []string{"pre-pr-checklist.md", "pre-pr-format.md"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				t.Errorf("unused instruction file %s still ships in %s", name, dir)
			}
		}
	}
}
