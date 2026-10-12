package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestShapeSizingPreservesIndependentNativeJourneys(t *testing.T) {
	root := repoRoot(t)
	for _, tc := range []struct {
		rel      string
		required []string
	}{
		{"content/skills/shape/SKILL.md", []string{"bounded, independently reviewable", "independently useful, independently verifiable", "Read back the native record"}},
		{"content/skills/shape/references/decomposition.md", []string{"complete vertical journeys", "own observable completion evidence", "reviewed independently", "Dependencies are not hierarchy", "verify both sides through readback"}},
		{"content/skills/implement/SKILL.md", []string{"Preserve the rideable journey across commits and delegation", "reduce breadth before integrity"}},
	} {
		body := readTextFile(t, filepath.Join(root, filepath.FromSlash(tc.rel)))
		for _, required := range tc.required {
			if !strings.Contains(body, required) {
				t.Errorf("%s missing sizing requirement %q", tc.rel, required)
			}
		}
		if strings.Contains(body, "fits one fresh context window and is verifiable alone") {
			t.Errorf("%s still uses the retired window-fit sizing rule", tc.rel)
		}
	}
}

func TestSkillContentHygieneStaleReferences(t *testing.T) {
	root := repoRoot(t)
	cases := []struct {
		rel       string
		forbidden []string
		required  []string
	}{
		{
			rel:       "content/skills/database-design/SKILL.md",
			forbidden: []string{"`infrastructure`"},
			required:  []string{"`infrastructure-management`"},
		},
		{
			rel:       "content/skills/power-systems-modeling/SKILL.md",
			forbidden: []string{"`database-patterns`"},
			required:  []string{"`database-design`"},
		},
		{
			rel: "content/skills/foundations/references/code-style.md",
			forbidden: []string{
				"`python` skill",
				"`typescript` skill",
				"`rails` skill",
			},
			required: []string{
				"`python-development` skill",
				"`typescript-development` skill",
				"`ruby-development` skill",
			},
		},
		{
			rel:       "content/skills/knowledge-base/SKILL.md",
			forbidden: []string{"CLAUDE.md", ".agents/AGENTS.md"},
			required:  []string{"AGENTS.md"},
		},
		{
			rel: "content/skills/orchestration/SKILL.md",
			forbidden: []string{
				"validate-council.py",
				"new-session.sh",
				"YYYYMMDD-HHMMSS-topic.md",
			},
			required: []string{
				"loaf journal log",
				"project-management/v1",
				"Assign one writer",
			},
		},
		{
			rel: "content/skills/council/SKILL.md",
			forbidden: []string{
				"YYYYMMDD}-{HHMMSS}",
				"new-council.sh",
				"validate-council.py",
			},
			required: []string{"loaf council new"},
		},
		{
			rel:       "content/skills/council/templates/council.md",
			forbidden: []string{"YYYYMMDD-HHMMSS-<topic-slug>.md"},
			required:  []string{"loaf council new"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.rel, func(t *testing.T) {
			body := readTextFile(t, filepath.Join(root, filepath.FromSlash(tc.rel)))
			for _, forbidden := range tc.forbidden {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s still contains stale reference %q", tc.rel, forbidden)
				}
			}
			for _, required := range tc.required {
				if !strings.Contains(body, required) {
					t.Fatalf("%s missing replacement reference %q", tc.rel, required)
				}
			}
		})
	}
}

func TestSkillContentHygieneMarkdownStructure(t *testing.T) {
	root := repoRoot(t)
	skillsRoot := filepath.Join(root, "content", "skills")
	var failures []string
	err := filepath.WalkDir(skillsRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "SKILL.md" {
			return nil
		}
		body := readTextFile(t, path)
		lines := strings.Split(body, "\n")
		if len(lines) > 100 && !strings.Contains(body, "\n## Contents\n") {
			failures = append(failures, relToRoot(t, root, path)+": missing ## Contents for "+strconv.Itoa(len(lines))+" lines")
		}
		if fences := strings.Count(body, "```"); fences%2 != 0 {
			failures = append(failures, relToRoot(t, root, path)+": unbalanced fenced code blocks")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s) error = %v", skillsRoot, err)
	}
	sort.Strings(failures)
	if len(failures) > 0 {
		t.Fatalf("skill markdown structure hygiene failures:\n%s", strings.Join(failures, "\n"))
	}
}

