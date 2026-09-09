package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedSkillScriptRetirementPreservesUserChanges(t *testing.T) {
	for _, modified := range []bool{false, true} {
		t.Run(map[bool]string{false: "owned", true: "modified"}[modified], func(t *testing.T) {
			root := t.TempDir()
			source, destination := filepath.Join(root, "source", "skills"), filepath.Join(root, "installed", "skills")
			script := filepath.Join("foundations", "scripts", "validate-adr.py")
			mkdirAll(t, filepath.Join(source, "foundations", "scripts"))
			writeFile(t, filepath.Join(source, "foundations", "SKILL.md"), "# Foundations\n")
			writeFile(t, filepath.Join(source, script), "# Former validator\n")
			if err := syncManagedSkillsDirIfExists(source, destination); err != nil {
				t.Fatal(err)
			}
			mkdirAll(t, filepath.Join(destination, "foreign", "scripts"))
			writeFile(t, filepath.Join(destination, "foreign", "scripts", "validate-adr.py"), "# User's validator\n")
			if modified {
				writeFile(t, filepath.Join(destination, script), "# User's changes\n")
			}
			if err := os.Remove(filepath.Join(source, script)); err != nil {
				t.Fatal(err)
			}
			err := syncManagedSkillsDirIfExists(source, destination)
			if modified {
				if err == nil || !strings.Contains(err.Error(), "modified") {
					t.Fatalf("modified script retirement: %v", err)
				}
				if got := readBuildFileString(t, filepath.Join(destination, script)); got != "# User's changes\n" {
					t.Fatalf("modified script changed: %q", got)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(destination, script)); !os.IsNotExist(err) {
					t.Fatalf("owned script retained: %v", err)
				}
			}
			if got := readBuildFileString(t, filepath.Join(destination, "foreign", "scripts", "validate-adr.py")); got != "# User's validator\n" {
				t.Fatalf("foreign script changed: %q", got)
			}
		})
	}
}

func TestManagedHookScriptRetirementPreservesUserChanges(t *testing.T) {
	for _, modified := range []bool{false, true} {
		t.Run(map[bool]string{false: "owned", true: "modified"}[modified], func(t *testing.T) {
			root := realpath(t, t.TempDir())
			dist, config := filepath.Join(root, "dist", "cursor"), filepath.Join(root, "cursor")
			script := "hooks/post-tool/kb-staleness-nudge.sh"
			body := "#!/bin/sh\necho former nudge\n"
			writeInstallFile(t, filepath.Join(dist, script), body)
			writeTestTargetAdapterManifest(t, dist, "cursor", []map[string]string{{
				"id": "hook-file:" + script, "kind": "hook-file", "source_path": script, "destination": script, "sha256": sha256Hex(body),
			}})
			options := targetInstallOptions{Target: "cursor", DistDir: dist, ConfigDir: config, HomeDir: filepath.Join(root, "home"), Version: "9.8.7-test.1"}
			if err := syncTargetAdapterManifest(options); err != nil {
				t.Fatal(err)
			}
			writeInstallFile(t, filepath.Join(config, "hooks", "user.sh"), "# User hook\n")
			if modified {
				writeInstallFile(t, filepath.Join(config, script), "# User's changes\n")
			}
			if err := os.Remove(filepath.Join(dist, script)); err != nil {
				t.Fatal(err)
			}
			writeTestTargetAdapterManifest(t, dist, "cursor", nil)
			err := syncTargetAdapterManifest(options)
			if modified {
				if err == nil || !strings.Contains(err.Error(), "modified") {
					t.Fatalf("modified hook retirement: %v", err)
				}
				assertInstallFile(t, filepath.Join(config, script), "# User's changes\n")
			} else {
				if err != nil {
					t.Fatal(err)
				}
				assertInstallPathMissing(t, filepath.Join(config, script))
			}
			assertInstallFile(t, filepath.Join(config, "hooks", "user.sh"), "# User hook\n")
		})
	}
}
