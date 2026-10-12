package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildUsesCanonicalContentAndIgnoresRetiredOverlays(t *testing.T) {
	root := setupBuildCommandLoafRoot(t)
	repo := testRepositoryRoot(t)
	for _, dir := range []string{"content", "config"} {
		if err := copyDirContentsForInstall(filepath.Join(repo, dir), filepath.Join(root, dir)); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(root, "content", "skills", "ship", "SKILL.md"), "---\nname: ship\ndescription: Reviews a candidate.\n---\nCanonical skill source.\n")
	writeFile(t, filepath.Join(root, "content", "agents", "background-runner.md"), "---\nname: background-runner\ndescription: Runs background work.\n---\nCanonical agent source.\n")
	// Stale source trees must neither override canonical content nor become
	// prerequisites for building it. No replacement runtime is present here.
	mkdirAll(t, filepath.Join(root, "vnext", "content", "skills", "ship"))
	mkdirAll(t, filepath.Join(root, "vnext", "content", "agents"))
	writeFile(t, filepath.Join(root, "vnext", "content", "skills", "ship", "SKILL.md"), "Retired skill shadow.\n")
	writeFile(t, filepath.Join(root, "vnext", "content", "agents", "background-runner.md"), "Retired agent shadow.\n")
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatalf("build without runtime isolation tree: %v\n%s", err, stdout.String())
	}
	for _, target := range defaultBuildTargets {
		body := readBuildFileString(t, filepath.Join(nativeBuildSkillTreeDir(root, target), "ship", "SKILL.md"))
		if !strings.Contains(body, "Canonical skill source.") || strings.Contains(body, "Retired skill shadow.") {
			t.Errorf("%s did not use the canonical skill source", target)
		}
	}
	for _, target := range []string{"claude-code", "opencode", "cursor"} {
		body := readBuildFileString(t, filepath.Join(nativeBuildTargetOutputDir(root, target), "agents", "background-runner.md"))
		if !strings.Contains(body, "Canonical agent source.") || strings.Contains(body, "Retired agent shadow.") {
			t.Errorf("%s did not use the canonical agent source", target)
		}
	}
}

func TestSharedBuildRejectsMissingConfiguredTemplate(t *testing.T) {
	root := setupBuildCommandLoafRoot(t)
	seedNativeCodexBuildFixture(t, root)
	path := filepath.Join(root, "content", "templates", "session.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build", "--target", "codex"})
	if err == nil || !strings.Contains(err.Error(), "session.md") {
		t.Fatalf("missing configured template must fail the build: %v\n%s", err, stdout.String())
	}
}