func TestArchitectureResourceOwnership(t *testing.T) {
	root := repoRoot(t)
	links := regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	for _, tc := range []struct {
		source string
		target string
	}{
		{"content/skills/documentation-standards/references/documentation.md", "content/skills/architecture/SKILL.md"},
		{"content/skills/architecture/SKILL.md", "content/skills/architecture/templates/architecture-topic.md"},
		{"content/skills/architecture/SKILL.md", "content/skills/architecture/templates/legacy-adr.md"},
		{"content/skills/architecture/SKILL.md", "content/skills/architecture/references/auditing-and-migration.md"},
		{"content/skills/architecture/SKILL.md", "content/skills/architecture/references/legacy-adrs.md"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			source := filepath.Join(root, filepath.FromSlash(tc.source))
			target := filepath.Join(root, filepath.FromSlash(tc.target))
			info, err := os.Stat(target)
			if err != nil {
				t.Fatalf("architecture resource %s: %v", tc.target, err)
			}
			if !info.Mode().IsRegular() {
				t.Fatalf("architecture resource %s is not a regular file", tc.target)
			}
			for _, link := range links.FindAllStringSubmatch(readTextFile(t, source), -1) {
				if filepath.Join(filepath.Dir(source), filepath.FromSlash(link[1])) == target {
					return
				}
			}
			t.Fatalf("%s has no resolving link to %s", tc.source, tc.target)
		})
	}
	for _, retired := range []string{
		"content/templates/adr.md",
		"content/skills/reflect/templates/adr.md",
		"content/skills/documentation-standards/templates/adr.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(retired))); !os.IsNotExist(err) {
			t.Errorf("obsolete duplicate architecture resource %s: stat = %v, want absent", retired, err)
		}
	}
}

func TestSkillHelperExecutableContracts(t *testing.T) {
	root := repoRoot(t)

	for _, retired := range []string{
		"content/skills/infrastructure-management/scripts/check-dockerfile.sh",
		"content/skills/infrastructure-management/scripts/validate-k8s-manifest.py",
		"content/skills/power-systems-modeling/scripts/validate-bounds.py",
		"content/skills/power-systems-modeling/scripts/convert-units.py",
		"content/skills/power-systems-modeling/scripts/check-standard-refs.sh",
		"content/hooks/subagent/validate-infra-safety.sh",
		"content/hooks/subagent/validate-sql-safety.sh",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(retired))); !os.IsNotExist(err) {
			t.Errorf("retired helper %s remains: %v", retired, err)
		}
	}

	powerSkillDir := filepath.Join(root, "content", "skills", "power-systems-modeling")
	scriptsDir := filepath.Join(powerSkillDir, "scripts")
	if info, err := os.Stat(scriptsDir); err == nil && info.IsDir() {
		sidecar := readTextFile(t, filepath.Join(powerSkillDir, "SKILL.claude-code.yaml"))
		if hasShellScripts(t, scriptsDir) && !allowedToolsCanRunShellScripts(sidecar) {
			t.Fatalf("power-systems-modeling ships .sh helpers but sidecar allowed-tools cannot run shell scripts")
		}
	}
}

func TestOrchestrationRetiredHelpersRemainAbsent(t *testing.T) {
	root := repoRoot(t)
	scriptsDir := filepath.Join(root, "content", "skills", "orchestration", "scripts")
	for _, name := range retiredOrchestrationSkillScripts {
		if _, err := os.Stat(filepath.Join(scriptsDir, name)); !os.IsNotExist(err) {
			t.Errorf("retired orchestration helper %s remains: %v", name, err)
		}
	}
}

