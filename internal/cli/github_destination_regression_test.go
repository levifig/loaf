package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const githubDestinationBoardFixture = "PVT_kwHNBkXOAYr6TQ"

func githubDestinationConfigFixture(root string, github map[string]any) map[string]any {
	config := defaultLoafConfig(time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC), root)
	if github != nil {
		config["integrations"].(map[string]any)["github"] = github
	}
	return config
}

func githubDestinationSelectedFixture() map[string]any {
	return map[string]any{"account": "levifig", "tracker": true, "repository": "levifig/loaf", "project_id": githubDestinationBoardFixture}
}

func writeGitHubDestinationConfigFixture(t *testing.T, root string, config map[string]any) {
	t.Helper()
	body, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, filepath.Join(root, ".agents", "loaf.json"), string(body)+"\n")
}

func setupGitHubDestinationFixture(t *testing.T) (string, string) {
	t.Helper()
	setupInstallCommandFixture(t)
	distribution := writeVersionFixture(t)
	project := realpath(t, t.TempDir())
	writeInstallFile(t, filepath.Join(project, "package.json"), `{"name":"stale-project","version":"1.1.1-stale"}`+"\n")
	wrong := githubDestinationSelectedFixture()
	wrong["repository"] = "wrong/distribution"
	wrong["project_id"] = "PVT_wrongDistribution"
	writeGitHubDestinationConfigFixture(t, distribution, githubDestinationConfigFixture(distribution, wrong))
	return project, distribution
}

func runGitHubDestinationFixture(t *testing.T, project, distribution string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := (Runner{Stdout: &out, WorkingDir: project, Executable: distributionFixtureExecutable(distribution)}).Run(args)
	return out.String(), err
}

func assertGitHubDestinationConfigExit(t *testing.T, err error, invalid bool) {
	t.Helper()
	if !invalid {
		if err != nil {
			t.Fatalf("config check error = %v, want success", err)
		}
		return
	}
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Fatalf("config check error = %v, want configuration exit 2", err)
	}
}

