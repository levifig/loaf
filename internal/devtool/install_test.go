package devtool

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallActivatesOnlyAfterSuccessfulVerification(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	launcher := filepath.Join(root, "bin", "loaf")
	var steps []string
	var stdout bytes.Buffer
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share")},
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			steps = append(steps, "build")
			writeFixture(t, launcher, "#!/bin/sh\necho installed-ok\n")
			return nil
		},
		Linker: func(path string) DevLinkResult {
			steps = append(steps, "link")
			if path != launcher {
				t.Fatalf("linker path = %q, want %s", path, launcher)
			}
			return DevLinkResult{Status: DevLinkLinked, Link: filepath.Join(home, ".local", "bin", "loaf")}
		},
	})
	if err != nil {
		t.Fatalf("Install error = %v", err)
	}
	if strings.Join(steps, ",") != "build,link" {
		t.Fatalf("steps = %q, want build then link", steps)
	}
}

func TestInstallDoesNotActivateWhenBuildFails(t *testing.T) {
	root := fixtureRoot(t)
	home := t.TempDir()
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Build:   func() error { return errors.New("compile failed") },
		Linker: func(string) DevLinkResult {
			t.Fatal("activation ran after a failed build")
			return DevLinkResult{Status: DevLinkLinked}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "compile failed") {
		t.Fatalf("Install error = %v, want the build failure", err)
	}
	assertNoPublicDevLink(t, home)
}

func TestInstallCreatesRelativeLinksAndExecutableCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	launcher := filepath.Join(root, "bin", "loaf")
	public := filepath.Join(home, ".local", "bin", "loaf")
	pointer := filepath.Join(home, ".local", "share", "loaf", "current-dev-launcher")
	var stdout bytes.Buffer
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share")},
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			writeFixture(t, launcher, "#!/bin/sh\necho installed-ok\n")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Install error = %v", err)
	}
	if target, err := os.Readlink(public); err != nil || filepath.IsAbs(target) {
		t.Fatalf("public link should be relative, got %q err=%v", target, err)
	}
	if resolved := resolvedSymlink(t, public); resolved != pointer {
		t.Fatalf("public link resolves to %q, want %s", resolved, pointer)
	}
	if target, err := os.Readlink(pointer); err != nil || filepath.IsAbs(target) {
		t.Fatalf("pointer should be relative, got %q err=%v", target, err)
	}
	if resolved := resolvedSymlink(t, pointer); resolved != launcher {
		t.Fatalf("pointer resolves to %q, want %s", resolved, launcher)
	}
	out, execErr := exec.Command(public).CombinedOutput()
	if execErr != nil || strings.TrimSpace(string(out)) != "installed-ok" {
		t.Fatalf("installed command output = %q err=%v", out, execErr)
	}
	if !strings.Contains(stdout.String(), public) {
		t.Fatalf("stdout = %q, want the installed PATH name", stdout.String())
	}
}

