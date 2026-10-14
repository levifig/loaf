package devtool

import (
	"fmt"
	"os"
	"strings"
)

// Vulncheck audits source under the same explicit toolchain and CGO setting as
// BuildGo. A newer Go on the caller's PATH must not mask release vulnerabilities.
func Vulncheck(options BuildOptions) error {
	pinned := PinnedToolchainEnv(options.RootDir)
	if pinned["GOTOOLCHAIN"] == "" {
		return fmt.Errorf("vulnerability audit requires a pinned toolchain in go.mod")
	}
	env := options.Env.With(map[string]string{"CGO_ENABLED": "0"}).With(pinned)
	runner := options.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	version, err := runner.Output(options.RootDir, env, "go", "env", "GOVERSION")
	if err != nil {
		return fmt.Errorf("resolve audit toolchain: %w", err)
	}
	if strings.TrimSpace(version) != pinned["GOTOOLCHAIN"] {
		return fmt.Errorf("audit toolchain %q does not match release pin %q", strings.TrimSpace(version), pinned["GOTOOLCHAIN"])
	}
	stdout := options.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	fmt.Fprintf(stdout, "Auditing source with pinned release toolchain %s (CGO_ENABLED=0)\n", strings.TrimSpace(version))
	if err := runner.Run(options.RootDir, env, "go", "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./..."); err != nil {
		return fmt.Errorf("vulnerability audit: %w", err)
	}
	return nil
}
