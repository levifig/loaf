package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	fencedStartMarker = "<!-- loaf:managed:start"
	fencedEndMarker   = "<!-- loaf:managed:end -->"
	fencedWarning     = "<!-- Maintained by loaf install/upgrade; edits inside this section are overwritten. Put custom instructions outside it. -->"
)

var fencedTargetFiles = map[string]string{
	"claude-code": ".claude/CLAUDE.md",
	"cursor":      "AGENTS.md",
	"codex":       "AGENTS.md",
	"opencode":    "AGENTS.md",
	"amp":         "AGENTS.md",
}

var fencedVersionField = regexp.MustCompile(`^v([\d.]+(?:-[-\w.]+)?)$`)

type fencedSectionRange struct {
	start           int
	end             int
	malformedHeader bool
	bodyStart       int
}

type fencedInstallResult struct {
	Action  string
	Version string
	Error   string
}

func installFencedSection(targetFile string, version string, upgrade bool) (fencedInstallResult, error) {
	canonicalTarget, err := canonicalFenceWritePath(targetFile)
	if err != nil {
		return fencedInstallResult{}, err
	}
	targetFile = canonicalTarget
	if version == "" {
		version = "0.0.0"
	}
	body, err := readRegularFile(targetFile, projectFileReadLimit)
	fileExisted := err == nil
	if err != nil && !os.IsNotExist(err) {
		return fencedInstallResult{}, refuseProjectFileRead(err)
	}
	content := string(body)
	if err := validateFencedStructure(content); err != nil {
		return fencedInstallResult{}, err
	}
	section, hasSection := findFencedSectionRange(content)
	newContent := generateFencedContent()

	switch {
	case hasSection:
		if section.malformedHeader {
			return fencedInstallResult{}, fmt.Errorf("managed Loaf section in %s has a malformed start marker; refusing to overwrite", targetFile)
		}
		if content[section.start:section.end] == newContent {
			return fencedInstallResult{Action: "skipped", Version: version}, nil
		}
		// The markers delimit Loaf-owned bytes. Preserve everything outside
		// them exactly, including user whitespace.
		updated := content[:section.start] + newContent + content[section.end:]
		if err := writeFileAtomically(targetFile, []byte(updated), fencedWriteMode(targetFile, true)); err != nil {
			return fencedInstallResult{}, err
		}
		return fencedInstallResult{Action: "updated", Version: version}, nil
	case fileExisted:
		trimmed := strings.TrimRight(content, " \t\r\n")
		updated := trimmed
		if updated != "" {
			updated += "\n\n"
		}
		updated += newContent + "\n"
		if err := writeFileAtomically(targetFile, []byte(updated), fencedWriteMode(targetFile, true)); err != nil {
			return fencedInstallResult{}, err
		}
		return fencedInstallResult{Action: "appended", Version: version}, nil
	default:
		if err := writeFileAtomically(targetFile, []byte(newContent+"\n"), 0o644); err != nil {
			return fencedInstallResult{}, err
		}
		return fencedInstallResult{Action: "created", Version: version}, nil
	}
}

func fencedWriteMode(path string, existed bool) os.FileMode {
	if existed {
		if info, err := os.Stat(path); err == nil {
			return info.Mode().Perm()
		}
	}
	return 0o644
}

func canonicalFenceWritePath(path string) (string, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return path, nil
	}
	return filepath.EvalSymlinks(path)
}

func validateFencedStructure(content string) error {
	starts, ends := strings.Count(content, fencedStartMarker), strings.Count(content, fencedEndMarker)
	if starts == 0 && ends == 0 {
		return nil
	}
	if starts != 1 || ends != 1 {
		return fmt.Errorf("managed Loaf section has invalid fence structure; refusing to overwrite")
	}
	start, end := strings.Index(content, fencedStartMarker), strings.Index(content, fencedEndMarker)
	if start > end {
		return fmt.Errorf("managed Loaf section has invalid fence structure; refusing to overwrite")
	}
	return nil
}

