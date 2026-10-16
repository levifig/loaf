package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUsageSeparatesBuildFromInstall(t *testing.T) {
	if strings.Contains(usage, "relink") || strings.Contains(strings.ToLower(usage), "auto") {
		t.Fatalf("usage still describes implicit activation:\n%s", usage)
	}
	for _, want := range []string{"build-cli", "build-go", "install"} {
		if !strings.Contains(usage, want) {
			t.Fatalf("usage missing %q:\n%s", want, usage)
		}
	}
	if strings.Contains(usage, "LOAF_DEV_LINK=0") && strings.Contains(usage, "do not touch the dev launcher pointer") {
		t.Fatal("usage still documents implicit build-time linking")
	}
}

func TestUsageSplitsNativeArtifactAndReleaseDryRun(t *testing.T) {
	if strings.Contains(usage, "LOAF_NATIVE_ARTIFACT_DRY_RUN=1, LOAF_RELEASE_DRY_RUN=1") {
		t.Fatal("usage still groups native-artifact and release dry-run as generic report-only")
	}
	if !strings.Contains(usage, "LOAF_NATIVE_ARTIFACT_DRY_RUN=1") || !strings.Contains(usage, "report-only for build-cli/build-go, install, and artifact verification") {
		t.Fatalf("usage missing native-artifact dry-run scope:\n%s", usage)
	}
	if !strings.Contains(usage, "LOAF_RELEASE_DRY_RUN=1") || !strings.Contains(usage, "report-only for release") {
		t.Fatalf("usage missing release dry-run scope:\n%s", usage)
	}
}

func TestMakefileRoutesBuildInstallAndBuildCLI(t *testing.T) {
	root := loafdevRepoRoot(t)
	defaultOut := makeDryRun(t, root)
	buildOut := makeDryRun(t, root, "build")
	if !strings.Contains(defaultOut, "cmd/loafdev build") || strings.Contains(defaultOut, "cmd/loafdev install") {
		t.Fatalf("default make = %q, want loafdev build without install", defaultOut)
	}
	if defaultOut != buildOut {
		t.Fatalf("make and make build diverged:\nmake: %q\nmake build: %q", defaultOut, buildOut)
	}
	cliOut := makeDryRun(t, root, "build-cli")
	if !strings.Contains(cliOut, "cmd/loafdev build-cli") || strings.Contains(cliOut, "cmd/loafdev install") {
		t.Fatalf("make build-cli = %q, want loafdev build-cli only", cliOut)
	}
	goOut := makeDryRun(t, root, "build-go")
	if !strings.Contains(goOut, "cmd/loafdev build-go") || strings.Contains(goOut, "cmd/loafdev install") {
		t.Fatalf("make build-go = %q, want retained loafdev build-go alias", goOut)
	}
	installOut := makeDryRun(t, root, "install")
	if !strings.Contains(installOut, "cmd/loafdev install") {
		t.Fatalf("make install = %q, want loafdev install", installOut)
	}
	if strings.Contains(installOut, "cmd/loafdev build\n") || strings.Contains(installOut, "cmd/loafdev build-go") {
		t.Fatalf("make install should not retarget the development launcher through a Make build dependency: %q", installOut)
	}
}

func TestMakefileVulnerabilityAuditUsesDevelopmentTool(t *testing.T) {
	out := makeDryRun(t, loafdevRepoRoot(t), "vulncheck")
	if !strings.Contains(out, "cmd/loafdev vulncheck") || strings.Contains(out, "govulncheck@latest") {
		t.Fatalf("make vulncheck bypasses release toolchain selection: %q", out)
	}
}

// Agent shells can hand make an open stdin pipe. loaf check reads any
// non-terminal stdin to EOF as a hook payload, so the gate must not inherit it.
func TestMakefileRenderDriftGateIgnoresInheritedStdin(t *testing.T) {
	var recipe string
	for _, line := range strings.Split(makeDryRun(t, loafdevRepoRoot(t), "verify-generated"), "\n") {
		if strings.Contains(line, "check --hook render-drift") {
			recipe = line
		}
	}
	if recipe == "" {
		t.Fatal("make verify-generated no longer runs the render-drift check")
	}

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Stands in for loaf check: reads stdin to EOF, then exits.
	if err := os.WriteFile(filepath.Join(dir, "bin", "loaf"), []byte("#!/bin/sh\ncat >/dev/null\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdin, held, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	defer held.Close()

	cmd := exec.Command("sh", "-c", recipe)
	cmd.Dir = dir
	cmd.Stdin = stdin
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("render-drift recipe %q: %v", recipe, err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("render-drift recipe %q waited for an open stdin pipe to close", recipe)
	}
}

func loafdevRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func makeDryRun(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("make", append([]string{"-n", "-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