func TestInstallRelativeLinksSurviveAsymmetricPhysicalPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	base := t.TempDir()
	physicalPrefix := filepath.Join(base, "private", "var", "folders")
	aliasPrefix := filepath.Join(base, "var")
	if err := os.MkdirAll(physicalPrefix, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(physicalPrefix, aliasPrefix); err != nil {
		t.Fatal(err)
	}
	physicalHome := filepath.Join(physicalPrefix, "home")
	physicalData := filepath.Join(physicalPrefix, "data")
	physicalCheckout := filepath.Join(base, "src", "loaf")
	home := filepath.Join(aliasPrefix, "home")
	dataHome := filepath.Join(aliasPrefix, "data")
	root := filepath.Join(base, "checkout")
	for _, dir := range []string{physicalHome, physicalData, physicalCheckout} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(physicalCheckout, root); err != nil {
		t.Fatal(err)
	}

	launcher := filepath.Join(root, "bin", "loaf")
	public := filepath.Join(home, ".local", "bin", "loaf")
	pointer := filepath.Join(dataHome, "loaf", "current-dev-launcher")
	install := func() error {
		return Install(InstallOptions{
			RootDir: root,
			Env:     Env{"HOME": home, "XDG_DATA_HOME": dataHome},
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			Build: func() error {
				writeFixture(t, launcher, "#!/bin/sh\necho installed-ok\n")
				return nil
			},
		})
	}
	if err := install(); err != nil {
		t.Fatalf("Install error = %v", err)
	}
	if target, err := os.Readlink(public); err != nil || filepath.IsAbs(target) {
		t.Fatalf("public link should be relative, got %q err=%v", target, err)
	}
	if filepath.Base(readLink(t, public)) != "current-dev-launcher" {
		t.Fatalf("public link must target the pointer, got %q", readLink(t, public))
	}
	if target, err := os.Readlink(pointer); err != nil || filepath.IsAbs(target) {
		t.Fatalf("pointer should be relative, got %q err=%v", target, err)
	}
	out, execErr := exec.Command(public).CombinedOutput()
	if execErr != nil || strings.TrimSpace(string(out)) != "installed-ok" {
		t.Fatalf("installed command output = %q err=%v", out, execErr)
	}
	if err := install(); err != nil {
		t.Fatalf("repeat Install error = %v", err)
	}
	out, execErr = exec.Command(public).CombinedOutput()
	if execErr != nil || strings.TrimSpace(string(out)) != "installed-ok" {
		t.Fatalf("reinstalled command output = %q err=%v", out, execErr)
	}

	physicalPointer := filepath.Join(physicalData, "loaf", "current-dev-launcher")
	if err := os.Remove(public); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(physicalPointer, public); err != nil {
		t.Fatal(err)
	}
	if err := install(); err != nil {
		t.Fatalf("Install over absolute pointer link error = %v", err)
	}
	if readLink(t, public) != physicalPointer {
		t.Fatalf("owned absolute pointer link was replaced: %q", readLink(t, public))
	}

	if err := os.Remove(public); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(launcher, public); err != nil {
		t.Fatal(err)
	}
	if err := install(); err == nil {
		t.Fatal("Install succeeded over a symlink that points at the binary")
	}
	if got := readLink(t, public); got != launcher {
		t.Fatalf("foreign binary symlink mutated: %q", got)
	}
}

func readLink(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestInstallPreservesForeignPATHEntriesAndReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	public := filepath.Join(home, ".local", "bin", "loaf")
	writeFixture(t, public, "homebrew-owned\n")
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share")},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			writeFixture(t, filepath.Join(root, "bin", "loaf"), "#!/bin/sh\necho leftover\n")
			return nil
		},
	})
	if err == nil {
		t.Fatal("Install succeeded over a foreign PATH entry")
	}
	if got := readFixture(t, public); got != "homebrew-owned\n" {
		t.Fatalf("foreign PATH entry mutated: %q", got)
	}
}

func TestInstallDoesNotActivateOnDryRunOrUnsupportedOrOmittedHost(t *testing.T) {
	root := fixtureRoot(t)
	home := t.TempDir()
	host := Target{"linux-x64", "linux", "amd64"}
	writeFixture(t, filepath.Join(root, "bin", "loaf"), "stale-host\n")

	dryErr := Install(InstallOptions{
		RootDir:    root,
		Env:        Env{"HOME": home, "LOAF_NATIVE_ARTIFACT_DRY_RUN": "1"},
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		HostTarget: &host,
		Build:      func() error { t.Fatal("dry-run install must not build"); return nil },
		Linker: func(string) DevLinkResult {
			t.Fatal("dry-run install activated")
			return DevLinkResult{Status: DevLinkLinked}
		},
	})
	if dryErr != nil {
		t.Fatalf("dry-run Install error = %v, want success without activation", dryErr)
	}
	assertNoPublicDevLink(t, home)

	skipErr := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			writeFixture(t, filepath.Join(root, "bin", "loaf"), "#!/bin/sh\necho skip\n")
			return nil
		},
		Linker: func(string) DevLinkResult { return DevLinkResult{Status: DevLinkSkipped} },
	})
	if skipErr == nil {
		t.Fatal("unsupported install succeeded")
	}
	assertNoPublicDevLink(t, home)

	omitErr := Install(InstallOptions{
		RootDir:    root,
		Env:        Env{"HOME": home, "LOAF_BUILD_TARGETS": "win32-x64"},
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		HostTarget: &host,
		Build:      func() error { return nil },
		Linker: func(string) DevLinkResult {
			t.Fatal("omitted-host install activated a stale binary")
			return DevLinkResult{Status: DevLinkLinked}
		},
	})
	if omitErr == nil || !strings.Contains(omitErr.Error(), "host") {
		t.Fatalf("omitted-host error = %v, want a host omission failure", omitErr)
	}
	assertNoPublicDevLink(t, home)
}

