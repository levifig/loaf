package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type failReader struct {
	err error
}

func (r failReader) Read([]byte) (int, error) {
	return 0, r.err
}

type terminalErrorReader struct {
	*strings.Reader
}

func (r terminalErrorReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if r.Len() == 0 {
		return n, errors.New("pipe closed after data")
	}
	return n, err
}

func TestRunnerCheckInfraSafetyBlocksAndWarns(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	cases := []struct {
		name    string
		command string
		block   bool
		warning string
		needle  string
	}{
		{name: "delete namespace", command: "kubectl delete namespace prod", block: true, needle: "kubectl delete namespace"},
		{name: "delete ns", command: "kubectl delete ns staging", block: true, needle: "kubectl delete namespace"},
		{name: "delete all", command: "kubectl delete pods --all", block: true, needle: "kubectl delete --all"},
		{name: "terraform auto-approve", command: "terraform destroy -auto-approve", block: true, needle: "terraform destroy -auto-approve"},
		{name: "docker prune all", command: "docker system prune --all", block: true, needle: "docker system prune -a"},
		{name: "rm rf root", command: "rm -rf /", block: true, needle: "rm -rf on system path"},
		{name: "rm rf tmp root", command: "rm -rf /tmp", block: true, needle: "rm -rf on system path"},
		{name: "rm rf usr root", command: "rm -rf /usr/", block: true, needle: "rm -rf on system path"},
		{name: "rm rf quoted root", command: `rm -rf '/etc/'`, block: true, needle: "rm -rf on system path"},
		{name: "rm rf dot root alias", command: "rm -rf /etc/.", block: true, needle: "rm -rf on system path"},
		{name: "rm rf repeated slash root alias", command: "rm -rf /etc//", block: true, needle: "rm -rf on system path"},
		{name: "rm rf normalized root alias", command: "rm -rf /usr/../etc", block: true, needle: "rm -rf on system path"},
		{name: "rm rf root before pipe", command: "rm -rf /var|tee log", block: true, needle: "rm -rf on system path"},
		{name: "rm rf root before redirection", command: "rm -rf /etc>/tmp/log", block: true, needle: "rm -rf on system path"},
		{name: "rm rf scratch directory", command: "rm -rf /tmp/loaf-scratch"},
		{name: "rm rf dependency directory", command: "rm -rf /Users/example/project/node_modules"},
		{name: "rm rf build directory", command: "rm -rf /opt/app/build"},
		{name: "rm rf root name prefix", command: "rm -rf /usr-local"},
		{name: "rm rf quoted scratch directory", command: `rm -rf '/tmp/loaf-scratch'`},
		{name: "rm rf quoted path with spaces", command: `rm -rf '/tmp build'`},
		{name: "delete and list separate lines", command: "kubectl delete pod web-0\nkubectl get pods --all-namespaces", warning: "kubectl delete detected"},
		{name: "delete and list separate clauses", command: "kubectl delete pod web-0; kubectl get pods --all-namespaces", warning: "kubectl delete detected"},
		{name: "destroy and apply separate lines", command: "terraform destroy\nterraform apply -auto-approve", warning: "terraform destroy detected"},
		{name: "real destructive second line", command: "kubectl get pods\nkubectl delete pods --all", block: true, needle: "kubectl delete --all"},
		{name: "continued kubectl command", command: "kubectl delete pods \\\n--all", block: true, needle: "kubectl delete --all"},
		{name: "continued terraform command", command: "terraform destroy \\\n-auto-approve", block: true, needle: "terraform destroy -auto-approve"},
		{name: "kubectl delete warn", command: "kubectl delete pod web-0", warning: "kubectl delete detected"},
		{name: "terraform destroy warn", command: "terraform destroy", warning: "terraform destroy detected"},
		{name: "docker force warn", command: "docker rm -f web", warning: "Forced docker removal"},
		{name: "helm uninstall warn", command: "helm uninstall loaf", warning: "helm uninstall detected"},
		{name: "safe command", command: "kubectl get pods"},
		{name: "rg quoted delete namespace", command: "rg -n 'kubectl delete namespace' content"},
		{name: "grep quoted delete namespace", command: `grep -n "kubectl delete namespace" content`},
		{name: "sudo rg quoted delete namespace", command: "sudo -n rg -n 'kubectl delete namespace' content"},
		{name: "bash c rg quoted delete namespace", command: `bash -c "rg -n 'kubectl delete namespace' content"`},
		{name: "quoted search then real delete", command: "rg -n 'kubectl delete namespace' content && kubectl delete namespace prod", block: true, needle: "kubectl delete namespace"},
		{name: "echo command substitution", command: `echo "$(kubectl delete namespace review-demo)"`, block: true, needle: "kubectl delete namespace"},
		{name: "find exec delete namespace", command: `find . -exec kubectl delete namespace review-demo \;`, block: true, needle: "kubectl delete namespace"},
		{name: "rg backgrounded delete", command: "rg needle . & kubectl delete namespace review-demo", block: true, needle: "kubectl delete namespace"},
		{name: "awk system delete namespace", command: `awk 'BEGIN {system("kubectl delete namespace review-demo")}'`, block: true, needle: "kubectl delete namespace"},
		{name: "echo mixed quote substitution", command: `echo 'prefix'"$(kubectl delete namespace review-demo)"`, block: true, needle: "kubectl delete namespace"},
		{name: "echo mixed unquoted substitution", command: `echo 'prefix'$(kubectl delete namespace review-demo)`, block: true, needle: "kubectl delete namespace"},
		{name: "rg mixed quote substitution", command: `rg -n 'prefix'"$(kubectl delete namespace review-demo)" content`, block: true, needle: "kubectl delete namespace"},
		{name: "rg mixed unquoted substitution", command: `rg -n 'prefix'$(kubectl delete namespace review-demo) content`, block: true, needle: "kubectl delete namespace"},
		{name: "echo quoted substitution text", command: `echo '$(kubectl delete namespace review-demo)'`, block: true, needle: "kubectl delete namespace"},
		{name: "git grep needle", command: "git grep needle"},
		{name: "git grep -n -e pattern -- path", command: "git grep -n -e pattern -- path"},
		{name: "git grep -O pager delete", command: "git grep -O 'kubectl delete namespace review-demo' needle", block: true, needle: "kubectl delete namespace"},
		{name: "git grep --open-files-in-pager", command: "git grep --open-files-in-pager 'kubectl delete namespace review-demo' needle", block: true, needle: "kubectl delete namespace"},
		{name: "git grep --open-files-in-pager equals kubectl", command: "git grep '--open-files-in-pager=kubectl delete namespace review-demo' needle", block: true, needle: "kubectl delete namespace"},
		{name: "git grep bundled -nO pager delete", command: "git grep -nO 'kubectl delete namespace review-demo' needle", block: true, needle: "kubectl delete namespace"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Runner{
				Stdout:     &stdout,
				Stderr:     &stderr,
				WorkingDir: root,
				Stdin:      bytes.NewBufferString(checkBashContext(tc.command)),
			}.Run([]string{"check", "--hook", "validate-infra-safety", "--json"})
			var output checkJSONOutput
			if unmarshalErr := json.Unmarshal(stdout.Bytes(), &output); unmarshalErr != nil {
				t.Fatalf("json: %v stdout=%s", unmarshalErr, stdout.String())
			}
			if tc.block {
				var exitErr ExitError
				if !errors.As(err, &exitErr) || exitErr.Code != 2 || !output.Blocked {
					t.Fatalf("err=%v output=%#v, want block", err, output)
				}
				if !strings.Contains(strings.Join(output.Errors, "\n"), tc.needle) {
					t.Fatalf("errors=%#v, want %q", output.Errors, tc.needle)
				}
				return
			}
			if err != nil {
				t.Fatalf("err=%v stderr=%s", err, stderr.String())
			}
			if output.Blocked {
				t.Fatalf("blocked unexpected: %#v", output)
			}
			if tc.warning != "" && !strings.Contains(strings.Join(output.Warnings, "\n"), tc.warning) {
				t.Fatalf("warnings=%#v, want %q", output.Warnings, tc.warning)
			}
		})
	}
}

