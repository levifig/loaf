package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffDeliveryGuidanceShipsToEveryTarget(t *testing.T) {
	root := setupIsolatedRepositoryBuildRoot(t)
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatalf("build: %v\n%s", err, stdout.String())
	}
	for _, target := range defaultBuildTargets {
		t.Run(target, func(t *testing.T) {
			skills := nativeBuildSkillTreeDir(root, target)
			body := readBuildFileString(t, filepath.Join(skills, "handoff", "SKILL.md"))
			frontmatter, _ := splitNativeBuildFrontmatter(body)
			for _, stale := range []string{".agents/handoffs/", "disposable", "parked for later"} {
				if strings.Contains(frontmatter, stale) {
					t.Errorf("routing description retains %q", stale)
				}
			}
			for _, required := range []string{
				"deliberate context transfers", "ordinary capture",
				"loaf handoff new", "loaf handoff show", "loaf handoff list", "loaf search",
				"direct harness delivery", "explicitly unpersisted",
				"portable snapshot", "source project", "re-read",
				"does not authorize", "Preserve existing handoff IDs",
			} {
				if !strings.Contains(body, required) {
					t.Errorf("Handoff lacks %q", required)
				}
			}
			for _, stale := range []string{"Save handoffs to", "## Lifecycle", "Deprecation Criteria", "Deletes deprecated handoffs"} {
				if strings.Contains(body, stale) {
					t.Errorf("Handoff retains obsolete instruction %q", stale)
				}
			}
			template := readBuildFileString(t, filepath.Join(skills, "handoff", "templates", "handoff.md"))
			for _, stale := range []string{".agents/handoffs/", "deprecated_at:", "status:", "## Deprecation Criteria"} {
				if strings.Contains(template, stale) {
					t.Errorf("packet template retains file lifecycle %q", stale)
				}
			}
			switch target {
			case "claude-code", "opencode", "cursor":
				librarian := readBuildFileString(t, filepath.Join(nativeBuildTargetOutputDir(root, target), "agents", "librarian.md"))
				if strings.Contains(librarian, ".agents/handoffs/") {
					t.Error("Librarian still prescribes filesystem handoff tending")
				}
			}
		})
	}
}
