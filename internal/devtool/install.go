package devtool

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallOptions drive explicit development-launcher activation after a
// complete build and verification.
type InstallOptions struct {
	RootDir    string
	Env        Env
	Runner     Runner
	Stdout     io.Writer
	Stderr     io.Writer
	HostTarget *Target
	// Build defaults to the full distribution Build. Tests inject a stand-in.
	Build func() error
	// Linker defaults to RefreshDevBuildLink. Tests inject a recorder.
	Linker func(launcher string) DevLinkResult
}

// Install runs the complete distribution build and verification, then
// explicitly points ~/.local/bin/loaf at this checkout through Loaf's
// user-local launcher pointer. Activation is never a side effect of Build.
func Install(options InstallOptions) error {
	root := options.RootDir
	stdout := options.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := options.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	env := options.Env
	host := options.HostTarget
	if host == nil {
		resolved, err := TargetForGo(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return fmt.Errorf("install refused: host platform %s/%s is unsupported; refusing to activate a stale host binary", runtime.GOOS, runtime.GOARCH)
		}
		host = &resolved
	}
	runner := options.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	explicit := strings.TrimSpace(env["LOAF_BUILD_TARGETS"]) != "" || strings.TrimSpace(env["LOAF_NATIVE_TARGETS"]) != ""
	var targets []Target
	if options.Build == nil || options.Runner != nil || explicit {
		pinned := env.With(map[string]string{"CGO_ENABLED": "0"}).With(PinnedToolchainEnv(root))
		resolved, err := requestedTargets(pinned, root, runner, "LOAF_BUILD_TARGETS", "LOAF_NATIVE_TARGETS")
		if err != nil {
			return err
		}
		if !containsTarget(resolved, *host) {
			return fmt.Errorf("install refused: selected targets omit the host platform %s; refusing to activate a stale host binary", host.RuntimeID)
		}
		targets = resolved
	}
	if env["LOAF_NATIVE_ARTIFACT_DRY_RUN"] == "1" {
		fmt.Fprintln(stdout, "DRY RUN: would build, verify, then activate this checkout through the development launcher")
		return nil
	}

	build := options.Build
	if build == nil {
		build = func() error {
			return Build(BuildOptions{RootDir: root, Env: env.With(map[string]string{"LOAF_BUILD_TARGETS": RuntimeIDs(targets)}), Runner: runner, Stdout: stdout, Stderr: stderr})
		}
	}
	if err := build(); err != nil {
		return err
	}

	launcher := filepath.Join(root, "bin", "loaf")
	if _, err := os.Stat(launcher); err != nil {
		return fmt.Errorf("install refused: host binary %s is missing after build", launcher)
	}
	linker := options.Linker
	if linker == nil {
		linker = func(path string) DevLinkResult {
			return RefreshDevBuildLink(path, DevLinkOptions{Env: env})
		}
	}
	result := linker(launcher)
	switch result.Status {
	case DevLinkLinked:
		fmt.Fprintf(stdout, "✓ Installed development CLI: %s -> %s\n", result.Link, launcher)
		return nil
	case DevLinkConflict:
		path := result.Link
		if result.Err != nil {
			return fmt.Errorf("install refused: %v", result.Err)
		}
		if path == "" {
			path = result.Pointer
		}
		return fmt.Errorf("install refused: %s already exists and was not replaced", path)
	case DevLinkSkipped:
		return fmt.Errorf("install refused: development launcher activation is unsupported on this platform")
	case DevLinkFailed:
		if result.Err != nil {
			return fmt.Errorf("install refused: failed to activate development launcher: %w", result.Err)
		}
		return fmt.Errorf("install refused: failed to activate development launcher")
	default:
		return fmt.Errorf("install refused: unexpected launcher status %q", result.Status)
	}
}