// These historical orchestration helpers must not return as a second work authority.
var retiredOrchestrationSkillScripts = []string{
	"new-session.sh",
	"git-context-summary.sh",
	"extract-magic-words.sh",
	"new-council.sh",
	"validate-council.py",
	"format-progress.sh",
	"check-linear-format.py",
	"suggest-team.py",
	"get-config.py",
	"validate-roadmap.py",
}

func TestOrchestrationDuplicateAuthorityReferencesRetired(t *testing.T) {
	root := repoRoot(t)
	orchestration := readTextFile(t, filepath.Join(root, "content", "skills", "orchestration", "SKILL.md"))
	for _, retired := range []string{
		"references/councils.md",
		"references/planning.md",
		"references/specs.md",
		"references/product-development.md",
	} {
		if strings.Contains(orchestration, retired) {
			t.Fatalf("orchestration router still links retired duplicate authority %q", retired)
		}
		if _, err := os.Stat(filepath.Join(root, "content", "skills", "orchestration", retired)); !os.IsNotExist(err) {
			t.Fatalf("%s stat = %v, want retired duplicate authority file missing", retired, err)
		}
	}
	for _, owner := range []string{
		"../project-management/SKILL.md",
		"references/delegation.md",
	} {
		if !strings.Contains(orchestration, owner) {
			t.Fatalf("orchestration router missing owning skill link %q", owner)
		}
	}
}

func TestCliReferenceCatalogsJournalFamily(t *testing.T) {
	root := repoRoot(t)
	rel := filepath.FromSlash("docs/knowledge/cli-reference.md")
	body := readTextFile(t, filepath.Join(root, rel))
	// The thinned router lists each command once with its subcommands
	// comma-joined; the journal row must catalog the whole family.
	if !strings.Contains(body, "`loaf journal`") {
		t.Fatalf("%s missing the `loaf journal` command index row", filepath.ToSlash(rel))
	}
	if !strings.Contains(body, "log, recent, search, show, context, export") {
		t.Fatalf("%s missing the journal subcommand family in the command index", filepath.ToSlash(rel))
	}
	// The journal is the supported conversation-continuity namespace; no retired
	// lifecycle command should survive in the generated CLI reference.
	if strings.Contains(body, "loaf session") {
		t.Fatalf("%s still references the deleted `loaf session` namespace", filepath.ToSlash(rel))
	}
}