func assertGitHubDestinationLocalText(t *testing.T, output string) {
	t.Helper()
	for _, want := range []string{"GitHub destination (project):", "Live board verification: not performed (local configuration only)."} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestGitHubDestinationConfigDiagnosis(t *testing.T) {
	cases := []struct {
		name       string
		change     func(map[string]any)
		invalid    bool
		selected   bool
		field      string
		forbidEcho string
	}{
		{name: "valid", selected: true},
		{name: "legacy_opaque_node_id", selected: true, change: func(g map[string]any) { g["project_id"] = "MDc6UHJvamVjdFYyMTIzNDU2" }},
		{name: "numeric_board_is_not_identifier", selected: true, invalid: true, field: "integrations.github.project_id", change: func(g map[string]any) { g["project_id"] = "123456789" }},
		{name: "missing_repository", selected: true, invalid: true, field: "integrations.github.repository", change: func(g map[string]any) { delete(g, "repository") }},
		{name: "missing_board", selected: true, invalid: true, field: "integrations.github.project_id", change: func(g map[string]any) { delete(g, "project_id") }},
		{name: "repository_wrong_type", selected: true, invalid: true, field: "integrations.github.repository", change: func(g map[string]any) { g["repository"] = 23 }},
		{name: "repository_wrong_shape", selected: true, invalid: true, field: "integrations.github.repository", forbidEcho: "https://example.invalid/private/repository", change: func(g map[string]any) { g["repository"] = "https://example.invalid/private/repository" }},
		{name: "repository_whitespace", selected: true, invalid: true, field: "integrations.github.repository", change: func(g map[string]any) { g["repository"] = "  " }},
		{name: "board_wrong_type", selected: true, invalid: true, field: "integrations.github.project_id", change: func(g map[string]any) { g["project_id"] = []any{"PVT_wrong"} }},
		{name: "board_title_is_not_identifier", selected: true, invalid: true, field: "integrations.github.project_id", change: func(g map[string]any) { g["project_id"] = "Loaf" }},
		{name: "board_url_is_not_identifier", selected: true, invalid: true, field: "integrations.github.project_id", forbidEcho: "https://github.com/users/levifig/projects/2", change: func(g map[string]any) { g["project_id"] = "https://github.com/users/levifig/projects/2" }},
		{name: "board_whitespace", selected: true, invalid: true, field: "integrations.github.project_id", change: func(g map[string]any) { g["project_id"] = "  " }},
		{name: "tracker_wrong_type", invalid: true, field: "integrations.github.tracker", change: func(g map[string]any) { g["tracker"] = "true" }},
		{name: "tracker_null", invalid: true, field: "integrations.github.tracker", change: func(g map[string]any) { g["tracker"] = nil }},
		{name: "account_only", change: func(g map[string]any) { delete(g, "tracker"); delete(g, "repository"); delete(g, "project_id") }},
		{name: "enabled_is_not_election", change: func(g map[string]any) {
			delete(g, "tracker")
			delete(g, "repository")
			delete(g, "project_id")
			g["enabled"] = true
		}},
		{name: "legacy_title_only_is_not_election", change: func(g map[string]any) { delete(g, "tracker"); delete(g, "project_id"); g["board"] = "Loaf" }},
		{name: "tracker_false", change: func(g map[string]any) { g["tracker"] = false; delete(g, "repository"); delete(g, "project_id") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project, distribution := setupGitHubDestinationFixture(t)
			github := githubDestinationSelectedFixture()
			if tc.change != nil {
				tc.change(github)
			}
			config := githubDestinationConfigFixture(project, github)
			writeGitHubDestinationConfigFixture(t, project, config)
			configPath := filepath.Join(project, ".agents", "loaf.json")
			before := readFileBytes(t, configPath)
			output, err := runGitHubDestinationFixture(t, project, distribution, "config", "check", "--json")
			assertGitHubDestinationConfigExit(t, err, tc.invalid)
			var result map[string]any
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("JSON error = %v\n%s", err, output)
			}
			destination, hasDestination := result["github_destination"].(map[string]any)
			if hasDestination != tc.selected {
				t.Fatalf("github_destination present = %v, want %v:\n%s", hasDestination, tc.selected, output)
			}
			if tc.selected && destination["live_verification"] != "not-performed" {
				t.Fatalf("destination = %#v, want explicit lack of live verification", destination)
			}
			if tc.name == "valid" && (destination["repository"] != "levifig/loaf" || destination["project_id"] != githubDestinationBoardFixture) {
				t.Fatalf("destination = %#v, want exact project-owned identifiers", destination)
			}
			if tc.invalid && !strings.Contains(output, tc.field) {
				t.Fatalf("missing field-specific diagnostic %q:\n%s", tc.field, output)
			}
			if tc.forbidEcho != "" && strings.Contains(output, tc.forbidEcho) {
				t.Fatalf("invalid destination echoed instead of sanitized:\n%s", output)
			}
			text, textErr := runGitHubDestinationFixture(t, project, distribution, "config", "check")
			assertGitHubDestinationConfigExit(t, textErr, tc.invalid)
			if tc.selected {
				assertGitHubDestinationLocalText(t, text)
			}
			if tc.invalid && !strings.Contains(text, tc.field) {
				t.Fatalf("human output missing %q:\n%s", tc.field, text)
			}
			if tc.forbidEcho != "" && strings.Contains(text, tc.forbidEcho) {
				t.Fatalf("human output echoed invalid destination:\n%s", text)
			}
			if tc.name == "valid" && (!strings.Contains(text, "levifig/loaf") || !strings.Contains(text, githubDestinationBoardFixture)) {
				t.Fatalf("human output missing configured identifiers:\n%s", text)
			}
			if !bytes.Equal(before, readFileBytes(t, configPath)) {
				t.Fatal("read-only config diagnosis changed project config")
			}
		})
	}
}

