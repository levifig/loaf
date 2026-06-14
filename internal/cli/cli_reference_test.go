package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerGenerateCLIReferenceWritesSkillNatively(t *testing.T) {
	workingDir := realpath(t, t.TempDir())
	var stdout bytes.Buffer

	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
	}.Run([]string{"__generate-cli-ref"})
	if err != nil {
		t.Fatalf("__generate-cli-ref error = %v", err)
	}

	outputPath := filepath.Join(workingDir, "content", "skills", "cli-reference", "SKILL.md")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", outputPath, err)
	}
	content := string(data)
	for _, want := range []string{
		"**Note:** This file is auto-generated from native CLI reference metadata.",
		"`-t, --target <name>` - Build a specific target only",
		"`loaf state repair legacy-project-database`",
		"`--dry-run` - Preview archive paths without writing",
		"`loaf state repair relationship-origin`",
		"`--origin <imported|manual>` - Provenance value to backfill",
		"`loaf state backup verify`",
		"`--json` - Output verification details as JSON",
		"`loaf state path`",
		"`--verbose` - Output command, scope, project root, and database path",
		"`loaf state backup` | Create a SQLite database backup under the global data-home backups directory",
		"`loaf state export`",
		"`--format <format>` - Output format for the selected export kind",
		"`loaf state export all`",
		"`--format <format>` - Output format: json",
		"`loaf state export release-readiness`",
		"`--format <format>` - Output format: markdown",
		"`loaf migrate worktree-storage`",
		"`--apply` - Perform the migration; dry-run is the default",
		"`--force-from-worktree` - On conflict, keep the worktree-local copy",
		"`--force-from-main` - On conflict, keep the main-worktree copy",
		"## Project Management",
		"`loaf project list`",
		"`--json` - Output database path, project IDs, friendly names, and current paths as JSON",
		"`loaf project rename`",
		"`--dry-run` - Validate and preview without writing",
		"`loaf project move`",
		"## Task Management",
		"`loaf task create`",
		"`--json` - Output created task as JSON",
		"`--status <status>` - Only show tasks with status: in_progress, blocked, todo, review, done, archived",
		"`--priority <level>` - Priority level: P0, P1, P2, P3",
		"`--status <status>` - New status: in_progress, blocked, todo, review, done",
		"`--priority <level>` - New priority: P0, P1, P2, P3",
		"`--json` - Output updated task as JSON",
		"`--json` - Output archive result as JSON",
		"`--json` - Output compatibility summary as JSON",
		"## Trace Management",
		"`loaf trace`",
		"`--json` - Output relationship trace as JSON",
		"## Idea Management",
		"`loaf idea capture`",
		"`--title <title>` - Idea title",
		"`loaf idea resolve`",
		"`--by <entity>` - Resolving entity",
		"## Spark Management",
		"`loaf spark capture`",
		"`--scope <scope>` - Spark scope",
		"`--text <text>` - Spark text",
		"## Bundle Management",
		"`loaf bundle create`",
		"`--tags <tags>` - Comma-separated tag query",
		"`loaf bundle update`",
		"## Link Management",
		"`loaf link create`",
		"`--from <entity>` - Source entity",
		"`--to <entity>` - Target entity",
		"`--type <type>` - Relationship type",
		"## Kb Management",
		"`loaf kb check`",
		"## Command Substitution Reference",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated CLI reference missing %q\n%s", want, content)
		}
	}
	if !strings.Contains(stdout.String(), outputPath) {
		t.Fatalf("stdout = %q, want generated path %q", stdout.String(), outputPath)
	}
	if !strings.Contains(content, "- `loaf report generate`:\n  - `--format <format>` - Output format: markdown") {
		t.Fatalf("generated CLI reference missing report generate markdown format guidance\n%s", content)
	}
	if !strings.Contains(content, "  - `--json` - Output JSON wrapper with markdown content") {
		t.Fatalf("generated CLI reference missing report generate JSON guidance\n%s", content)
	}
	if !strings.Contains(content, "- `loaf report list`:\n  - `--type <type>` - Filter by report type\n  - `--status <status>` - Filter by status; Loaf lifecycle statuses: draft, final, archived") {
		t.Fatalf("generated CLI reference missing report list status guidance\n%s", content)
	}
}
