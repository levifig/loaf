package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedImplementationPrerequisiteReachesFreshAndExistingTargets(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "fresh"
		if existing {
			name = "upgrade-preserves-custom-instructions"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			mkdirAll(t, filepath.Join(root, ".claude"))
			prefix, suffix := "# User rules\n\nKeep my chosen toolchain.\n \t\n", "\n\n# User tail\nDo not deploy.\n\n"
			paths := []string{filepath.Join(root, "AGENTS.md"), filepath.Join(root, ".claude", "CLAUDE.md")}
			if existing {
				for _, path := range paths {
					writeInstallFile(t, path, prefix+"<!-- loaf:managed:start -->\nPrevious Loaf guidance\n"+fencedEndMarker+suffix)
				}
			}
			targets := []string{"codex", "claude-code", "cursor", "opencode", "amp"}
			results, err := installFencedSectionsForTargets(targets, root, "0.5.0", existing)
			if err != nil {
				t.Fatal(err)
			}
			wantAction := "created"
			if existing {
				wantAction = "updated"
			}
			for _, target := range []string{"codex", "claude-code"} {
				if results[target].Action != wantAction {
					t.Fatalf("%s action = %q, want %q", target, results[target].Action, wantAction)
				}
			}
			installed := map[string]string{}
			for _, path := range paths {
				body := string(readFileBytes(t, path))
				section, ok := findFencedSectionRange(body)
				if !ok || !strings.Contains(body[section.start:section.end], "**Before Implementation:**") {
					t.Fatalf("%s lacks the implementation prerequisite in its managed section", path)
				}
				if existing && (body[:section.start] != prefix || body[section.end:] != suffix) {
					t.Fatalf("refresh changed custom instructions in %s", path)
				}
				installed[path] = body
			}
			results, err = installFencedSectionsForTargets(targets, root, "0.5.0", true)
			if err != nil {
				t.Fatal(err)
			}
			for _, target := range targets {
				if results[target].Action != "skipped" {
					t.Fatalf("repeat %s action = %q, want skipped", target, results[target].Action)
				}
			}
			for path, body := range installed {
				assertInstallFile(t, path, body)
			}
		})
	}
}

func TestManagedSectionRefreshOwnsOnlyMarkedBytes(t *testing.T) {
	for _, header := range []string{
		"<!-- loaf:managed:start -->",
		"<!-- loaf:managed:start sha256=" + strings.Repeat("a", 64) + " -->",
		"<!-- loaf:managed:start v1.2.3 sha256=" + strings.Repeat("b", 64) + " -->",
	} {
		t.Run(header, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "AGENTS.md")
			before, after := "# User text\n \t\n", "\n\n \t# User tail\n\n"
			seed := before + header + "\nuser edits inside the managed section\n" + fencedEndMarker + after
			writeInstallFile(t, path, seed)
			if action, detail := planFencedSection(path, "1.2.3"); action != "updated" {
				t.Fatalf("plan = %s: %s", action, detail)
			}
			assertInstallFile(t, path, seed)
			result, err := installFencedSection(path, "1.2.3", true)
			if err != nil || result.Action != "updated" {
				t.Fatalf("refresh = %#v, %v", result, err)
			}
			want := before + generateFencedContent() + after
			assertInstallFile(t, path, want)
			if !strings.HasPrefix(generateFencedContent(), "<!-- loaf:managed:start -->\n") {
				t.Fatal("generated section must use a plain start marker")
			}
			if action, detail := planFencedSection(path, "2.0.0"); action != "skipped" {
				t.Fatalf("repeat plan = %s: %s", action, detail)
			}
			result, err = installFencedSection(path, "2.0.0", true)
			if err != nil || result.Action != "skipped" {
				t.Fatalf("repeat refresh = %#v, %v", result, err)
			}
			assertInstallFile(t, path, want)
		})
	}
}
