package cli

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestJournalingSkillCommandsMatchCodexPolicy(t *testing.T) {
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

	for _, source := range []string{"content", "vnext/content"} {
		paths, err := filepath.Glob(filepath.Join(repo, source, "skills", "*", "SKILL.md"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("discover %s skills: paths = %v, error = %v", source, paths, err)
		}
		for _, path := range paths {
			body := readBuildFileString(t, path)
			if !strings.Contains(body, "loaf journal log") {
				continue
			}
			t.Run(source+"/"+filepath.Base(filepath.Dir(path)), func(t *testing.T) {
				commands, ok := labeledHarnessSectionBody(body, "## Journal Commands")
				if !ok {
					t.Fatal("journaling skill has no command selection")
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
						args = append(strings.Fields(prefix), "skill(test): command guidance")
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
			})
		}
	}
}