func TestRunnerCheckSQLSafetyBlocksAndWarns(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	cases := []struct {
		name    string
		command string
		block   bool
		warning string
		needle  string
	}{
		{name: "drop table", command: "psql -c 'drop table users'", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "drop database", command: "DROP DATABASE bakery", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "drop schema", command: "drop schema public cascade", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "truncate", command: "TRUNCATE TABLE events", block: true, needle: "TRUNCATE"},
		{name: "delete without where", command: "DELETE FROM users", warning: "DELETE without WHERE"},
		{name: "delete with where", command: "DELETE FROM users WHERE id = 1"},
		{name: "drop column", command: "ALTER TABLE users DROP COLUMN email", warning: "DROP COLUMN"},
		{name: "unrelated drop column text on next line", command: "psql -c 'ALTER TABLE users ADD COLUMN age int'\necho 'DROP COLUMN example'"},
		{name: "drop index allowed", command: "DROP INDEX users_email_idx"},
		{name: "select", command: "SELECT * FROM users"},
		{name: "rg quoted drop table", command: "rg -n 'DROP TABLE' internal"},
		{name: "git grep quoted drop table", command: "git grep -n 'DROP TABLE'"},
		{name: "git grep needle", command: "git grep needle"},
		{name: "git grep -n -e pattern -- path", command: "git grep -n -e pattern -- path"},
		{name: "git grep --open-files-in-pager equals drop table", command: "git grep '--open-files-in-pager=psql -c DROP TABLE users' needle", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "git grep -O drop table", command: "git grep -O 'DROP TABLE users' needle", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "git grep --open-files-in-pager drop table", command: "git grep --open-files-in-pager 'DROP TABLE users' needle", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "echo quoted drop table", command: "echo 'DROP TABLE users'", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "bash c rg quoted drop table", command: `bash -c "rg -n 'DROP TABLE' internal"`},
		{name: "echo pipe psql still destructive", command: "echo 'DROP TABLE users' | psql", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "echo sql command substitution", command: `echo "$(psql -c 'DROP TABLE users')"`, block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "echo unquoted sql substitution", command: "echo $(DROP TABLE users)", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "echo mixed quote sql substitution", command: `echo 'prefix'"$(psql -c 'DROP TABLE users')"`, block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "echo mixed unquoted sql substitution", command: `echo 'prefix'$(psql -c 'DROP TABLE users')`, block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "awk system drop table", command: `awk 'BEGIN {system("psql -c \"DROP TABLE users\"")}'`, block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
		{name: "rg backgrounded drop table", command: "rg needle . & psql -c 'DROP TABLE users'", block: true, needle: "DROP DATABASE/TABLE/SCHEMA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			err := Runner{
				Stdout:     &stdout,
				WorkingDir: root,
				Stdin:      bytes.NewBufferString(checkBashContext(tc.command)),
			}.Run([]string{"check", "--hook", "validate-sql-safety", "--json"})
			var output checkJSONOutput
			if unmarshalErr := json.Unmarshal(stdout.Bytes(), &output); unmarshalErr != nil {
				t.Fatalf("json: %v stdout=%s", unmarshalErr, stdout.String())
			}
			if tc.block {
				var exitErr ExitError
				if !errors.As(err, &exitErr) || exitErr.Code != 2 || !output.Blocked {
					t.Fatalf("err=%v output=%#v, want block", err, output)
				}
				if !strings.Contains(strings.Join(output.Errors, "\n"), tc.needle) {
					t.Fatalf("errors=%#v, want %q", output.Errors, tc.needle)
				}
				return
			}
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if output.Blocked {
				t.Fatalf("blocked unexpected: %#v", output)
			}
			if tc.warning != "" && !strings.Contains(strings.Join(output.Warnings, "\n"), tc.warning) {
				t.Fatalf("warnings=%#v, want %q", output.Warnings, tc.warning)
			}
			if tc.warning == "" && len(output.Warnings) != 0 {
				t.Fatalf("unexpected warnings: %#v", output.Warnings)
			}
		})
	}
}

