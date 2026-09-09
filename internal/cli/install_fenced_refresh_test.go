package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

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
