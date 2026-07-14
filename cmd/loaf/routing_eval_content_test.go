package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRoutingEvalDryRunValidatesCurrentSkillSuite(t *testing.T) {
	root := repoRoot(t)
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node not found: %v", err)
	}

	cmd := exec.Command("node", "cli/scripts/eval-skill-routing.mjs", "--dry-run")
	cmd.Dir = root
	cmd.Env = envWith("ANTHROPIC_API_KEY=")
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		t.Fatalf("dry-run routing eval failed: %v\n%s", err, output)
	}

	for _, want := range []string{
		"Loaded 34 skills",
		"Selected cases: 119",
		"Suite validation passed.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, output)
		}
	}
}

func TestRoutingEvalContentHasNoPhantomSkillCases(t *testing.T) {
	root := repoRoot(t)
	body := readTextFile(t, filepath.Join(root, "cli", "scripts", "eval-skill-routing.mjs"))
	for _, forbidden := range []string{
		"council-session",
		"cleanup",
		"resume-session",
		"reference-session",
		"thermo-nuclear-code-quality-review",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("routing eval still references phantom or absent skill %q", forbidden)
		}
	}
}

func TestSkillArchitectureDescribesSemanticTaxonomy(t *testing.T) {
	root := repoRoot(t)
	body := readTextFile(t, filepath.Join(root, "docs", "knowledge", "skill-architecture.md"))
	for _, want := range []string{
		"## Categories",
		"| Category | `user-invocable` | Examples |",
		"| Reference/Knowledge | `false` |",
		"| Workflow/Process | `true` (default) |",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("skill architecture doc missing semantic taxonomy %q", want)
		}
	}
	if regexp.MustCompile(`(?i)\b[0-9]+\s+skills\s+total\b`).MatchString(body) {
		t.Fatal("skill architecture doc publishes a volatile exact skill-count snapshot")
	}
}
