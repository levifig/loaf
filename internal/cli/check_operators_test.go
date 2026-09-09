package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerCheckOperatorHelpListsStandaloneValidators(t *testing.T) {
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: t.TempDir()}).Run([]string{"check", "--help"}); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	for _, want := range []string{"commit-msg", "secrets", "changelog", "compliance", "test-naming", "dockerfile", "k8s-manifest", "bounds", "units", "standard-refs", "Usage: loaf check --hook <id> [--advisory] [--json]"} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want %q", output, want)
		}
	}
}

func TestRunnerCheckUnknownOperatorIsRejected(t *testing.T) {
	err := (Runner{WorkingDir: t.TempDir()}).Run([]string{"check", "python-style"})
	if err == nil || !strings.Contains(err.Error(), "unknown check subcommand") {
		t.Fatalf("python-style: %v", err)
	}
}

func TestRunnerCheckCommitMsgValidatesFileAndStdinWithoutInterpreter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeFile(t, filepath.Join(root, "ok.txt"), "feat: add native commit-msg check\n")
	writeFile(t, filepath.Join(root, "bad.txt"), "not conventional\n")

	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "commit-msg", "ok.txt"}); err != nil {
		t.Fatalf("valid file: %v\n%s", err, stdout.String())
	}

	stdout.Reset()
	err := (Runner{Stdout: &stdout, Stdin: bytes.NewBufferString("feat: add stdin path\n"), WorkingDir: root}).Run([]string{"check", "commit-msg", "-"})
	if err != nil {
		t.Fatalf("valid stdin: %v\n%s", err, stdout.String())
	}

	var stderr bytes.Buffer
	err = (Runner{Stderr: &stderr, WorkingDir: root}).Run([]string{"check", "commit-msg", "bad.txt"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 || !strings.Contains(stderr.String(), "Conventional Commits") {
		t.Fatalf("invalid file: err=%v stderr=%q", err, stderr.String())
	}
}

func TestRunnerCheckCommitMsgJSONAndSharedEngine(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeFile(t, filepath.Join(root, "scoped.txt"), "feat(core): add scoped commit\n")
	writeFile(t, filepath.Join(root, "ai.txt"), "feat: add guard\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")
	long := "feat: " + strings.Repeat("x", 80)
	writeFile(t, filepath.Join(root, "long.txt"), long+"\n")

	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "commit-msg", "scoped.txt", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("scoped: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Check != "commit-msg" || output.Passed || output.ExitCode != 1 || !strings.Contains(strings.Join(output.Errors, "\n"), "Conventional Commits") {
		t.Fatalf("output = %#v", output)
	}

	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "commit-msg", "ai.txt", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("ai: %v", err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "AI attribution") {
		t.Fatalf("ai output = %#v", output)
	}

	stdout.Reset()
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "commit-msg", "long.txt"}); err != nil {
		t.Fatalf("long subject should warn, not fail: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "recommended: <=72") {
		t.Fatalf("long subject output = %q", stdout.String())
	}
}

func TestRunnerCheckCommitMsgRequiresPath(t *testing.T) {
	err := (Runner{WorkingDir: t.TempDir()}).Run([]string{"check", "commit-msg"})
	if err == nil || !strings.Contains(err.Error(), "file path") {
		t.Fatalf("missing path: %v", err)
	}
}

func TestRunnerCheckSecretsTreeScanIsNotHookPayload(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mkdirAll(t, filepath.Join(root, "src"))
	writeFile(t, filepath.Join(root, "src", "ok.go"), "package src\n")
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "secrets", "src"}); err != nil {
		t.Fatalf("clean tree: %v\n%s", err, stdout.String())
	}

	// Construct assignment fixtures so the test source itself is not a hit.
	writeFile(t, filepath.Join(root, "src", "config.go"), "package src\n"+secretTreeAssignment("password", "hunter2")+"\n")
	writeFile(t, filepath.Join(root, "notes.md"), secretTreeAssignment("password", "hunter2")+"\n")
	writeFile(t, filepath.Join(root, ".env"), "NAME=dev\n")
	writeFile(t, filepath.Join(root, "id_rsa"), "not a real key\n")
	mkdirAll(t, filepath.Join(root, "node_modules"))
	writeFile(t, filepath.Join(root, "node_modules", "secret.go"), secretTreeAssignment("password", "hunter2")+"\n")

	stdout.Reset()
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "secrets", ".", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("dirty tree: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(output.Findings, "\n")
	if !strings.Contains(joined, "password assignment in src/config.go") || !strings.Contains(joined, ".env file") || !strings.Contains(joined, "private key file: id_rsa") {
		t.Fatalf("findings = %#v", output.Findings)
	}
	if strings.Contains(joined, "notes.md") || strings.Contains(joined, "node_modules") {
		t.Fatalf("excluded path leaked: %#v", output.Findings)
	}
}

