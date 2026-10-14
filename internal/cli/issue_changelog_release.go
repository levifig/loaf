package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strings"
)

// Pure release bumps must keep Loaf's checked-in version stamps and ownership
// digests synchronized. This closed set permits those mechanical edits only;
// being under dist/ or plugins/ is never a general release exemption.
func validateReleaseVersionOutput(root, base, head, file, oldVersion, newVersion string) error {
	before, err := readChangelogCandidateFile(root, base, file)
	if err != nil {
		return fmt.Errorf("release-only cannot add/delete files: %s", file)
	}
	after, err := readChangelogCandidateFile(root, head, file)
	if err != nil {
		return err
	}
	prefix := ""
	for _, candidate := range []string{"dist/amp/", "dist/codex/", "dist/cursor/", "dist/opencode/", "plugins/loaf/"} {
		if strings.HasPrefix(file, candidate) {
			prefix = candidate
			break
		}
	}
	rel := strings.TrimPrefix(file, prefix)
	if prefix != "" && (strings.HasPrefix(rel, "skills/") || strings.HasPrefix(rel, "agents/") || (prefix == "dist/opencode/" && strings.HasPrefix(rel, "commands/"))) && strings.HasSuffix(rel, ".md") {
		end := strings.Index(strings.TrimPrefix(before, "---\n"), "\n---\n")
		if strings.HasPrefix(before, "---\n") && end >= 0 {
			end += 5
			stamp := "\nversion: " + oldVersion + "\n"
			if strings.Count(before[:end], stamp) == 1 && after == strings.Replace(before[:end], stamp, "\nversion: "+newVersion+"\n", 1)+before[end:] {
				return nil
			}
		}
		footer := "\n---\nversion: " + oldVersion + "\n"
		if strings.HasPrefix(rel, "agents/") && strings.HasSuffix(before, footer) && after == strings.TrimSuffix(before, footer)+"\n---\nversion: "+newVersion+"\n" {
			return nil
		}
	}
	if file == "dist/amp/.amp/plugins/loaf.js" || file == "dist/opencode/plugins/hooks.js" {
		stamp := " * @version " + oldVersion + "\n"
		if strings.HasPrefix(before, "/**\n") && strings.Index(before, stamp) >= 0 && strings.Index(before, stamp) < 200 && after == strings.Replace(before, stamp, " * @version "+newVersion+"\n", 1) {
			return nil
		}
	}
	var old, new map[string]any
	if json.Unmarshal([]byte(before), &old) != nil || json.Unmarshal([]byte(after), &new) != nil || old == nil || new == nil {
		return fmt.Errorf("release-only includes non-mechanical output: %s", file)
	}
	switch {
	case file == ".claude-plugin/marketplace.json":
		metadata, ok := old["metadata"].(map[string]any)
		if !ok || metadata["version"] != oldVersion {
			break
		}
		metadata["version"] = newVersion
		plugins, ok := old["plugins"].([]any)
		if !ok {
			break
		}
		for _, plugin := range plugins {
			item, ok := plugin.(map[string]any)
			if !ok || item["version"] != oldVersion {
				return fmt.Errorf("release-only marketplace has unexpected version metadata")
			}
			item["version"] = newVersion
		}
		if reflect.DeepEqual(old, new) {
			return nil
		}
	case file == "plugins/loaf/.claude-plugin/plugin.json" || file == "plugins/loaf/package.json":
		if old["version"] == oldVersion {
			old["version"] = newVersion
			if reflect.DeepEqual(old, new) {
				return nil
			}
		}
	case prefix != "" && rel == ".loaf-hook-catalog.json":
		if old["package_version"] == oldVersion {
			old["package_version"] = newVersion
			if reflect.DeepEqual(old, new) {
				return nil
			}
		}
	case prefix != "" && rel == ".loaf-target-manifest.json":
		if old["package_version"] != oldVersion {
			break
		}
		old["package_version"] = newVersion
		artifacts, ok := old["artifacts"].([]any)
		if !ok {
			break
		}
		for _, artifact := range artifacts {
			item, ok := artifact.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid release-only target manifest")
			}
			source, _ := item["source_path"].(string)
			if source == "" {
				continue
			}
			if path.IsAbs(source) || path.Clean(source) != source || strings.HasPrefix(source, "../") {
				return fmt.Errorf("unsafe release-only manifest source")
			}
			oldSource, err := readChangelogCandidateFile(root, base, prefix+source)
			if err != nil {
				return err
			}
			newSource, err := readChangelogCandidateFile(root, head, prefix+source)
			if err != nil {
				return err
			}
			oldDigest, newDigest := sha256.Sum256([]byte(oldSource)), sha256.Sum256([]byte(newSource))
			if item["sha256"] != hex.EncodeToString(oldDigest[:]) {
				return fmt.Errorf("release-only base manifest digest mismatch: %s", file)
			}
			item["sha256"] = hex.EncodeToString(newDigest[:])
		}
		if reflect.DeepEqual(old, new) {
			return nil
		}
	}
	return fmt.Errorf("release-only includes implementation or non-mechanical output: %s", file)
}