func TestInstallIgnoresReleaseDryRunAndHonorsNativeArtifactDryRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	launcher := filepath.Join(root, "bin", "loaf")
	var built bool
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share"), "LOAF_RELEASE_DRY_RUN": "1"},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			built = true
			writeFixture(t, launcher, "#!/bin/sh\necho installed-ok\n")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Install with LOAF_RELEASE_DRY_RUN error = %v, want activation", err)
	}
	if !built {
		t.Fatal("LOAF_RELEASE_DRY_RUN suppressed install")
	}
	public := filepath.Join(home, ".local", "bin", "loaf")
	if _, err := os.Lstat(public); err != nil {
		t.Fatalf("release dry-run suppressed the public link: %v", err)
	}

	home2 := t.TempDir()
	var nativeStdout bytes.Buffer
	nativeErr := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home2, "LOAF_NATIVE_ARTIFACT_DRY_RUN": "1"},
		Stdout:  &nativeStdout,
		Stderr:  &bytes.Buffer{},
		Build:   func() error { t.Fatal("native-artifact dry-run install must not build"); return nil },
		Linker: func(string) DevLinkResult {
			t.Fatal("native-artifact dry-run install activated")
			return DevLinkResult{Status: DevLinkLinked}
		},
	})
	if nativeErr != nil {
		t.Fatalf("native-artifact dry-run Install error = %v", nativeErr)
	}
	assertNoPublicDevLink(t, home2)
}

func TestInstallRecreatesMissingDataHomeAndRelinksOwnedDanglingPublic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	dataHome := filepath.Join(home, ".local", "share")
	launcher := filepath.Join(root, "bin", "loaf")
	public := filepath.Join(home, ".local", "bin", "loaf")
	pointer := filepath.Join(dataHome, "loaf", "current-dev-launcher")
	install := func() error {
		return Install(InstallOptions{
			RootDir: root,
			Env:     Env{"HOME": home, "XDG_DATA_HOME": dataHome},
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			Build: func() error {
				writeFixture(t, launcher, "#!/bin/sh\necho installed-ok\n")
				return nil
			},
		})
	}
	if err := install(); err != nil {
		t.Fatalf("first Install error = %v", err)
	}
	if err := os.RemoveAll(filepath.Join(dataHome, "loaf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(public); err != nil {
		t.Fatalf("public link missing after data-home deletion: %v", err)
	}
	if err := install(); err != nil {
		t.Fatalf("reinstall after deleting data home error = %v", err)
	}
	if resolved := resolvedSymlink(t, public); resolved != pointer {
		t.Fatalf("public link resolves to %q, want %s", resolved, pointer)
	}
	out, execErr := exec.Command(public).CombinedOutput()
	if execErr != nil || strings.TrimSpace(string(out)) != "installed-ok" {
		t.Fatalf("reinstalled command output = %q err=%v", out, execErr)
	}

	foreignHome := t.TempDir()
	foreignPublic := filepath.Join(foreignHome, ".local", "bin", "loaf")
	writeFixture(t, foreignPublic, "homebrew-owned\n")
	foreignData := filepath.Join(foreignHome, ".local", "share")
	if err := os.RemoveAll(filepath.Join(foreignData, "loaf")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": foreignHome, "XDG_DATA_HOME": foreignData},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			writeFixture(t, launcher, "#!/bin/sh\necho leftover\n")
			return nil
		},
	})
	if err == nil {
		t.Fatal("Install succeeded over a foreign PATH entry after missing data home")
	}
	if got := readFixture(t, foreignPublic); got != "homebrew-owned\n" {
		t.Fatalf("foreign PATH entry mutated: %q", got)
	}
}

func TestInstallReportsPointerPathOnRegularFileConflict(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	pointer := filepath.Join(home, ".local", "share", "loaf", "current-dev-launcher")
	writeFixture(t, pointer, "operator-owned-pointer\n")
	err := Install(InstallOptions{
		RootDir: root,
		Env:     Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share")},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Build: func() error {
			writeFixture(t, filepath.Join(root, "bin", "loaf"), "#!/bin/sh\necho leftover\n")
			return nil
		},
	})
	if err == nil {
		t.Fatal("Install succeeded over a regular-file pointer")
	}
	if !strings.Contains(err.Error(), pointer) {
		t.Fatalf("Install error = %v, want pointer path %s", err, pointer)
	}
	if got := readFixture(t, pointer); got != "operator-owned-pointer\n" {
		t.Fatalf("foreign pointer mutated: %q", got)
	}
}