func TestRunnerCheckSafetyHooksFailClosedOnBadPayloads(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	for _, hook := range []string{"validate-infra-safety", "validate-sql-safety"} {
		t.Run(hook+" malformed", func(t *testing.T) {
			var stdout bytes.Buffer
			err := Runner{
				Stdout:     &stdout,
				WorkingDir: root,
				Stdin:      bytes.NewBufferString("{not json"),
			}.Run([]string{"check", "--hook", hook, "--json"})
			assertSafetyHookBlocked(t, err, stdout.Bytes(), "malformed hook payload")
		})
		t.Run(hook+" unreadable", func(t *testing.T) {
			var stdout bytes.Buffer
			err := Runner{
				Stdout:     &stdout,
				WorkingDir: root,
				Stdin:      failReader{err: errors.New("pipe closed")},
			}.Run([]string{"check", "--hook", hook, "--json"})
			assertSafetyHookBlocked(t, err, stdout.Bytes(), "unreadable hook payload")
		})
		t.Run(hook+" oversized", func(t *testing.T) {
			var stdout bytes.Buffer
			err := Runner{
				Stdout:     &stdout,
				WorkingDir: root,
				Stdin:      strings.NewReader(strings.Repeat("x", projectFileReadLimit+1)),
			}.Run([]string{"check", "--hook", hook, "--json"})
			assertSafetyHookBlocked(t, err, stdout.Bytes(), "unreadable hook payload")
		})
		t.Run(hook+" empty command allowed", func(t *testing.T) {
			var stdout bytes.Buffer
			err := Runner{
				Stdout:     &stdout,
				WorkingDir: root,
				Stdin:      bytes.NewBufferString(`{"tool":{"name":"Bash"},"tool_input":{}}`),
			}.Run([]string{"check", "--hook", hook, "--json"})
			if err != nil {
				t.Fatalf("empty command: %v %s", err, stdout.String())
			}
		})
	}
}

