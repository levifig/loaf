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
		"`loaf state repair legacy-project-database`",
		"`--dry-run` - Preview archive paths without writing",
		"`loaf state repair relationship-origin`",
		"`--origin <imported|manual>` - Provenance value to backfill",
		"## Project Management",
		"`loaf project list`",
		"`--json` - Output database path, project IDs, friendly names, and current paths as JSON",
		"`loaf project rename`",
		"`--dry-run` - Validate and preview without writing",
		"`loaf project move`",
		"## Task Management",
		"`loaf task create`",
		"`--status <status>` - Only show tasks with status: in_progress, blocked, todo, review, done, archived",
		"`--priority <level>` - Priority level: P0, P1, P2, P3",
		"`--status <status>` - New status: in_progress, blocked, todo, review, done",
		"`--priority <level>` - New priority: P0, P1, P2, P3",
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
}