func TestRunnerCheckChangelogValidatesMicroSchema(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeFile(t, filepath.Join(root, "ok.md"), "# Doc\n\n**Last Updated**: 2026-09-05\n\n## Changelog\n\n- 2026-09-05 - Added native check\n- 2026-08-01 - First draft\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "changelog", "ok.md"}); err != nil {
		t.Fatalf("valid: %v", err)
	}

	writeFile(t, filepath.Join(root, "missing.md"), "# Doc\n\nNo section.\n")
	err := (Runner{WorkingDir: root}).Run([]string{"check", "changelog", "missing.md"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("missing section: %v", err)
	}

	writeFile(t, filepath.Join(root, "order.md"), "# Doc\n\n## Changelog\n\n- 2026-08-01 - Older first\n- 2026-09-05 - Newer second\n")
	var stdout bytes.Buffer
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "changelog", "order.md", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("order: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "reverse chronological") {
		t.Fatalf("order output = %#v", output)
	}

	writeFile(t, filepath.Join(root, "stale.md"), "# Doc\n\n**Last Updated**: 2026-01-01\n\n## Changelog\n\n- 2026-09-05 - Added native check\n")
	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "changelog", "stale.md", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("stale header: %v", err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "Last Updated") {
		t.Fatalf("stale output = %#v", output)
	}
}

func TestRunnerCheckComplianceRequiresCheckedBoxes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeFile(t, filepath.Join(root, "done.md"), "- [x] Threat model\n- [X] Secrets scan\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "compliance", "done.md"}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	writeFile(t, filepath.Join(root, "empty.md"), "# Notes\n\nNo boxes.\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "compliance", "empty.md"}); err != nil {
		t.Fatalf("no boxes: %v", err)
	}
	writeFile(t, filepath.Join(root, "open.md"), "- [x] Threat model\n- [ ] Secrets scan\n")
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "compliance", "open.md", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("incomplete: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Check != "compliance" || len(output.Findings) != 1 || output.Findings[0] != "Secrets scan" {
		t.Fatalf("output = %#v", output)
	}
}

func TestRunnerCheckTestNamingIsARealGate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mkdirAll(t, filepath.Join(root, "tests"))
	mkdirAll(t, filepath.Join(root, "fixtures"))
	writeFile(t, filepath.Join(root, "tests", "test_valid.py"), "def test_ok():\n    pass\n\ndef _helper():\n    pass\n")
	writeFile(t, filepath.Join(root, "fixtures", "payload_perfect.json"), "{}\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "test-naming"}); err != nil {
		t.Fatalf("valid: %v", err)
	}

	writeFile(t, filepath.Join(root, "tests", "helpers.py"), "def helper():\n    pass\n")
	writeFile(t, filepath.Join(root, "tests", "test_bad.py"), "def helper():\n    pass\n")
	writeFile(t, filepath.Join(root, "fixtures", "payload.json"), "{}\n")
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", ".", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("invalid should fail (old script always exited 0): %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(output.Errors, "\n")
	if !strings.Contains(joined, "helpers.py") || !strings.Contains(joined, "doesn't start with 'test_'") || !strings.Contains(joined, "payload.json") {
		t.Fatalf("errors = %#v", output.Errors)
	}
}

func TestRunnerCheckTestNamingFailsWhenTargetCannotBeTraversed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "missing", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("missing dir: %v stdout=%s", err, stdout.String())
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Passed || output.ExitCode != 1 {
		t.Fatalf("output = %#v", output)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "unreadable path") {
		t.Fatalf("errors = %#v", output.Errors)
	}
}

func TestRunnerCheckTestNamingKeepsTargetDirectoryContext(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mkdirAll(t, filepath.Join(root, "tests"))
	mkdirAll(t, filepath.Join(root, "fixtures"))
	writeFile(t, filepath.Join(root, "tests", "helpers.py"), "def helper():\n    pass\n")
	writeFile(t, filepath.Join(root, "fixtures", "payload.json"), "{}\n")

	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "tests", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("tests dir: %v stdout=%s", err, stdout.String())
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "helpers.py") {
		t.Fatalf("tests errors = %#v", output.Errors)
	}

	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "fixtures", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("fixtures dir: %v stdout=%s", err, stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "payload.json") {
		t.Fatalf("fixtures errors = %#v", output.Errors)
	}
}

func TestRunnerCheckTestNamingKeepsNestedTargetDirectoryContext(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mkdirAll(t, filepath.Join(root, "tests", "unit"))
	mkdirAll(t, filepath.Join(root, "fixtures", "api"))
	writeFile(t, filepath.Join(root, "tests", "unit", "helpers.py"), "def helper():\n    pass\n")
	writeFile(t, filepath.Join(root, "fixtures", "api", "payload.json"), "{}\n")

	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "tests", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("tests parent: %v stdout=%s", err, stdout.String())
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "helpers.py") {
		t.Fatalf("tests parent errors = %#v", output.Errors)
	}

	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "tests/unit", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("tests/unit: %v stdout=%s", err, stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "helpers.py") {
		t.Fatalf("tests/unit errors = %#v", output.Errors)
	}

	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "fixtures", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("fixtures parent: %v stdout=%s", err, stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "payload.json") {
		t.Fatalf("fixtures parent errors = %#v", output.Errors)
	}

	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "fixtures/api", "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("fixtures/api: %v stdout=%s", err, stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "payload.json") {
		t.Fatalf("fixtures/api errors = %#v", output.Errors)
	}
}