func TestRunnerCheckSecretsStillPassesOnMalformedJSON(t *testing.T) {
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: t.TempDir(),
		Stdin:      bytes.NewBufferString("not valid json"),
	}.Run([]string{"check", "--hook", "check-" + "secrets"})
	if err != nil {
		t.Fatalf("legacy fail-open changed: %v", err)
	}
}

func TestRunnerCheckHooksRejectOversizedPayloads(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	// The same synthetic secret must not disappear when input exceeds the cap.
	secret := "AKIA" + strings.Repeat("A", 16)
	payload, err := json.Marshal(map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]string{"file_path": "example.txt", "content": secret + strings.Repeat("x", projectFileReadLimit)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, hook := range sortedKeys(validCheckHooks) {
		if hook == "kb-staleness-nudge" {
			continue // This advisory hook has its own intentionally non-blocking reader.
		}
		t.Run(hook, func(t *testing.T) {
			var stdout bytes.Buffer
			err := Runner{Stdout: &stdout, WorkingDir: root, Stdin: bytes.NewReader(payload)}.Run([]string{"check", "--hook", hook, "--json"})
			assertSafetyHookBlocked(t, err, stdout.Bytes(), "exceeds the project read limit")
		})
	}
	t.Run("explicit advisory preserves exit zero", func(t *testing.T) {
		var stdout bytes.Buffer
		err := Runner{Stdout: &stdout, WorkingDir: root, Stdin: bytes.NewReader(payload)}.Run([]string{"check", "--hook", "check-secrets", "--advisory", "--json"})
		if err != nil {
			t.Fatal(err)
		}
		var output checkJSONOutput
		if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
			t.Fatal(err)
		}
		if output.Passed || output.ExitCode != 0 || !strings.Contains(strings.Join(output.Errors, "\n"), "exceeds the project read limit") {
			t.Fatalf("output=%#v, want reported inspection failure with advisory exit zero", output)
		}
	})
}

func TestRunnerCheckSecretsScansPayloadAtReadLimit(t *testing.T) {
	for _, secret := range []bool{false, true} {
		content := "safe"
		if secret {
			content = "AKIA" + strings.Repeat("A", 16)
		}
		body := `{"tool_name":"Write","tool_input":{"file_path":"example.txt","content":"` + content + `"}}`
		// JSON permits trailing whitespace; the entire document is exactly at the cap.
		body += strings.Repeat(" ", projectFileReadLimit-len(body))
		var stdout bytes.Buffer
		err := Runner{Stdout: &stdout, WorkingDir: t.TempDir(), Stdin: strings.NewReader(body)}.Run([]string{"check", "--hook", "check-secrets", "--json"})
		if secret {
			assertSafetyHookBlocked(t, err, stdout.Bytes(), "Potential secrets")
		} else if err != nil {
			t.Fatalf("safe payload at limit: %v %s", err, stdout.String())
		}
	}
}