func TestPlanningVocabularyConverged(t *testing.T) {
	root := repoRoot(t)
	cases := []struct {
		rel       string
		forbidden []string
		required  []string
	}{
		{
			rel: "AGENTS.md",
			forbidden: []string{
				"TRANSITIONAL:",
				"Transition in progress — the Change model is landing.",
				"until the conversion pass",
				"spec-conversion-and-guidance-sweep",
			},
			required: []string{
				"**New shared work is tracker-native.**",
				"project-management/v1",
				"Releases are retroactive",
			},
		},
		{
			rel: "README.md",
			forbidden: []string{
				"**Spec-first pipeline**",
				"## The Pipeline",
				"PHASE 1:",
				"PHASE 2:",
				"PHASE 3:",
				"### Phase 1:",
				"### Phase 2:",
				"### Phase 3:",
			},
			required: []string{
				"**Tracker-native workflow**",
				"## Workflow",
				"canonical native tracker",
				"Loaf never synchronizes",
				"retroactively",
				"/pitch",
			},
		},
		{
			rel: "docs/VISION.md",
			forbidden: []string{
				"Every change flows through a deliberate pipeline",
				"Idea, Spec, Tasks, Code, Learnings",
				"pipeline's three-artifact model",
				"A pipeline that prevents scope creep",
				"## vNext Direction",
				"`/shape` turns a brief into a bounded Change",
			},
			required: []string{
				"The Loaf Flow is pitch → shape → implement → ship → release.",
				"The selected native tracker owns shared work identity",
				"Loaf does not proxy provider traffic",
				"Scratchpad is deferred",
			},
		},
		{
			rel: "docs/STRATEGY.md",
			forbidden: []string{
				"## What Has Been Proven",
				"24 specs shipped, 6 in progress.",
				"1. **Journal continuity** (proven: SPEC-056",
				"4. **Agent routing enforcement** (next: SPEC-022)",
				"## What We Do Not Know Yet",
				"## vNext Reset",
			},
			required: []string{
				"## Strategic Commitments",
				"### Private continuity without session machinery",
				"## Delivery Sequence",
				"## Open Decisions",
				"Historical local issues, tasks, Changes, release cohorts, and receipts are migration or compatibility inputs",
			},
		},
		{
			rel: "docs/knowledge/task-system.md",
			forbidden: []string{
				"# Task System",
				"Loaf implements a Shape Up-inspired task management system",
				"## Pipeline",
				"/shape → SPEC file",
				"## Journal Model (SPEC-056)",
				"One concern per task.",
				"Loaf uses Change artifacts for new bounded work.",
			},
			required: []string{
				"# Work Records and Compatibility",
				"## Current Shared Work",
				"New shared work lives only in the selected native tracker.",
				"These surfaces preserve access to existing data",
				"Migration never introduces background push/pull",
			},
		},
		{
			rel: "docs/ARCHITECTURE.md",
			forbidden: []string{
				"### Stateful Runtime Migration (ADR-014)",
				"The transition shape is a Go front controller",
				"### Mode-Aware Skills (Linear-Native Mode, ADR-011)",
				"The project journal is the **only** session-related structure (SPEC-056).",
				"## Change-First Execution Model",
				"New bounded work uses a Change as its primary contract.",
			},
			required: []string{
				"## Authority Model",
				"## Shipped and Pending Boundaries",
				"## Loaf Flow",
				"## Compatibility Names",
				"entry point to owning architecture topics",
				"agreement is not proof of public or complete implementation",
			},
		},
		{
			rel:       ".github/workflows/release.yml",
			forbidden: []string{"v2.0.0-dev.49", "for example v2.0.0"},
			required:  []string{"Release tag to publish, for example v0.2.20", "if: needs.resolve.outputs.dev != 'true'"},
		},
		{
			rel:       "docs/knowledge/cli-reference.md",
			forbidden: []string{"Markdown-to-native transition"},
			required:  []string{"`transitional-tasks`", "Open task-board records retained for compatibility."},
		},
		{
			rel: "docs/knowledge/cli-reference/command-routing.md",
			forbidden: []string{
				"TRANSITIONAL",
				"conversion pass",
				"spec-conversion-and-guidance-sweep",
				"CLAUDE.md",
			},
			required: []string{
				"Shape new bounded work",
				"Start implementing new bounded work",
				"Continue an existing task or spec record",
				"`loaf task` and `loaf spec` remain readable for legacy records; new work is issues",
			},
		},
		{
			rel:       "config/hooks.yaml",
			forbidden: []string{"Journal-first (SPEC-056)", "active specs"},
			required:  []string{"Journal-first continuity has no session entity", "Block tracked ephemeral Markdown and dangling references from retained spec records"},
		},
		{
			rel:       "internal/state/journal_context.go",
			forbidden: []string{"during U6", "open transitional task"},
			// Split around the alignment padding: gofmt re-pads this block whenever a
			// longer constant name joins it, and the padding is not the contract.
			required: []string{"JournalContextLayerTasks", `= "transitional_tasks"`, `json:"transitional_tasks"`, "these keep existing callers source-compatible.", "open task-board record retained for compatibility"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.rel, func(t *testing.T) {
			body := readTextFile(t, filepath.Join(root, filepath.FromSlash(tc.rel)))
			for _, forbidden := range tc.forbidden {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s still contains leaked planning vocabulary %q", tc.rel, forbidden)
				}
			}
			for _, required := range tc.required {
				if !strings.Contains(body, required) {
					t.Fatalf("%s missing semantic replacement %q", tc.rel, required)
				}
			}
		})
	}
}