func TestRunnerCheckTestNamingContextIsIndependentOfCwd(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	testsUnit := filepath.Join(root, "tests", "unit")
	fixturesAPI := filepath.Join(root, "fixtures", "api")
	mkdirAll(t, testsUnit)
	mkdirAll(t, fixturesAPI)
	writeFile(t, filepath.Join(testsUnit, "helpers.py"), "def helper():\n    pass\n")
	writeFile(t, filepath.Join(fixturesAPI, "payload.json"), "{}\n")

	families := []struct {
		name      string
		relTarget string
		absTarget string
		needle    string
	}{
		{name: "tests/unit helpers.py", relTarget: filepath.Join("tests", "unit"), absTarget: testsUnit, needle: "helpers.py"},
		{name: "fixtures/api payload.json", relTarget: filepath.Join("fixtures", "api"), absTarget: fixturesAPI, needle: "payload.json"},
	}
	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			cwds := []struct {
				name string
				dir  string
			}{
				{name: "parent cwd", dir: root},
				{name: "target cwd", dir: family.absTarget},
			}
			for _, cwd := range cwds {
				relArg := family.relTarget
				if cwd.dir == family.absTarget {
					relArg = "."
				}
				argForms := []struct {
					name string
					args []string
				}{
					{name: "omitted target", args: []string{"check", "test-naming", "--json"}},
					{name: "relative target", args: []string{"check", "test-naming", relArg, "--json"}},
					{name: "absolute target", args: []string{"check", "test-naming", family.absTarget, "--json"}},
				}
				for _, form := range argForms {
					t.Run(cwd.name+" "+form.name, func(t *testing.T) {
						var stdout bytes.Buffer
						err := (Runner{Stdout: &stdout, WorkingDir: cwd.dir}).Run(form.args)
						var exitErr ExitError
						if !errors.As(err, &exitErr) || exitErr.Code != 1 {
							t.Fatalf("err=%v stdout=%s", err, stdout.String())
						}
						var output checkOperatorJSON
						if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
							t.Fatal(err)
						}
						if output.Passed {
							t.Fatalf("passed=true; errors=%#v", output.Errors)
						}
						if !strings.Contains(strings.Join(output.Errors, "\n"), family.needle) {
							t.Fatalf("errors=%#v, want %q", output.Errors, family.needle)
						}
					})
				}
			}
		})
	}
}

func TestTestNamingRulePathUsesTargetAncestorsNotCwd(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		root string
		rel  string
		want string
	}{
		{
			name: "nested tests target keeps tests ancestor",
			root: "/proj/tests/unit",
			rel:  "helpers.py",
			want: filepath.Join("tests", "unit", "helpers.py"),
		},
		{
			name: "nested fixtures target keeps fixtures ancestor",
			root: "/proj/fixtures/api",
			rel:  "payload.json",
			want: filepath.Join("fixtures", "api", "payload.json"),
		},
		{
			name: "direct tests target keeps tests ancestor",
			root: "/proj/tests",
			rel:  "helpers.py",
			want: filepath.Join("tests", "helpers.py"),
		},
		{
			name: "project root relies on walk-relative path",
			root: "/proj",
			rel:  filepath.Join("tests", "unit", "helpers.py"),
			want: filepath.Join("tests", "unit", "helpers.py"),
		},
		{
			name: "out-of-tree nested target keeps relevant suffix",
			root: "/other/fixtures/api",
			rel:  "payload.json",
			want: filepath.Join("fixtures", "api", "payload.json"),
		},
		{
			name: "no relevant ancestor leaves walk-relative path",
			root: "/proj/src",
			rel:  "helpers.py",
			want: "helpers.py",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := testNamingRulePath(tc.root, tc.rel)
			if got != tc.want {
				t.Fatalf("testNamingRulePath(%q, %q) = %q, want %q", tc.root, tc.rel, got, tc.want)
			}
		})
	}
}

func TestRunnerCheckTestNamingReadsLinesWithinProjectLimit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mkdirAll(t, filepath.Join(root, "tests"))
	body := "#" + strings.Repeat("x", 70*1024) + "\ndef helper():\n    pass\n"
	writeFile(t, filepath.Join(root, "tests", "test_long.py"), body)

	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "test-naming", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("long line: %v stdout=%s", err, stdout.String())
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Errors, "\n"), "doesn't start with 'test_'") {
		t.Fatalf("errors = %#v", output.Errors)
	}
}

func secretTreeAssignment(key, value string) string {
	return key + " = \"" + value + "\""
}

func TestCheckOperatorArgsRejectUnknownFlags(t *testing.T) {
	if _, err := parseCheckOperatorArgs([]string{"--unknown"}, false); err == nil {
		t.Fatal("accepted unknown flag")
	}
	options, err := parseCheckOperatorArgs([]string{"--json", "--", "-file.md"}, true)
	if err != nil || !options.jsonOutput || options.path != "-file.md" {
		t.Fatalf("options=%#v err=%v", options, err)
	}
}
