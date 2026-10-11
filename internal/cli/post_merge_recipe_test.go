package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the actual authored commands with verified PR input. Authority,
// dirty-checkout and remote-identity preconditions still belong to the agent;
// this test proves the Git recipe, not universal adherence to that guidance.
func TestPostMergeRecipesReconcileAuthoritativeForkBase(t *testing.T) {
	for _, surface := range []string{
		"vnext/content/skills/ship/SKILL.md",
		"content/hooks/instructions/post-merge.md",
	} {
		t.Run(surface, func(t *testing.T) {
			body := readBuildFileString(t, filepath.Join(testRepositoryRoot(t), surface))
			_, commands, ok := strings.Cut(body, "   git switch <baseRefName>\n")
			if !ok {
				t.Fatal("post-merge command block missing")
			}
			commands, _, ok = strings.Cut(commands, "   ```")
			if !ok {
				t.Fatal("post-merge command block not closed")
			}
			commands = "git switch <baseRefName>\n" + commands
			for _, tc := range []struct {
				name        string
				remote      string
				customFetch bool
			}{
				{name: "authoritative upstream", remote: "upstream"},
				{name: "custom fetch mapping", remote: "upstream", customFetch: true},
				{name: "wrong fork remote", remote: "origin"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					repo, base, landing, feature := postMergeForkFixture(t)
					if tc.customFetch {
						gitCLI(t, repo, "config", "remote.upstream.fetch", "+refs/heads/*:refs/remotes/verified/*")
						gitCLI(t, repo, "update-ref", "-d", "refs/remotes/upstream/"+base)
					}
					recipe := strings.NewReplacer(
						"<baseRefName>", base,
						"<baseRemote>", tc.remote,
						"<mergeCommit>", landing,
					).Replace(commands)
					cmd := exec.Command("sh", "-ec", recipe)
					cmd.Dir = repo
					output, err := cmd.CombinedOutput()
					if tc.remote == "origin" {
						if err == nil {
							t.Fatalf("fork remote falsely passed reconciliation:\n%s", output)
						}
					} else {
						if err != nil {
							t.Fatalf("authoritative remote reconciliation: %v\n%s", err, output)
						}
						if got := gitOutputReleaseTest(t, repo, "rev-parse", base); got != landing {
							t.Errorf("local base = %s, want observed upstream landing %s", got, landing)
						}
						if err := exec.Command("git", "-C", repo, "branch", "-d", "feature").Run(); err == nil {
							t.Error("squash-merged feature should refuse safe deletion")
						}
					}
					if got := gitOutputReleaseTest(t, repo, "rev-parse", "feature"); got != feature {
						t.Errorf("feature branch changed: %s, want %s", got, feature)
					}
					if got := gitOutputReleaseTest(t, repo, "status", "--porcelain"); got != "" {
						t.Errorf("checkout is dirty after reconciliation: %s", got)
					}
				})
			}
		})
	}
}

func postMergeForkFixture(t *testing.T) (repo, base, landing, feature string) {
	t.Helper()
	repo = initCLIGitRepo(t)
	base = gitOutputReleaseTest(t, repo, "branch", "--show-current")
	original := gitOutputReleaseTest(t, repo, "rev-parse", "HEAD")
	for _, remote := range []string{"origin", "upstream"} {
		bare := filepath.Join(t.TempDir(), remote+".git")
		gitCLI(t, repo, "clone", "--bare", repo, bare)
		gitCLI(t, repo, "remote", "add", remote, bare)
		gitCLI(t, repo, "fetch", remote)
	}
	gitCLI(t, repo, "switch", "-c", "feature")
	writeFile(t, filepath.Join(repo, "feature.txt"), "feature\n")
	gitCLI(t, repo, "add", "feature.txt")
	gitCLI(t, repo, "commit", "-m", "feat: feature checkpoint")
	feature = gitOutputReleaseTest(t, repo, "rev-parse", "HEAD")
	gitCLI(t, repo, "switch", base)
	gitCLI(t, repo, "merge", "--squash", "feature")
	gitCLI(t, repo, "commit", "-m", "feat: squash landing")
	landing = gitOutputReleaseTest(t, repo, "rev-parse", "HEAD")
	gitCLI(t, repo, "push", "upstream", base)
	gitCLI(t, repo, "switch", "feature")
	gitCLI(t, repo, "update-ref", "refs/heads/"+base, original)
	return repo, base, landing, feature
}