func TestGitHubDestinationVersionKeepsDistributionAndProjectSeparate(t *testing.T) {
	project, distribution := setupGitHubDestinationFixture(t)
	writeGitHubDestinationConfigFixture(t, project, githubDestinationConfigFixture(project, githubDestinationSelectedFixture()))
	for _, command := range []string{"version", "--version", "-v"} {
		t.Run(command, func(t *testing.T) {
			output, err := runGitHubDestinationFixture(t, project, distribution, command)
			if err != nil {
				t.Fatalf("informational version failed: %v", err)
			}
			assertGitHubDestinationLocalText(t, output)
			for _, want := range []string{"9.8.7-test.1", "Skills:  2", "levifig/loaf", githubDestinationBoardFixture} {
				if !strings.Contains(output, want) {
					t.Fatalf("output missing %q:\n%s", want, output)
				}
			}
			for _, wrong := range []string{"1.1.1-stale", "wrong/distribution", "PVT_wrongDistribution"} {
				if strings.Contains(output, wrong) {
					t.Fatalf("version mixed project/distribution roots (%q):\n%s", wrong, output)
				}
			}
		})
	}
}

func TestGitHubDestinationVersionRemainsInformationalForIncompleteConfig(t *testing.T) {
	for _, missing := range []string{"repository", "project_id"} {
		t.Run(missing, func(t *testing.T) {
			project, distribution := setupGitHubDestinationFixture(t)
			github := githubDestinationSelectedFixture()
			delete(github, missing)
			writeGitHubDestinationConfigFixture(t, project, githubDestinationConfigFixture(project, github))
			output, err := runGitHubDestinationFixture(t, project, distribution, "version")
			if err != nil {
				t.Fatalf("version error = %v, want informational success", err)
			}
			assertGitHubDestinationLocalText(t, output)
			if !strings.Contains(output, "integrations.github."+missing) {
				t.Fatalf("missing value not visibly diagnosed:\n%s", output)
			}
		})
	}
}

func TestGitHubDestinationVersionOutsideProjectDoesNotAdoptDistributionRouting(t *testing.T) {
	project, distribution := setupGitHubDestinationFixture(t)
	output, err := runGitHubDestinationFixture(t, project, distribution, "version")
	if err != nil || !strings.Contains(output, "9.8.7-test.1") {
		t.Fatalf("normal version reporting failed: %v\n%s", err, output)
	}
	for _, wrong := range []string{"GitHub destination", "wrong/distribution", "integrations.github.repository", "integrations.github.project_id"} {
		if strings.Contains(output, wrong) {
			t.Fatalf("outside-project version claimed project routing (%q):\n%s", wrong, output)
		}
	}
	check, checkErr := runGitHubDestinationFixture(t, project, distribution, "config", "check", "--json")
	assertGitHubDestinationConfigExit(t, checkErr, true)
	var status map[string]any
	if err := json.Unmarshal([]byte(check), &status); err != nil {
		t.Fatal(err)
	}
	if _, exists := status["github_destination"]; exists {
		t.Fatalf("missing config adopted distribution destination:\n%s", check)
	}
	if !strings.Contains(check, ".agents/loaf.json is missing") || strings.Contains(check, "integrations.github.repository") {
		t.Fatalf("outside-project config check should retain its missing-config finding:\n%s", check)
	}
}

func TestGitHubDestinationVersionStillShowsProjectWhenDistributionUnresolved(t *testing.T) {
	project, _ := setupGitHubDestinationFixture(t)
	writeGitHubDestinationConfigFixture(t, project, githubDestinationConfigFixture(project, githubDestinationSelectedFixture()))
	bare := realpath(t, t.TempDir())
	var out bytes.Buffer
	err := (Runner{Stdout: &out, WorkingDir: project, Executable: func() (string, error) { return filepath.Join(bare, "loaf"), nil }}).Run([]string{"version"})
	if err != nil {
		t.Fatalf("degraded version failed: %v", err)
	}
	assertGitHubDestinationLocalText(t, out.String())
	for _, want := range []string{"levifig/loaf", githubDestinationBoardFixture, "reinstall Loaf"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("degraded output missing %q:\n%s", want, out.String())
		}
	}
}

