package cli

import (
	"encoding/json"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const journalCommandsOwnerSkill = "loaf-core"

// journalCommandsRoute is how every other instruction surface names the single
// owner. Plain text resolves the same way on every harness and install layout,
// including OpenCode commands, where relative skill links do not.
const journalCommandsRoute = "loaf-core skill's Journal Commands section"

func journalingSkillBodies(t *testing.T) map[string]string {
	t.Helper()
	repo := testRepositoryRoot(t)
	paths, err := filepath.Glob(filepath.Join(repo, "content", "skills", "*", "SKILL.md"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("discover skills: paths = %v, error = %v", paths, err)
	}
	bodies := make(map[string]string, len(paths))
	for _, path := range paths {
		bodies[filepath.Base(filepath.Dir(path))] = readBuildFileString(t, path)
	}
	return bodies
}

// walkInstructionMarkdown visits every Markdown file a skill, profile, or
// shared template can deliver, plus the maintained instruction documents.
func walkInstructionMarkdown(t *testing.T, visit func(rel, body string)) {
	t.Helper()
	repo := testRepositoryRoot(t)
	for _, dir := range []string{"content/skills", "content/templates", "content/agents"} {
		root := filepath.Join(repo, filepath.FromSlash(dir))
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			rel, _ := filepath.Rel(repo, path)
			visit(filepath.ToSlash(rel), readBuildFileString(t, path))
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	for _, rel := range []string{"AGENTS.md", "docs/knowledge/skill-architecture.md"} {
		visit(rel, readBuildFileString(t, filepath.Join(repo, filepath.FromSlash(rel))))
	}
}

// TestJournalCommandsHaveOneOwner keeps the runnable journal command forms,
// including the Codex policy-safe form, in Loaf Core alone. Other skills,
// references, templates, and profiles describe the entry and name the owner.
func TestJournalCommandsHaveOneOwner(t *testing.T) {
	owner := journalingSkillBodies(t)[journalCommandsOwnerSkill]
	commands, ok := labeledHarnessSectionBody(owner, "## Journal Commands")
	if !ok {
		t.Fatalf("%s must own the Journal Commands section", journalCommandsOwnerSkill)
	}
	if !strings.Contains(commands, "`loaf journal log \"type(scope): description\"`") {
		t.Fatalf("%s Journal Commands lost the ordinary command form", journalCommandsOwnerSkill)
	}
	codex, ok := labeledHarnessSectionBody(commands, "### Codex")
	if !ok || !strings.Contains(codex, "`loaf journal log --execpolicy-safe \"type(scope): description\"`") {
		t.Fatalf("%s Journal Commands lost the Codex policy-safe form", journalCommandsOwnerSkill)
	}

	ownerPath := "content/skills/" + journalCommandsOwnerSkill + "/SKILL.md"
	walkInstructionMarkdown(t, func(rel, body string) {
		if rel == ownerPath || !strings.HasPrefix(rel, "content/") {
			return
		}
		if strings.Contains(body, "## Journal Commands") {
			t.Errorf("%s copies the Journal Commands section; name the %s instead", rel, journalCommandsRoute)
		}
		if match := runnableJournalCommand.FindString(body); match != "" {
			t.Errorf("%s repeats runnable journal command syntax (%q); describe the entry and name the %s", rel, match, journalCommandsRoute)
		}
		if strings.Contains(body, "--execpolicy-safe") {
			t.Errorf("%s repeats the Codex journal form; name the %s instead", rel, journalCommandsRoute)
		}
		if strings.Contains(body, "#journal-commands") {
			t.Errorf("%s links to the journal commands; relative skill links do not resolve from every install layout, so name the %s", rel, journalCommandsRoute)
		}
	})
}

// runnableJournalCommand matches a journal write with a payload, whatever the
// quoting or flags, while leaving a bare command-index mention alone.
var runnableJournalCommand = regexp.MustCompile(`loaf\s+journal\s+log(\s+--[a-z-]+)*\s+['"]`)

// journalCommandRouteSurfaces are the instruction surfaces outside Loaf Core
// that ask for journal writes. Each must name where the command forms live.
var journalCommandRouteSurfaces = []string{
	"content/skills/bootstrap/SKILL.md",
	"content/skills/brainstorm/SKILL.md",
	"content/skills/council/SKILL.md",
	"content/skills/handoff/SKILL.md",
	"content/skills/idea/SKILL.md",
	"content/skills/reflect/SKILL.md",
	"content/skills/research/SKILL.md",
	"content/skills/strategy/SKILL.md",
	"content/skills/loaf-reference/references/artifact-routing.md",
	"content/templates/journal.md",
}

// TestJournalWritingSurfacesNameTheCommandOwner keeps a route from every
// journal-writing surface to the single command owner, so a Codex agent can
// still find the policy-safe form.
func TestJournalWritingSurfacesNameTheCommandOwner(t *testing.T) {
	repo := testRepositoryRoot(t)
	for _, rel := range journalCommandRouteSurfaces {
		body := readBuildFileString(t, filepath.Join(repo, filepath.FromSlash(rel)))
		if !strings.Contains(body, journalCommandsRoute) {
			t.Errorf("%s asks for journal writes without naming the %s", rel, journalCommandsRoute)
		}
	}
}

// TestMigrationReceiptReportsObservedJournalWrites keeps the receipt from
// asserting a fixed answer about local continuity writes.
func TestMigrationReceiptReportsObservedJournalWrites(t *testing.T) {
	repo := testRepositoryRoot(t)
	receipt := readBuildFileString(t, filepath.Join(repo, "content", "skills", "migrate-project", "templates", "tracker-migration-receipt.md"))
	for _, fixed := range []string{"- Local continuity writes: none\n", "- Local continuity writes: invocation journal entry only"} {
		if strings.Contains(receipt, fixed) {
			t.Errorf("migration receipt prefills a fixed continuity-write answer (%q)", strings.TrimSpace(fixed))
		}
	}
	if !strings.Contains(receipt, "- Local continuity writes: [") {
		t.Error("migration receipt no longer asks for the journal entries actually written")
	}
}

// TestSkillsDoNotLogTheirOwnInvocation guards against reintroducing the
// retired invocation entry in any delivered instruction surface.
func TestSkillsDoNotLogTheirOwnInvocation(t *testing.T) {
	invocationEntry := regexp.MustCompile("skill\\([a-z<][a-z0-9<>-]*\\)")
	walkInstructionMarkdown(t, func(rel, body string) {
		if match := invocationEntry.FindString(body); match != "" {
			t.Errorf("%s still names a skill invocation journal entry (%q)", rel, match)
		}
		for _, token := range []string{"Log invocation first", "log invocation before", "invocation journal entry", "workflow invocation was added", "log themselves first", "log their invocation"} {
			if strings.Contains(body, token) {
				t.Errorf("%s still asks agents to log the skill invocation (%q)", rel, token)
			}
		}
	})
}

func TestJournalCommandOwnerMatchesCodexPolicy(t *testing.T) {
	codex, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("codex command is not installed; skill command classification is not verified")
	}
	t.Setenv("CODEX_HOME", t.TempDir())
	repo := testRepositoryRoot(t)
	template := readBuildFileString(t, filepath.Join(repo, "content", "codex", "rules", "loaf.rules.tmpl"))
	rule, err := renderCodexJournalRule(template)
	if err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(t.TempDir(), "loaf.rules")
	writeFile(t, rulePath, rule)

	body := journalingSkillBodies(t)[journalCommandsOwnerSkill]
	commands, ok := labeledHarnessSectionBody(body, "## Journal Commands")
	if !ok {
		t.Fatal("journal command owner has no command selection")
	}
	codexBody, ok := labeledHarnessSectionBody(commands, "### Codex")
	if !ok {
		t.Fatal("journal command selection has no Codex section")
	}
	var args []string
	for i, example := range strings.Split(codexBody, "`") {
		if i%2 != 1 || !strings.HasPrefix(example, "loaf journal log ") {
			continue
		}
		prefix, _, quoted := strings.Cut(example, ` "`)
		if quoted {
			args = append(strings.Fields(prefix), "decision(test): command guidance")
			break
		}
	}
	if len(args) == 0 {
		t.Fatal("Codex section has no journal command example")
	}
	command := append([]string{"execpolicy", "check", "--rules", rulePath}, args...)
	output, err := exec.Command(codex, command...).CombinedOutput()
	if err != nil {
		t.Fatalf("classify skill command: %v\n%s", err, output)
	}
	var result struct {
		Decision     string `json:"decision"`
		MatchedRules []any  `json:"matchedRules"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode classification: %v\n%s", err, output)
	}
	if result.Decision != "allow" || len(result.MatchedRules) == 0 {
		t.Fatalf("skill command %q is not allowed by the managed policy: %s", args, output)
	}
}
