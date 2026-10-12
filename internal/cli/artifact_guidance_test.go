package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactBodyWriteRespectsFileBackedEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, path, existing string
		blocked              bool
	}{
		{name: "new temporary report", path: ".agents/reports/20261012-module-depth.md"},
		{name: "existing temporary report", path: ".agents/reports/20261012-module-depth.md", existing: "# Analysis\nKeep this evidence.\n"},
		{name: "file council", path: ".agents/councils/COUNCIL-20261012-options.md", existing: "---\nstatus: draft\n---\n# Deliberation\n"},
		{name: "SQLite report projection", path: ".agents/reports/report-options.md", existing: "<!-- loaf:render entity_id=report-options -->\n# Render\n", blocked: true},
		{name: "persisted handoff", path: ".agents/handoffs/transfer.md", blocked: true},
	} {
		for _, tool := range []string{"Write", "Edit", "Bash"} {
			t.Run(tc.name+"/"+tool, func(t *testing.T) {
				root := t.TempDir()
				if tc.existing != "" {
					writeInstallFile(t, filepath.Join(root, tc.path), tc.existing)
				}
				payload, _ := json.Marshal(map[string]any{"tool": map[string]string{"name": tool}, "tool_input": map[string]string{
					"file_path": tc.path, "content": "# Evidence\nUpdated body.\n", "new_string": "Updated body.", "command": "cat > " + tc.path,
				}})
				var stdout bytes.Buffer
				err := (Runner{WorkingDir: root, Stdout: &stdout, Stdin: bytes.NewReader(payload)}).Run([]string{"check", "--hook", "artifact-body-write", "--json"})
				var output checkJSONOutput
				if decodeErr := json.Unmarshal(stdout.Bytes(), &output); decodeErr != nil {
					t.Fatalf("decode: %v; command error: %v; %s", decodeErr, err, stdout.String())
				}
				if output.Blocked != tc.blocked || (err != nil) != tc.blocked {
					t.Fatalf("blocked=%v, error=%v, want blocked=%v: %s", output.Blocked, err, tc.blocked, stdout.String())
				}
				if tc.existing != "" {
					assertInstallFile(t, filepath.Join(root, tc.path), tc.existing)
				}
			})
		}
	}
}

func TestArtifactBodyWriteChecksAbsoluteReportTarget(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	path := filepath.Join(other, ".agents", "reports", "report-options.md")
	body := "<!-- loaf:render entity_id=report-options -->\n# Projection\n"
	writeInstallFile(t, path, body)
	for _, tool := range []string{"Write", "Edit"} {
		payload, _ := json.Marshal(map[string]any{"tool": map[string]string{"name": tool}, "tool_input": map[string]string{"file_path": path, "content": "# Replacement body", "new_string": "Replacement body"}})
		var stdout bytes.Buffer
		err := (Runner{WorkingDir: root, Stdout: &stdout, Stdin: bytes.NewReader(payload)}).Run([]string{"check", "--hook", "artifact-body-write", "--json"})
		if err == nil || !strings.Contains(stdout.String(), `"blocked": true`) {
			t.Fatalf("%s checked cwd instead of the actual report: %v\n%s", tool, err, stdout.String())
		}
	}
	assertInstallFile(t, path, body)
}

func TestArtifactGuidanceReachesIsolatedInstallations(t *testing.T) {
	root := setupIsolatedRepositoryBuildRoot(t)
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatalf("build: %v\n%s", err, stdout.String())
	}
	for _, target := range defaultBuildTargets {
		t.Run(target, func(t *testing.T) {
			skills := nativeBuildSkillTreeDir(root, target)
			agents := filepath.Join(nativeBuildTargetOutputDir(root, target), "agents")
			if target != "claude-code" {
				home := t.TempDir()
				options := targetInstallOptions{Target: target, DistDir: nativeBuildTargetOutputDir(root, target), ConfigDir: filepath.Join(home, "."+target), HomeDir: home, Version: "0.6.0-rc.2", HookState: installTestHookStateAfterMigration(t, target)}
				if err := installTargetDistribution(options); err != nil {
					t.Fatal(err)
				}
				skills = installSkillsDestination(options)
				agents = filepath.Join(options.ConfigDir, "agents")
			}
			proposal := readBuildFileString(t, filepath.Join(skills, "refactor-deepen", "SKILL.md"))
			description, _ := splitNativeBuildFrontmatter(proposal)
			if strings.Contains(description, "PLAN file") || !strings.Contains(description, "analysis") {
				t.Errorf("proposal routing description misstates its output: %s", description)
			}
			for _, stale := range []string{"Workflow handoff is pending", "Write the PLAN", "Plan saved to"} {
				if strings.Contains(proposal, stale) {
					t.Errorf("proposal retains %q", stale)
				}
			}
			handoff := readBuildFileString(t, filepath.Join(skills, "handoff", "SKILL.md"))
			handoffDescription, _ := splitNativeBuildFrontmatter(handoff)
			if strings.Contains(handoffDescription, ".agents/handoffs") || !strings.Contains(handoffDescription, "SQLite") {
				t.Errorf("installed Handoff description misroutes saved transfers: %s", handoffDescription)
			}
			routing := readBuildFileString(t, filepath.Join(skills, "loaf-reference", "references", "artifact-routing.md"))
			for _, required := range []string{"loaf handoff list", "loaf handoff show", "loaf journal search", "loaf journal show", "loaf council show", "explicit file", "unavailable"} {
				if !strings.Contains(routing, required) {
					t.Errorf("artifact routing lacks %q", required)
				}
			}
			if target == "claude-code" || target == "opencode" || target == "cursor" {
				librarian := readBuildFileString(t, filepath.Join(agents, "librarian.md"))
				for _, required := range []string{"draft", "invoking agent", "does not expose a CLI", "report the limitation"} {
					if !strings.Contains(librarian, required) {
						t.Errorf("Librarian capability contract lacks %q", required)
					}
				}
				for _, stale := range []string{"Durable artifact handler for SQLite-backed", "append them\n  to the project journal", "spec changelog"} {
					if strings.Contains(librarian, stale) {
						t.Errorf("Librarian retains unavailable operation %q", stale)
					}
				}
			}
		})
	}
}