// installFencedSectionsForTargets writes each target's managed section and
// stops the batch at the first failure, returning the results gathered so far
// alongside the error so the caller can still report what happened.
//
// Stopping is the point. A refusal is a statement about this project, not about
// one harness: the targets share files — cursor, codex, opencode, and amp all
// name AGENTS.md — and a fenced write is refused precisely when the file is not
// currently Loaf's to manage. Carrying on past that let a refused write to
// AGENTS.md be followed by creating .claude/CLAUDE.md for the next target in
// the list, which is the half-deployed project both callers already stop for
// once they see the failure.
func installFencedSectionsForTargets(targets []string, projectRoot string, version string, upgrade bool) (map[string]fencedInstallResult, error) {
	results := map[string]fencedInstallResult{}
	writtenPaths := map[string]string{}
	for _, target := range targets {
		relPath, ok := fencedTargetFiles[target]
		if !ok {
			err := fmt.Errorf("Unknown target: %s", target)
			results[target] = fencedInstallResult{Action: "error", Error: err.Error()}
			return results, err
		}
		targetFile := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		canonicalBefore := canonicalInstallPath(targetFile)
		if writtenVersion, ok := writtenPaths[canonicalBefore]; ok {
			results[target] = fencedInstallResult{Action: "skipped", Version: writtenVersion}
			continue
		}
		result, err := installFencedSection(targetFile, version, upgrade)
		if err != nil {
			results[target] = fencedInstallResult{Action: "error", Error: err.Error()}
			return results, err
		}
		results[target] = result
		writtenPaths[canonicalInstallPath(targetFile)] = result.Version
	}
	return results, nil
}

func findFencedSectionRange(content string) (fencedSectionRange, bool) {
	start := strings.Index(content, fencedStartMarker)
	if start < 0 {
		return fencedSectionRange{}, false
	}
	endStart := strings.Index(content[start:], fencedEndMarker)
	if endStart < 0 {
		return fencedSectionRange{}, false
	}
	end := start + endStart + len(fencedEndMarker)
	lineEnd := strings.IndexByte(content[start:], '\n')
	if lineEnd < 0 {
		lineEnd = len(content) - start
	}
	startLineEnd := strings.Index(content[start:start+lineEnd], "-->")
	if startLineEnd >= 0 && start+startLineEnd < start+endStart {
		_, _, valid := parseFencedStartHeader(content[start : start+startLineEnd+3])
		if !valid {
			return fencedSectionRange{start: start, end: end, malformedHeader: true}, true
		}
		bodyStart := start + startLineEnd + 3
		if bodyStart < end && content[bodyStart] == '\n' {
			bodyStart++
		}
		return fencedSectionRange{start: start, end: end, bodyStart: bodyStart}, true
	}
	return fencedSectionRange{start: start, end: end, malformedHeader: true}, true
}

func parseFencedStartHeader(line string) (string, string, bool) {
	if !strings.HasPrefix(line, fencedStartMarker) || !strings.HasSuffix(line, "-->") {
		return "", "", false
	}
	fields := strings.Fields(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, fencedStartMarker), "-->")))
	if len(fields) == 0 {
		return "", "", true
	}
	if len(fields) > 2 {
		return "", "", false
	}
	const shaPrefix = "sha256="
	parseSHA := func(field string) (string, bool) {
		if !strings.HasPrefix(field, shaPrefix) || len(field) != len(shaPrefix)+64 || !isCanonicalFenceSHA256(field[len(shaPrefix):]) {
			return "", false
		}
		return field[len(shaPrefix):], true
	}

	if len(fields) == 1 {
		// Legacy: sha256=<hex> alone. Parsed for migration, not ownership.
		if sha, ok := parseSHA(fields[0]); ok {
			return "", sha, true
		}
		// Legacy: v<version> alone.
		match := fencedVersionField.FindStringSubmatch(fields[0])
		if len(match) != 2 {
			return "", "", false
		}
		return match[1], "", true
	}

	// Legacy: v<version> sha256=<hex>.
	match := fencedVersionField.FindStringSubmatch(fields[0])
	if len(match) != 2 {
		return "", "", false
	}
	sha, ok := parseSHA(fields[1])
	if !ok {
		return "", "", false
	}
	return match[1], sha, true
}

func isCanonicalFenceSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func generateFencedContent() string {
	body := strings.Join([]string{
		fencedWarning,
		"## Loaf Framework",
		"",
		"**Journal Entry Types:**",
		"- `decision(scope)`: Key decisions with rationale",
		"- `discover(scope)`: Something learned",
		"- `block(scope)` / `unblock(scope)`: Blockers and resolutions",
		"- `spark(scope)`: Ideas to promote via `/idea`",
		"- `todo(scope)`: Action items to file in the configured native tracker",
		"",
		"**Tracker Authority:**",
		"Shared work identity, definition, definition of done, status, hierarchy, assignment, and collaboration live only in the configured native tracker. Use the selected `project-management/v1` provider skill through a connection already exposed and authenticated by the harness.",
		"Loaf never configures provider authentication, calls a provider API itself, proxies tracker traffic, or keeps an ongoing local-to-tracker mapping. Local-to-tracker synchronization does not exist. A legacy local project may be moved once through an explicit, agentic, verified migration; ongoing work then remains tracker-native.",
		"",
		"**Before Implementation:**",
		"Before substantive edits, including ordinary fixes and regression tests, use the configured project-management provider to search for and reuse or create a native issue, then verify its scope, completion criteria, and selected state. Honor an existing explicit implementation request without asking for selection again; issue creation or Backlog alone is not selection. Load the loaf-reference skill's Flow Semantics for the canonical rule and narrow exceptions.",
		"Read-only discovery and purely incidental corrections with no behavior, meaning, requirement, or scope change may proceed first; a one-line bug or security fix is substantive. If the tracker or required readback is unavailable, report the blocker and continue only permitted discovery unless the user explicitly overrides this prerequisite. Never substitute a local shadow tracker or configure authentication.",
		"",
		"**CLI Commands:**",
		"- `loaf journal log/recent/search/context` - Project journal",
		"- `loaf check` - Run enforcement hooks",
		"- `loaf kb` - Local project knowledge",
		"- `loaf issue` and its Linear push/pull/reconcile commands are frozen migration compatibility only; never use them as ongoing work authority or synchronization",
		"",
		"**Journal Discipline:**",
		"Before completing any response that includes edits, commits, or significant decisions, log journal entries using `loaf journal log \"type(scope): description\"`. Entry types: `decision`, `discover`, `wrap`. Do not defer journaling - log before responding.",
		"In Codex Auto mode, when the user explicitly installed the managed basic-command policy, use the PATH `loaf` command for classified leaves, including `loaf journal log --execpolicy-safe` for journal writes. Do not substitute an absolute executable pin or a shell/environment wrapper. The policy authorizes only explicitly classified basic Loaf command leaves and does not grant unclassified/operator commands, a bare Loaf namespace, or general filesystem access. Global instructions remain user-owned; no Loaf block in `CODEX_HOME/AGENTS.md` is required. Other harness adapters are not implied.",
		"",
		"See the Loaf `orchestration` skill for full details.",
		fencedEndMarker,
	}, "\n")
	return "<!-- loaf:managed:start -->\n" + body
}

func fencedContentFingerprint(content string) string {
	section, ok := findFencedSectionRange(content)
	if !ok {
		return ""
	}
	return sha256Hex(content[section.bodyStart:section.end])
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func canonicalInstallPath(path string) string {
	if realPath, err := filepath.EvalSymlinks(path); err == nil {
		if abs, absErr := filepath.Abs(realPath); absErr == nil {
			return abs
		}
		return realPath
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return filepath.Clean(path)
}