func TestInstallDefaultBuildUsesRunnerThenVerifiesBeforeLinking(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	host, err := TargetForGo(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(root, "bin", "loaf")
	var linked bool
	runner := &stagingRecordingRunner{t: t, root: root, host: host}
	err = Install(InstallOptions{
		RootDir:    root,
		Env:        Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share"), "LOAF_BUILD_TARGETS": host.RuntimeID, "LOAF_VERIFY_TARGETS": host.RuntimeID},
		Runner:     runner,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		HostTarget: &host,
		Linker: func(path string) DevLinkResult {
			linked = true
			if path != launcher {
				t.Fatalf("linker path = %q, want %s", path, launcher)
			}
			return DevLinkResult{Status: DevLinkLinked, Link: filepath.Join(home, ".local", "bin", "loaf")}
		},
	})
	if err != nil {
		t.Fatalf("default-build Install error = %v", err)
	}
	if !linked {
		t.Fatal("successful verification did not link")
	}
	joined := strings.Join(runner.calls, "\n")
	if !strings.Contains(joined, "go build ") {
		t.Fatalf("runner calls = %#v, want go build staging", runner.calls)
	}
	if !strings.Contains(joined, filepath.Join(root, "bin", "loaf")+" __generate-cli-ref") {
		t.Fatalf("runner calls = %#v, want CLI reference generation", runner.calls)
	}
	if !strings.Contains(joined, filepath.Join(root, "bin", "loaf")+" build") {
		t.Fatalf("runner calls = %#v, want content build", runner.calls)
	}
	buildIdx, refIdx, contentIdx := -1, -1, -1
	for i, call := range runner.calls {
		if buildIdx < 0 && strings.HasPrefix(call, "go build ") {
			buildIdx = i
		}
		if call == filepath.Join(root, "bin", "loaf")+" __generate-cli-ref" {
			refIdx = i
		}
		if call == filepath.Join(root, "bin", "loaf")+" build" {
			contentIdx = i
		}
	}
	if !(buildIdx < refIdx && refIdx < contentIdx) {
		t.Fatalf("pipeline order = %#v, want compile then CLI ref then content", runner.calls)
	}
	if _, err := os.Stat(launcher); err != nil {
		t.Fatalf("host binary missing after default build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "plugins", "loaf", "hooks", "hooks.json")); err != nil {
		t.Fatalf("content artifact missing after default build: %v", err)
	}

	failRoot := fixtureRoot(t)
	failHome := t.TempDir()
	failRunner := &stagingRecordingRunner{t: t, root: failRoot, host: host, failVerify: true}
	err = Install(InstallOptions{
		RootDir:    failRoot,
		Env:        Env{"HOME": failHome, "XDG_DATA_HOME": filepath.Join(failHome, ".local", "share"), "LOAF_BUILD_TARGETS": host.RuntimeID, "LOAF_VERIFY_TARGETS": host.RuntimeID},
		Runner:     failRunner,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		HostTarget: &host,
		Linker: func(string) DevLinkResult {
			t.Fatal("verification failure linked")
			return DevLinkResult{Status: DevLinkLinked}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "must not exist") {
		t.Fatalf("verification-failure Install error = %v, want VerifyArtifacts refusal", err)
	}
	assertNoPublicDevLink(t, failHome)
}

func TestInstallRefusesCrossArchGoEnvFallbackWithoutTouchingStaleHostBinary(t *testing.T) {
	root := fixtureRoot(t)
	home := t.TempDir()
	host := Target{"linux-x64", "linux", "amd64"}
	cross, err := TargetForGo("linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if cross.RuntimeID == host.RuntimeID {
		t.Fatal("need a distinct cross architecture")
	}
	launcher := filepath.Join(root, "bin", "loaf")
	writeFixture(t, launcher, "stale-host\n")
	before := readFixture(t, launcher)
	runner := &crossArchGoEnvRunner{t: t, goos: cross.GOOS, goarch: cross.GOARCH}
	err = Install(InstallOptions{
		RootDir:    root,
		Env:        Env{"HOME": home, "GOOS": cross.GOOS, "GOARCH": cross.GOARCH},
		Runner:     runner,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		HostTarget: &host,
		Linker: func(string) DevLinkResult {
			t.Fatal("cross-arch install activated a stale binary")
			return DevLinkResult{Status: DevLinkLinked}
		},
	})
	if err == nil || !strings.Contains(err.Error(), host.RuntimeID) {
		t.Fatalf("cross-arch Install error = %v, want host omission of %s", err, host.RuntimeID)
	}
	if got := readFixture(t, launcher); got != before {
		t.Fatalf("stale bin/loaf mutated: %q, want %q", got, before)
	}
	if !strings.Contains(strings.Join(runner.calls, "\n"), "go env GOOS GOARCH") {
		t.Fatalf("runner calls = %#v, want go env fallback", runner.calls)
	}
	for _, call := range runner.calls {
		if strings.HasPrefix(call, "go build ") {
			t.Fatalf("runner calls = %#v, want no go build", runner.calls)
		}
	}
	assertNoPublicDevLink(t, home)
}

func TestInstallDefaultBuildResolvesHostFromGoEnvWhenTargetListIsOmitted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dev links are skipped on Windows")
	}
	root := fixtureRoot(t)
	home := t.TempDir()
	host, err := TargetForGo(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(root, "bin", "loaf")
	var linked bool
	runner := &stagingRecordingRunner{t: t, root: root, host: host}
	err = Install(InstallOptions{
		RootDir:    root,
		Env:        Env{"HOME": home, "XDG_DATA_HOME": filepath.Join(home, ".local", "share")},
		Runner:     runner,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		HostTarget: &host,
		Linker: func(path string) DevLinkResult {
			linked = true
			if path != launcher {
				t.Fatalf("linker path = %q, want %s", path, launcher)
			}
			return DevLinkResult{Status: DevLinkLinked, Link: filepath.Join(home, ".local", "bin", "loaf")}
		},
	})
	if err != nil {
		t.Fatalf("implicit-host Install error = %v", err)
	}
	if !linked {
		t.Fatal("implicit-host install did not link")
	}
	joined := strings.Join(runner.calls, "\n")
	if !strings.Contains(joined, "go env GOOS GOARCH") {
		t.Fatalf("runner calls = %#v, want go env fallback", runner.calls)
	}
	if !strings.Contains(joined, "go build ") {
		t.Fatalf("runner calls = %#v, want go build staging", runner.calls)
	}
	if got := readFixture(t, launcher); got != "compiled-"+host.RuntimeID+"\n" {
		t.Fatalf("bin/loaf = %q, want the rebuilt host binary", got)
	}
}

type stagingRecordingRunner struct {
	t          *testing.T
	root       string
	host       Target
	failVerify bool
	calls      []string
}

func (r *stagingRecordingRunner) Run(dir string, env Env, name string, args ...string) error {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == "go" && len(args) > 0 && args[0] == "build" {
		dest := ""
		for i, arg := range args {
			if arg == "-o" && i+1 < len(args) {
				dest = args[i+1]
				break
			}
		}
		if dest == "" {
			r.t.Fatalf("go build missing -o: %v", args)
		}
		writeFixture(r.t, dest, "compiled-"+r.host.RuntimeID+"\n")
		return nil
	}
	loaf := filepath.Join(r.root, "bin", "loaf")
	if name == loaf && len(args) == 1 && args[0] == "__generate-cli-ref" {
		writeFixture(r.t, filepath.Join(r.root, "content", "skills", "loaf-reference", "SKILL.md"), "# CLI\n")
		return nil
	}
	if name == loaf && len(args) == 1 && args[0] == "build" {
		writeFixture(r.t, filepath.Join(r.root, "plugins", "loaf", "hooks", "hooks.json"), "{}\n")
		if r.failVerify {
			writeFixture(r.t, filepath.Join(r.root, "plugins", "loaf", "bin", "loaf"), "forbidden\n")
		}
		return nil
	}
	r.t.Fatalf("unexpected Run %s %v", name, args)
	return nil
}

func (r *stagingRecordingRunner) Output(dir string, env Env, name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == "go" && len(args) == 3 && args[0] == "env" {
		return r.host.GOOS + "\n" + r.host.GOARCH + "\n", nil
	}
	return "", nil
}

type crossArchGoEnvRunner struct {
	t            *testing.T
	goos, goarch string
	calls        []string
}

func (r *crossArchGoEnvRunner) Run(dir string, env Env, name string, args ...string) error {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	r.t.Fatalf("unexpected Run %s %v", name, args)
	return nil
}

func (r *crossArchGoEnvRunner) Output(dir string, env Env, name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == "go" && len(args) == 3 && args[0] == "env" {
		return r.goos + "\n" + r.goarch + "\n", nil
	}
	r.t.Fatalf("unexpected Output %s %v", name, args)
	return "", nil
}

func assertNoPublicDevLink(t *testing.T, home string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "loaf")); !os.IsNotExist(err) {
		t.Fatalf("public PATH entry exists: %v", err)
	}
}
