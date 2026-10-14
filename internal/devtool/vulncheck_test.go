package devtool

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type auditRunner struct {
	root      string
	env       Env
	args      []string
	version   string
	lookupErr error
	auditErr  error
	runs      int
}

func (r *auditRunner) Output(root string, env Env, name string, args ...string) (string, error) {
	r.root, r.env = root, env
	if name != "go" || !reflect.DeepEqual(args, []string{"env", "GOVERSION"}) {
		return "", errors.New("unexpected toolchain lookup")
	}
	if r.version != "" {
		return r.version, r.lookupErr
	}
	return env["GOTOOLCHAIN"] + "\n", r.lookupErr
}

func (r *auditRunner) Run(root string, env Env, name string, args ...string) error {
	r.root, r.env = root, env
	r.args = append([]string{name}, args...)
	r.runs++
	return r.auditErr
}

func TestVulnerabilityAuditOverridesInheritedToolchainWithReleasePin(t *testing.T) {
	root := fixtureRoot(t)
	pin := PinnedToolchainEnv(root)["GOTOOLCHAIN"]
	parent := Env{"GOTOOLCHAIN": "go9.99.9", "CGO_ENABLED": "1", "PATH": "caller-path"}
	runner := &auditRunner{}
	var output bytes.Buffer
	if err := Vulncheck(BuildOptions{RootDir: root, Env: parent, Runner: runner, Stdout: &output}); err != nil {
		t.Fatal(err)
	}
	if runner.root != root || runner.env["GOTOOLCHAIN"] != pin || runner.env["CGO_ENABLED"] != "0" || runner.env["PATH"] != parent["PATH"] {
		t.Fatalf("audit root/env = %q/%v, want release settings in selected root", runner.root, runner.env)
	}
	if want := []string{"go", "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./..."}; !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("audit command = %q, want %q", runner.args, want)
	}
	if parent["GOTOOLCHAIN"] != "go9.99.9" || parent["CGO_ENABLED"] != "1" {
		t.Fatalf("audit changed caller settings: %v", parent)
	}
	if !strings.Contains(output.String(), pin) {
		t.Fatalf("audit did not report the verified toolchain: %q", output.String())
	}
}

func TestVulnerabilityAuditFailsClosed(t *testing.T) {
	failure := errors.New("fixture failure")
	for _, test := range []struct {
		name      string
		module    string
		runner    auditRunner
		want      string
		wantRuns  int
		wantCause bool
	}{
		{name: "missing pin", module: "module example.com/audit\n\ngo 1.25\n", want: "requires a pinned toolchain"},
		{name: "lookup failure", runner: auditRunner{lookupErr: failure}, want: "resolve audit toolchain", wantCause: true},
		{name: "different runtime", runner: auditRunner{version: "go9.99.9\n"}, want: "does not match release pin"},
		{name: "audit failure", runner: auditRunner{auditErr: failure}, want: "vulnerability audit", wantRuns: 1, wantCause: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := fixtureRoot(t)
			if test.module != "" {
				if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(test.module), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := Vulncheck(BuildOptions{RootDir: root, Env: Env{}, Runner: &test.runner, Stdout: &bytes.Buffer{}})
			if err == nil || !strings.Contains(err.Error(), test.want) || test.runner.runs != test.wantRuns {
				t.Fatalf("err/runs = %v/%d, want %q/%d", err, test.runner.runs, test.want, test.wantRuns)
			}
			if test.wantCause && !errors.Is(err, failure) {
				t.Fatalf("audit lost failure cause: %v", err)
			}
		})
	}
}