func TestOperationalArtifactsUseSemanticFilenames(t *testing.T) {
	root := repoRoot(t)
	implementationUnitPrefix := regexp.MustCompile(`(?i)^u[0-9]+[-_]`)
	developmentUnitFixtureLabel := regexp.MustCompile(`(?i)loaf-u[0-9]+`)
	var filenameFailures []string
	var fixtureFailures []string

	for _, relRoot := range []string{"cli/scripts", "docs/changes", "internal"} {
		walkRoot := filepath.Join(root, filepath.FromSlash(relRoot))
		err := filepath.WalkDir(walkRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel := relToRoot(t, root, path)
			isResearchEvidence := relRoot == "docs/changes" && strings.Contains(filepath.ToSlash(rel), "/research/")
			checkFilename := relRoot == "cli/scripts" || isResearchEvidence
			checkFixtureContent := relRoot == "cli/scripts" || relRoot == "internal"
			if !checkFilename && !checkFixtureContent {
				return nil
			}
			if checkFilename && implementationUnitPrefix.MatchString(entry.Name()) {
				filenameFailures = append(filenameFailures, rel)
			}
			if checkFixtureContent {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if developmentUnitFixtureLabel.Match(content) {
					fixtureFailures = append(fixtureFailures, rel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WalkDir(%s) error = %v", walkRoot, err)
		}
	}

	// Change contracts and historical SPEC/ADR provenance remain valid; only operational runners and retained research evidence must use semantic filenames.
	sort.Strings(filenameFailures)
	sort.Strings(fixtureFailures)
	if len(filenameFailures) > 0 || len(fixtureFailures) > 0 {
		var failures []string
		if len(filenameFailures) > 0 {
			failures = append(failures, "implementation-unit filename prefixes:\n"+strings.Join(filenameFailures, "\n"))
		}
		if len(fixtureFailures) > 0 {
			failures = append(failures, "development-unit fixture labels:\n"+strings.Join(fixtureFailures, "\n"))
		}
		t.Fatalf("operational artifacts still expose planning vocabulary:\n%s", strings.Join(failures, "\n"))
	}
}

func hasShellScripts(t *testing.T, dir string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sh") {
			return true
		}
	}
	return false
}

func allowedToolsCanRunShellScripts(sidecar string) bool {
	for _, line := range strings.Split(sidecar, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "allowed-tools:") {
			continue
		}
		tools := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "allowed-tools:"))
		for _, rawTool := range strings.Split(tools, ",") {
			tool := strings.TrimSpace(rawTool)
			if tool == "Bash" || strings.Contains(tool, ".sh") || strings.Contains(tool, "bash") {
				return true
			}
		}
	}
	return false
}

func relToRoot(t *testing.T, root string, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("Rel(%s, %s) error = %v", root, path, err)
	}
	return filepath.ToSlash(rel)
}

func TestFlowSkillsUseNativeTrackerAuthority(t *testing.T) {
	root := repoRoot(t)
	for _, skill := range []string{"shape", "implement", "ship"} {
		rel := "content/skills/" + skill + "/SKILL.md"
		body := readTextFile(t, filepath.Join(root, filepath.FromSlash(rel)))
		if !strings.Contains(body, "project-management/v1") {
			t.Errorf("%s must route shared work through the selected native provider", rel)
		}
		for _, forbidden := range []string{"loaf issue dod add", "loaf issue check", "loaf issue start", "loaf issue show", "loaf issue new --ref", "loaf issue frontier", "Work units are issues."} {
			if strings.Contains(body, forbidden) {
				t.Errorf("%s still routes new work through frozen local authority %q", rel, forbidden)
			}
		}
	}
}