func TestGitHubDestinationOtherTrackerAndLegacyAuthorityDoNotElectGitHub(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "other_tracker", true: "legacy_authority"}[legacy], func(t *testing.T) {
			project, distribution := setupGitHubDestinationFixture(t)
			config := githubDestinationConfigFixture(project, map[string]any{"account": "levifig", "board": "Loaf"})
			if legacy {
				config["issue"] = map[string]any{"authority": "github", "prefix": "LOAF"}
			} else {
				config["integrations"].(map[string]any)["linear"] = map[string]any{"enabled": true, "mcp_server_name": "linear-work", "tracker": true}
			}
			writeGitHubDestinationConfigFixture(t, project, config)
			output, err := runGitHubDestinationFixture(t, project, distribution, "config", "check", "--json")
			assertGitHubDestinationConfigExit(t, err, false)
			var parsed map[string]any
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				t.Fatal(err)
			}
			if _, exists := parsed["github_destination"]; exists {
				t.Fatalf("non-GitHub election invented destination:\n%s", output)
			}
			version, versionErr := runGitHubDestinationFixture(t, project, distribution, "version")
			if versionErr != nil || strings.Contains(version, "GitHub destination") {
				t.Fatalf("non-GitHub version claimed destination: %v\n%s", versionErr, version)
			}
		})
	}
}

func TestGitHubDestinationFixPreservesOpaqueRoutingAndNeverInventsChoices(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "account_only", true: "selected"}[selected], func(t *testing.T) {
			project, distribution := setupGitHubDestinationFixture(t)
			github := map[string]any{"account": "levifig", "board": "old title", "opaque": map[string]any{"nested": []any{"keep", float64(7)}}}
			if selected {
				for key, value := range githubDestinationSelectedFixture() {
					github[key] = value
				}
			}
			config := githubDestinationConfigFixture(project, github)
			config["opaque_root"] = map[string]any{"preserve": []any{true, "yes"}}
			config["integrations"].(map[string]any)["custom-provider"] = map[string]any{"routing": map[string]any{"opaque": "retain"}}
			delete(config, "version")
			writeGitHubDestinationConfigFixture(t, project, config)
			before := readInstallCommandJSON(t, filepath.Join(project, ".agents", "loaf.json"))
			output, err := runGitHubDestinationFixture(t, project, distribution, "config", "check", "--fix", "--json")
			assertGitHubDestinationConfigExit(t, err, false)
			after := readInstallCommandJSON(t, filepath.Join(project, ".agents", "loaf.json"))
			if after["version"] != loafConfigSchemaVersion || !reflect.DeepEqual(before["integrations"], after["integrations"]) || !reflect.DeepEqual(before["opaque_root"], after["opaque_root"]) {
				t.Fatalf("mechanical fix changed project-owned/opaque values:\nbefore=%#v\nafter=%#v\n%s", before, after, output)
			}
		})
	}
	project, distribution := setupGitHubDestinationFixture(t)
	github := githubDestinationSelectedFixture()
	delete(github, "project_id")
	github["board"] = "Loaf"
	writeGitHubDestinationConfigFixture(t, project, githubDestinationConfigFixture(project, github))
	path := filepath.Join(project, ".agents", "loaf.json")
	before := readFileBytes(t, path)
	_, err := runGitHubDestinationFixture(t, project, distribution, "config", "check", "--fix", "--json")
	assertGitHubDestinationConfigExit(t, err, true)
	if !bytes.Equal(before, readFileBytes(t, path)) {
		t.Fatal("--fix invented/replaced destination or title-only routing instead of reporting it")
	}
}

