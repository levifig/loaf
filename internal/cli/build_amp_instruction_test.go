package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNativeAmpInstructionFallbackHasNoDanglingResources(t *testing.T) {
	root := setupIsolatedRepositoryBuildRoot(t)
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build", "--target", "amp"}); err != nil {
		t.Fatalf("build Amp: %v\n%s", err, stdout.String())
	}
	pluginPath := filepath.Join(root, "dist", "amp", ".amp", "plugins", "loaf.js")
	maps := readAmpInstructionFallbackMaps(t, pluginPath)
	hooks, err := readNativeBuildHooks(filepath.Join(root, "config", "hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	instructionCount := 0
	remainingCount := 0
	for _, hook := range hooks {
		if hook.typeName == "prompt" || (hook.section != "pre-tool" && hook.section != "post-tool") {
			continue
		}
		entries := maps[hook.section][hook.matcher]
		var found *nativeAmpHookEntry
		for i := range entries {
			if entries[i].ID == hook.id {
				found = &entries[i]
				break
			}
		}
		if hook.instruction != "" {
			instructionCount++
			if found != nil {
				t.Errorf("unsupported Amp instruction %s remains projected", hook.id)
			}
			continue
		}
		if hook.id == "detect-linear-magic" {
			continue
		} // Existing unsupported root-session identity boundary.
		remainingCount++
		if found == nil {
			t.Errorf("supported Amp hook %s missing", hook.id)
			continue
		}
		want := nativeAmpHookEntryFor(hook)
		if hook.id == "ephemeral-provenance" {
			want.Command = "loaf check --hook ephemeral-provenance"
		}
		if !reflect.DeepEqual(*found, want) {
			t.Errorf("Amp hook %s = %#v, want unchanged registry semantics %#v", hook.id, *found, want)
		}
	}
	if instructionCount != 3 {
		t.Fatalf("registry instruction coverage = %d, want all three pre/post checklists", instructionCount)
	}
	actualCount := 0
	for _, groups := range maps {
		for _, entries := range groups {
			actualCount += len(entries)
			for _, entry := range entries {
				if err := ampInstructionFallbackResourceReferences(filepath.Dir(pluginPath), entry); err != nil {
					t.Error(err)
				}
			}
		}
	}
	if actualCount != remainingCount {
		t.Errorf("Amp projected hook count = %d, want %d supported non-instruction hooks", actualCount, remainingCount)
	}
}

func TestNativeAmpInstructionResourceInventoryDetectsMissingFiles(t *testing.T) {
	root := t.TempDir()
	for _, entry := range []nativeAmpHookEntry{
		{ID: "instruction", Command: `cat "$LOAF_PLUGIN_DIR/hooks/instructions/missing.md"`},
		{ID: "script", Script: "post-tool/missing.sh"},
	} {
		if err := ampInstructionFallbackResourceReferences(root, entry); err == nil {
			t.Fatalf("missing resource in %#v passed inventory", entry)
		}
	}
	mkdirAll(t, filepath.Join(root, "hooks", "instructions"))
	writeFile(t, filepath.Join(root, "hooks", "instructions", "present.md"), "Present.\n")
	if err := ampInstructionFallbackResourceReferences(root, nativeAmpHookEntry{ID: "present", Command: `cat "$LOAF_PLUGIN_DIR/hooks/instructions/present.md"`}); err != nil {
		t.Fatal(err)
	}
}

func readAmpInstructionFallbackMaps(t *testing.T, path string) map[string]map[string][]nativeAmpHookEntry {
	t.Helper()
	body := readBuildFileString(t, path)
	maps := make(map[string]map[string][]nativeAmpHookEntry)
	for section, name := range map[string]string{"pre-tool": "preToolHooks", "post-tool": "postToolHooks"} {
		marker := "const " + name + " = "
		start := strings.Index(body, marker)
		if start < 0 {
			t.Fatalf("%s missing %s", path, name)
		}
		rest := body[start+len(marker):]
		end := strings.Index(rest, ";\n")
		if end < 0 {
			t.Fatalf("%s missing %s delimiter", path, name)
		}
		var groups map[string][]nativeAmpHookEntry
		if err := json.Unmarshal([]byte(rest[:end]), &groups); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		maps[section] = groups
	}
	return maps
}

func ampInstructionFallbackResourceReferences(pluginDir string, entry nativeAmpHookEntry) error {
	var paths []string
	if entry.Script != "" {
		paths = append(paths, entry.Script)
	}
	for _, match := range regexp.MustCompile(`\$LOAF_PLUGIN_DIR/hooks/([^"'\s]+)`).FindAllStringSubmatch(entry.Command, -1) {
		paths = append(paths, match[1])
	}
	for _, path := range paths {
		info, err := os.Stat(filepath.Join(pluginDir, "hooks", filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("Amp hook %s references unavailable resource %s: %w", entry.ID, path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Amp hook %s resource %s is not a regular file", entry.ID, path)
		}
	}
	return nil
}
