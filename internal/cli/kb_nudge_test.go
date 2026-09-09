package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerKbStalenessNudgeUsesNativePayloadAndConversationScope(t *testing.T) {
	repo := writeKbCheckFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("TOOL_INPUT", "")
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	gitOnly := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(gitOnly, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", gitOnly)
	for _, tc := range []struct{ name, payload string }{
		{"first conversation", `{"session_id":"one","tool_input":{"file_path":"src/nested/main.go"}}`},
		{"second conversation", `{"session_id":"two","tool":{"input":{"file_path":"src/nested/main.go"}}}`},
		{"escaped path", `{"conversation_id":"three","input":{"file_path":"src/nested/\"quoted\".go"}}`},
		{"flat payload", `{"thread_id":"four","file_path":"src/nested/main.go"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for attempt := 0; attempt < 2; attempt++ {
				var stdout, stderr bytes.Buffer
				err := (Runner{Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(tc.payload), WorkingDir: repo}).Run([]string{"check", "--hook", "kb-staleness-nudge"})
				if err != nil || stderr.Len() != 0 {
					t.Fatalf("nudge: %v stderr=%s", err, stderr.String())
				}
				if attempt == 0 {
					if !strings.Contains(stdout.String(), "docs/knowledge/go.md") || !strings.Contains(stdout.String(), "verify") {
						t.Fatalf("missing substantive-review nudge: %q", stdout.String())
					}
				} else if stdout.Len() != 0 {
					t.Fatalf("duplicate nudge: %q", stdout.String())
				}
			}
		})
	}
}

func TestKbStalenessNudgeFallbackExpiresAndSkipsArchitecture(t *testing.T) {
	repo := writeKbCheckFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	mkdirAll(t, filepath.Join(repo, "docs", "architecture"))
	knowledge := filepath.Join(repo, "docs", "knowledge", "go.md")
	before := readFileBytes(t, knowledge)
	writeFile(t, filepath.Join(repo, "docs", "architecture", "model.md"), string(before))
	payload := map[string]any{"file_path": "src/nested/main.go"}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		now  time.Time
		want int
	}{{now, 1}, {now, 0}, {now.Add(24 * time.Hour), 1}} {
		warnings := kbStalenessNudges(repo, payload, tc.now)
		if len(warnings) != tc.want {
			t.Fatalf("warnings = %v, want %d", warnings, tc.want)
		}
		for _, warning := range warnings {
			if strings.Contains(warning, "architecture/") {
				t.Fatalf("nudged architecture document: %s", warning)
			}
		}
	}
	if !bytes.Equal(before, readFileBytes(t, knowledge)) {
		t.Fatal("nudge changed the knowledge file")
	}
	writeFile(t, knowledge, strings.Replace(string(before), "2000-01-01", time.Now().Add(24*time.Hour).Format("2006-01-02"), 1))
	if warnings := kbStalenessNudges(repo, payload, now.Add(48*time.Hour)); len(warnings) != 0 {
		t.Fatalf("fresh knowledge nudged: %v", warnings)
	}
}

func TestKbNudgeCacheIsAtomicAndDisposable(t *testing.T) {
	cache := t.TempDir()
	var claims atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if claimKbNudge(cache, "project", "conversation", "doc.md") {
				claims.Add(1)
			}
		}()
	}
	group.Wait()
	if got := claims.Load(); got != 1 {
		t.Fatalf("concurrent claims = %d", got)
	}
	for _, keys := range [][3]string{{"other project", "conversation", "doc.md"}, {"project", "other conversation", "doc.md"}, {"project", "conversation", "other doc.md"}} {
		if !claimKbNudge(cache, keys[0], keys[1], keys[2]) {
			t.Fatalf("unrelated nudge suppressed: %v", keys)
		}
	}
	broken := filepath.Join(t.TempDir(), "not-a-directory")
	writeFile(t, broken, "user content")
	if !claimKbNudge(broken, "project", "conversation", "doc.md") || readBuildFileString(t, broken) != "user content" {
		t.Fatal("cache failure suppressed nudge or changed user content")
	}
}

func TestRunnerKbStalenessNudgeIsQuietAndNonblocking(t *testing.T) {
	repo := writeKbCheckFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("TOOL_INPUT", "")
	for _, payload := range []string{"", "invalid JSON", `{"content":"file_path: src/nested/main.go"}`, `{"file_path":"uncovered.txt"}`, `{"file_path":"../src/nested/main.go"}`} {
		var stdout, stderr bytes.Buffer
		err := (Runner{Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(payload), WorkingDir: repo}).Run([]string{"check", "--hook", "kb-staleness-nudge"})
		if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("payload %q: err=%v stdout=%q stderr=%q", payload, err, stdout.String(), stderr.String())
		}
	}
	// No interpreter is needed. Missing Git degrades quietly rather than blocking.
	t.Setenv("PATH", t.TempDir())
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, Stdin: strings.NewReader(`{"file_path":"src/nested/main.go"}`), WorkingDir: repo}).Run([]string{"check", "--hook", "kb-staleness-nudge"})
	if err != nil || stdout.Len() != 0 {
		t.Fatalf("missing Git: %v %q", err, stdout.String())
	}
}

func TestRunnerKbStalenessNudgeEnvironmentAndJSON(t *testing.T) {
	repo := writeKbCheckFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	payload, err := json.Marshal(map[string]string{"file_path": filepath.Join(repo, "src", "nested", "main.go")})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOOL_INPUT", string(payload))
	var stdout bytes.Buffer
	err = (Runner{Stdout: &stdout, Stdin: strings.NewReader("ignored"), WorkingDir: repo}).Run([]string{"check", "--hook", "kb-staleness-nudge", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	var result checkJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Blocked || result.ExitCode != 0 || len(result.Warnings) != 1 {
		t.Fatalf("result = %#v", result)
	}
}