func TestGitHubDestinationStandaloneNeverRunsProviderProgramsOrCreatesDatabase(t *testing.T) {
	project, distribution := setupGitHubDestinationFixture(t)
	writeGitHubDestinationConfigFixture(t, project, githubDestinationConfigFixture(project, githubDestinationSelectedFixture()))
	calls := filepath.Join(t.TempDir(), "provider-calls")
	t.Setenv("GITHUB_DESTINATION_TEST_CALLS", calls)
	for _, name := range []string{"gh", "curl", "wget"} {
		path := filepath.Join(os.Getenv("PATH"), name)
		writeInstallFile(t, path, "#!/bin/sh\nprintf 'provider invoked\\n' >> \"$GITHUB_DESTINATION_TEST_CALLS\"\nexit 99\n")
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	before := readFileBytes(t, filepath.Join(project, ".agents", "loaf.json"))
	for _, args := range [][]string{{"version"}, {"config", "check", "--json"}, {"config", "check"}} {
		if output, err := runGitHubDestinationFixture(t, project, distribution, args...); err != nil {
			t.Fatalf("standalone command %v failed: %v\n%s", args, err, output)
		}
	}
	for _, path := range []string{calls, os.Getenv("LOAF_DB")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("read-only local diagnosis created/invoked %s: %v", path, err)
		}
	}
	if !bytes.Equal(before, readFileBytes(t, filepath.Join(project, ".agents", "loaf.json"))) {
		t.Fatal("standalone commands changed config")
	}
}

func TestGitHubDestinationNestedProjectResolutionMatchesConfig(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable for nested-root proof")
	}
	project, distribution := setupGitHubDestinationFixture(t)
	t.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+filepath.Dir(gitPath))
	if output, err := exec.Command(gitPath, "init", "-q", project).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	writeGitHubDestinationConfigFixture(t, project, githubDestinationConfigFixture(project, githubDestinationSelectedFixture()))
	nested := filepath.Join(project, "nested", "directory")
	mkdirAll(t, nested)
	for _, args := range [][]string{{"version"}, {"config", "check", "--json"}} {
		output, err := runGitHubDestinationFixture(t, nested, distribution, args...)
		if err != nil || !strings.Contains(output, "levifig/loaf") || !strings.Contains(output, githubDestinationBoardFixture) {
			t.Fatalf("nested project routing differs for %v: %v\n%s", args, err, output)
		}
	}
}

func TestGitHubDestinationFixPreservesOpaqueNumericPrecision(t *testing.T) {
	project, distribution := setupGitHubDestinationFixture(t)
	config := githubDestinationConfigFixture(project, githubDestinationSelectedFixture())
	config["integrations"].(map[string]any)["custom-provider"] = json.RawMessage(`{"integer":9007199254740993,"decimal":1.234567890123456789,"exponent":9.007199254740993e+15}`)
	delete(config, "version")
	writeGitHubDestinationConfigFixture(t, project, config)
	output, err := runGitHubDestinationFixture(t, project, distribution, "config", "check", "--fix", "--json")
	assertGitHubDestinationConfigExit(t, err, false)
	decoder := json.NewDecoder(bytes.NewReader(readFileBytes(t, filepath.Join(project, ".agents", "loaf.json"))))
	decoder.UseNumber()
	var actual map[string]any
	if err := decoder.Decode(&actual); err != nil {
		t.Fatal(err)
	}
	provider := actual["integrations"].(map[string]any)["custom-provider"].(map[string]any)
	for key, want := range map[string]string{"integer": "9007199254740993", "decimal": "1.234567890123456789", "exponent": "9.007199254740993e+15"} {
		if provider[key] != json.Number(want) {
			t.Fatalf("opaque %s changed: got %v, want %s; %s", key, provider[key], want, output)
		}
	}
}

func TestGitHubDestinationOpaqueLargeNumberDoesNotHideVersionRouting(t *testing.T) {
	project, distribution := setupGitHubDestinationFixture(t)
	config := githubDestinationConfigFixture(project, githubDestinationSelectedFixture())
	config["opaque_root"] = json.RawMessage(`{"large":1e400}`)
	writeGitHubDestinationConfigFixture(t, project, config)
	for _, args := range [][]string{{"version"}, {"config", "check", "--json"}} {
		output, err := runGitHubDestinationFixture(t, project, distribution, args...)
		if err != nil || !strings.Contains(output, githubDestinationBoardFixture) {
			t.Fatalf("opaque number hid routing for %v: %v; %s", args, err, output)
		}
	}
}