func TestRunnerCheckSecretsRejectsOversizedDataWithReadError(t *testing.T) {
	var stdout bytes.Buffer
	err := Runner{
		Stdout: &stdout, WorkingDir: t.TempDir(),
		Stdin: terminalErrorReader{strings.NewReader(strings.Repeat("x", projectFileReadLimit+1))},
	}.Run([]string{"check", "--hook", "check-secrets", "--json"})
	assertSafetyHookBlocked(t, err, stdout.Bytes(), "exceeds the project read limit")
}

func assertSafetyHookBlocked(t *testing.T, err error, stdout []byte, needle string) {
	t.Helper()
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Fatalf("err=%v, want exit 2", err)
	}
	var output checkJSONOutput
	if unmarshalErr := json.Unmarshal(stdout, &output); unmarshalErr != nil {
		t.Fatalf("json: %v stdout=%s", unmarshalErr, stdout)
	}
	if !output.Blocked || !strings.Contains(strings.Join(output.Errors, "\n"), needle) {
		t.Fatalf("output=%#v, want blocked %q", output, needle)
	}
}

func TestSafetyInspectableCommandOmitsOnlyInertSearchForms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		command string
		empty   bool
		want    string
	}{
		{name: "rg drop table", command: "rg -n 'DROP TABLE' internal", empty: true},
		{name: "rg kubectl delete", command: "rg -n 'kubectl delete namespace' content", empty: true},
		{name: "grep kubectl delete", command: `grep -n "kubectl delete namespace" content`, empty: true},
		{name: "git grep drop table", command: "git grep -n 'DROP TABLE'", empty: true},
		{name: "git grep needle", command: "git grep needle", empty: true},
		{name: "git grep -n -e pattern -- path", command: "git grep -n -e pattern -- path", empty: true},
		{name: "git grep -O stays visible", command: "git grep -O needle", want: "git grep -O needle"},
		{name: "git grep --open-files-in-pager stays visible", command: "git grep --open-files-in-pager needle", want: "--open-files-in-pager"},
		{name: "git grep pager equals kubectl stays visible", command: "git grep '--open-files-in-pager=kubectl delete namespace review-demo' needle", want: "kubectl delete namespace"},
		{name: "git grep pager equals drop table stays visible", command: "git grep '--open-files-in-pager=psql -c DROP TABLE users' needle", want: "DROP TABLE"},
		{name: "real kubectl", command: "kubectl delete namespace prod", want: "kubectl delete namespace prod"},
		{name: "real psql", command: "psql -c 'drop table users'", want: "psql -c 'drop table users'"},
		{name: "mixed", command: "rg -n 'DROP TABLE' internal && kubectl delete namespace prod", want: "kubectl delete namespace prod"},
		{name: "echo substitution stays visible", command: `echo "$(kubectl delete namespace review-demo)"`, want: `echo "$(kubectl delete namespace review-demo)"`},
		{name: "find exec stays visible", command: `find . -exec kubectl delete namespace review-demo \;`, want: "kubectl delete namespace review-demo"},
		{name: "backgrounded delete stays visible", command: "rg needle . & kubectl delete namespace review-demo", want: "kubectl delete namespace review-demo"},
		{name: "sql substitution stays visible", command: `echo "$(psql -c 'DROP TABLE users')"`, want: "DROP TABLE"},
		{name: "awk system stays visible", command: `awk 'BEGIN {system("kubectl delete namespace review-demo")}'`, want: "kubectl delete namespace"},
		{name: "echo mixed quote stays visible", command: `echo 'prefix'"$(kubectl delete namespace review-demo)"`, want: "kubectl delete namespace"},
		{name: "echo mixed unquoted stays visible", command: `echo 'prefix'$(kubectl delete namespace review-demo)`, want: "kubectl delete namespace"},
		{name: "rg mixed quote stays visible", command: `rg -n 'prefix'"$(kubectl delete namespace review-demo)" content`, want: "kubectl delete namespace"},
		{name: "rg mixed unquoted stays visible", command: `rg -n 'prefix'$(kubectl delete namespace review-demo) content`, want: "kubectl delete namespace"},
		{name: "echo is not an inert search", command: "echo 'DROP TABLE users'", want: "DROP TABLE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := safetyInspectableCommand(tc.command)
			if tc.empty {
				if strings.TrimSpace(got) != "" {
					t.Fatalf("inspectable=%q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("inspectable=%q, want %q", got, tc.want)
			}
		})
	}
}
