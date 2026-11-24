package cli

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const journalCommandsOwnerSkill = "foundations"

// journalCommandsOwnerLink is how a skill reaches the single Journal Commands
// section from its sibling directory, in source and in every build target.
const journalCommandsOwnerLink = "../foundations/SKILL.md#journal-commands"

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

// TestJournalCommandsHaveOneOwner keeps the journal command forms, including
// the Codex policy-safe form, in a single skill section that every other
// journaling skill links to instead of copying.
func TestJournalCommandsHaveOneOwner(t *testing.T) {
	for name, body := range journalingSkillBodies(t) {
		_, hasSection := labeledHarnessSectionBody(body, "## Journal Commands")
		if name == journalCommandsOwnerSkill {
			if !hasSection {
				t.Fatalf("%s must own the Journal Commands section", name)
			}
			commands, _ := labeledHarnessSectionBody(body, "## Journal Commands")
			if _, ok := labeledHarnessSectionBody(commands, "### Codex"); !ok {
				t.Fatalf("%s Journal Commands has no Codex section", name)
			}
			continue
		}
		if hasSection {
			t.Errorf("%s copies the Journal Commands section; link to %s instead", name, journalCommandsOwnerLink)
		}
		if strings.Contains(body, "--execpolicy-safe") {
			t.Errorf("%s repeats the Codex journal form; link to %s instead", name, journalCommandsOwnerLink)
		}
		if strings.Contains(body, "loaf journal log") && !strings.Contains(body, journalCommandsOwnerLink) {
			t.Errorf("%s asks for journal writes without linking %s", name, journalCommandsOwnerLink)
		}
	}
}

// TestSkillsDoNotLogTheirOwnInvocation guards against reintroducing the
// retired first-action invocation entry in any skill.
func TestSkillsDoNotLogTheirOwnInvocation(t *testing.T) {
	for name, body := range journalingSkillBodies(t) {
		for _, token := range []string{`loaf journal log "skill(`, "As the first action", "Log invocation first", "log invocation before"} {
			if strings.Contains(body, token) {
				t.Errorf("%s still asks agents to log the skill invocation (%q)", name, token)
			}
		}
	}
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
