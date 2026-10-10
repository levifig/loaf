package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigCheckValidatesGitMergeStrategy(t *testing.T) {
	const wantError = "git.merge_strategy must be one of squash, merge, rebase"
	cases := []struct {
		name string
		git  any
		want []string
	}{
		{name: "squash", git: map[string]any{"merge_strategy": "squash"}},
		{name: "merge", git: map[string]any{"merge_strategy": "merge"}},
		{name: "rebase", git: map[string]any{"merge_strategy": "rebase"}},
		{name: "section without strategy", git: map[string]any{}},
		{name: "case differs from gh flag", git: map[string]any{"merge_strategy": "Squash"}, want: []string{wantError}},
		{name: "unsupported method", git: map[string]any{"merge_strategy": "fast-forward"}, want: []string{wantError}},
		{name: "empty string", git: map[string]any{"merge_strategy": ""}, want: []string{wantError}},
		{name: "non-string", git: map[string]any{"merge_strategy": float64(1)}, want: []string{wantError}},
		{name: "section not an object", git: "squash", want: []string{"git must be an object"}},
	}
	now := time.Date(2026, time.October, 4, 0, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := roundTripDefaultLoafConfig(t, now)
			config["git"] = tc.git
			updated, _, errs := ensureLoafConfigDefaults(config, now)
			if strings.Join(errs, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("errors = %#v, want %#v", errs, tc.want)
			}
			for _, field := range updated {
				if strings.HasPrefix(field, "git") {
					t.Fatalf("updated = %#v, want git left as written", updated)
				}
			}
		})
	}
}

// An unset strategy is valid and stays unset: the git-workflow skill owns the
// fallback, so --fix must not pin a project to squash.
func TestConfigCheckFixNeverInventsMergeStrategy(t *testing.T) {
	now := time.Date(2026, time.October, 4, 0, 0, 0, 0, time.UTC)
	config := map[string]any{}
	_, _, errs := ensureLoafConfigDefaults(config, now)
	if len(errs) != 0 {
		t.Fatalf("errors = %#v, want none", errs)
	}
	if _, exists := config["git"]; exists {
		t.Fatalf("config = %#v, want no git section added", config)
	}
}

func TestConfiguredMergeStrategyReadsOnlyValidValues(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "merge", body: `{"git":{"merge_strategy":"merge"}}`, want: "merge"},
		{name: "rebase", body: `{"git":{"merge_strategy":"rebase"}}`, want: "rebase"},
		{name: "unset", body: `{"git":{}}`, want: ""},
		{name: "invalid", body: `{"git":{"merge_strategy":"octopus"}}`, want: ""},
		{name: "non-string", body: `{"git":{"merge_strategy":true}}`, want: ""},
		{name: "unparseable", body: `{"git":`, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(root, ".agents", "loaf.json"), tc.body)
			if got := configuredMergeStrategy(root); got != tc.want {
				t.Fatalf("configuredMergeStrategy = %q, want %q", got, tc.want)
			}
		})
	}
	if got := configuredMergeStrategy(t.TempDir()); got != "" {
		t.Fatalf("configuredMergeStrategy without loaf.json = %q, want empty", got)
	}
}

func TestBaseBranchAbsorptionWarningFollowsMergeStrategy(t *testing.T) {
	cases := map[string]string{
		"squash": "main has 2 unpushed commit(s) that will be absorbed into this PR's squash merge",
		"merge":  "main has 2 unpushed commit(s) that will land on main through this PR's merge commit",
		"rebase": "main has 2 unpushed commit(s) that will be rebased onto main with this PR's commits",
		"":       "main has 2 unpushed commit(s) that will land on main as part of this PR",
	}
	for strategy, want := range cases {
		if got := baseBranchAbsorptionWarning("main", "2", strategy); got != want {
			t.Errorf("strategy %q warning = %q, want %q", strategy, got, want)
		}
	}
}

func roundTripDefaultLoafConfig(t *testing.T, now time.Time) map[string]any {
	t.Helper()
	body, err := json.Marshal(defaultLoafConfig(now, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatal(err)
	}
	return config
}
