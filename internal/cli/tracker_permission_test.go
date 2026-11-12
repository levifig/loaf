package cli

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The values config check accepts must be the kinds the project-management
// contract lets a project lift from drafting, so the two lists cannot drift.
func TestTrackerStandingPermissionsMatchProjectManagementContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepositoryRoot(t), "content", "skills", "project-management", "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		DraftPolicy struct {
			StandingPermission map[string][]string `json:"standing_permission"`
		} `json:"draft_policy"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	got := slices.Sorted(slices.Values(trackerStandingPermissions))
	want := slices.Sorted(maps.Keys(contract.DraftPolicy.StandingPermission))
	if !slices.Equal(got, want) || len(want) == 0 {
		t.Errorf("config check accepts %v; project-management contract lists %v", got, want)
	}
}

func TestConfigCheckValidatesTrackerStandingPermission(t *testing.T) {
	const wantError = "tracker.standing_permission must be a list of distinct values from comments, status, edits"
	cases := []struct {
		name    string
		tracker any
		want    []string
	}{
		{name: "empty list", tracker: map[string]any{"standing_permission": []any{}}},
		{name: "one value", tracker: map[string]any{"standing_permission": []any{"status"}}},
		{name: "all values in any order", tracker: map[string]any{"standing_permission": []any{"edits", "comments", "status"}}},
		{name: "section without permission", tracker: map[string]any{}},
		{name: "unknown value", tracker: map[string]any{"standing_permission": []any{"comments", "close"}}, want: []string{wantError}},
		{name: "case differs", tracker: map[string]any{"standing_permission": []any{"Comments"}}, want: []string{wantError}},
		{name: "singular form", tracker: map[string]any{"standing_permission": []any{"comment"}}, want: []string{wantError}},
		{name: "duplicate value", tracker: map[string]any{"standing_permission": []any{"status", "status"}}, want: []string{wantError}},
		{name: "bare string", tracker: map[string]any{"standing_permission": "comments"}, want: []string{wantError}},
		{name: "boolean", tracker: map[string]any{"standing_permission": true}, want: []string{wantError}},
		{name: "null", tracker: map[string]any{"standing_permission": nil}, want: []string{wantError}},
		{name: "non-string element", tracker: map[string]any{"standing_permission": []any{float64(1)}}, want: []string{wantError}},
		{name: "object", tracker: map[string]any{"standing_permission": map[string]any{"comments": true}}, want: []string{wantError}},
		{name: "section not an object", tracker: []any{"comments"}, want: []string{"tracker must be an object"}},
	}
	now := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := roundTripDefaultLoafConfig(t, now)
			config["tracker"] = tc.tracker
			updated, _, errs := ensureLoafConfigDefaults(config, now)
			if strings.Join(errs, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("errors = %#v, want %#v", errs, tc.want)
			}
			for _, field := range updated {
				if strings.HasPrefix(field, "tracker") {
					t.Fatalf("updated = %#v, want tracker left as written", updated)
				}
			}
		})
	}
}

// Absence is the default (everything drafted), so --fix must never write a
// tracker section that would read as a recorded choice.
func TestConfigCheckFixNeverInventsTrackerStandingPermission(t *testing.T) {
	now := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)
	config := map[string]any{}
	_, _, errs := ensureLoafConfigDefaults(config, now)
	if len(errs) != 0 {
		t.Fatalf("errors = %#v, want none", errs)
	}
	if _, exists := config["tracker"]; exists {
		t.Fatalf("config = %#v, want no tracker section added", config)
	}
}

func TestRunnerConfigCheckReportsTrackerStandingPermission(t *testing.T) {
	const base = `"version":"1.0.0","initialized":"2026-11-12T00:00:00Z","knowledge":{"local":["docs/knowledge"],"staleness_threshold_days":30,"imports":[]},"integrations":{}`

	root, _ := setupInstallCommandFixture(t)
	path := filepath.Join(root, ".agents", "loaf.json")
	writeInstallFile(t, path, `{`+base+`,"tracker":{"standing_permission":["comments","Status"]}}`+"\n")
	result := runConfigCheckJSON(t, root, false)
	want := "tracker.standing_permission must be a list of distinct values from comments, status, edits"
	if result.OK || result.Config.Status != "invalid" || strings.Join(result.Config.Errors, "|") != want {
		t.Fatalf("config = %#v, want invalid with %q", result.Config, want)
	}

	writeInstallFile(t, path, `{`+base+`,"tracker":{"standing_permission":["comments","edits"]}}`+"\n")
	result = runConfigCheckJSON(t, root, true)
	if result.Config.Status != "ok" || len(result.Config.Errors) != 0 {
		t.Fatalf("config = %#v, want valid config left ok", result.Config)
	}
	config := readInstallCommandJSON(t, path)
	tracker, _ := config["tracker"].(map[string]any)
	if got := strings.Join(jsonStrings(t, tracker["standing_permission"]), ","); got != "comments,edits" {
		t.Fatalf("standing_permission after --fix = %q, want comments,edits preserved", got)
	}
}
