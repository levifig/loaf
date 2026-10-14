package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssueChangelogReleaseWithRealGeneratedOutputs(t *testing.T) {
	source := testRepositoryRoot(t)
	repo := initCLIGitRepo(t)
	files, err := changelogGit(source, "ls-files", "-z")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range strings.Split(files, "\x00") {
		if file == "" {
			continue
		}
		info, err := os.Lstat(filepath.Join(source, file))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filepath.Join(source, file))
			if err != nil {
				t.Fatal(err)
			}
			mkdirAll(t, filepath.Dir(filepath.Join(repo, file)))
			if err := os.Symlink(target, filepath.Join(repo, file)); err != nil {
				t.Fatal(err)
			}
			continue
		}
		body, err := os.ReadFile(filepath.Join(source, file))
		if err != nil {
			t.Fatal(err)
		}
		mkdirAll(t, filepath.Dir(filepath.Join(repo, file)))
		if err := os.WriteFile(filepath.Join(repo, file), body, info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	// Normalize the fixture's baseline through the actual content builder, then
	// change only package version and perform the real generated-content path.
	if err := runBuildContent(repo, buildOptions{}, io.Discard); err != nil {
		t.Fatal(err)
	}
	before := coverageChangelog("- Require linked changelog coverage " + coverageLink + ".")
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), before)
	coverageCommit(t, repo)
	gitCLI(t, repo, "switch", "-c", "release-fixture")
	packagePath := filepath.Join(repo, "package.json")
	packageBody, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageBody, &version); err != nil {
		t.Fatal(err)
	}
	oldVersion := version.Version
	newVersion := oldVersion + "-changelog-fixture"
	if strings.Contains(oldVersion, "-") {
		newVersion = oldVersion + ".changelog-fixture"
	}
	writeFile(t, packagePath, strings.Replace(string(packageBody), `"version": "`+oldVersion+`"`, `"version": "`+newVersion+`"`, 1))
	marketplacePath := filepath.Join(repo, ".claude-plugin", "marketplace.json")
	marketplace, err := os.ReadFile(marketplacePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, marketplacePath, strings.ReplaceAll(string(marketplace), `"version": "`+oldVersion+`"`, `"version": "`+newVersion+`"`))
	parts, err := splitIssueChangelog(before)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), parts.prefix+"\n## ["+newVersion+"] - 2026-10-14\n\n"+strings.TrimSpace(parts.unreleased)+"\n\n"+parts.released)
	if err := runBuildContent(repo, buildOptions{}, io.Discard); err != nil {
		t.Fatal(err)
	}
	coverageCommit(t, repo)
	request := issueChangelogRequest{coverageRepository, "main", "HEAD", implementedIssues{ReleaseOnly: true}}
	if err := checkIssueChangelog(repo, request); err != nil {
		t.Fatalf("real mechanical generated release rejected: %v", err)
	}
	// A generated location does not exempt a behavior edit, even when the
	// version stamp and its ownership manifest are otherwise legitimate.
	plugin := filepath.Join(repo, "dist", "amp", ".amp", "plugins", "loaf.js")
	file, err := os.OpenFile(plugin, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\nthrow new Error('new behavior');\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	coverageCommit(t, repo)
	if err := checkIssueChangelog(repo, request); err == nil {
		t.Fatal("arbitrary generated code accepted as release-only")
	}
}
