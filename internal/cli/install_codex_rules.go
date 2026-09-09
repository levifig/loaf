package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	codexJournalRuleRelativePath         = "loaf.rules"
	codexJournalRuleTemplateRelativePath = "loaf.rules.tmpl"
	codexJournalGuidanceRelativePath     = "AGENTS.md"
	codexJournalRuleManifest             = ".loaf-managed-rules.json"
	codexJournalExecutablePlaceholder    = "{{LOAF_EXECUTABLE}}"
	codexPathLoafCommandName             = "loaf"
	codexJournalHookMatcher              = "startup|resume|clear|compact"
	codexJournalHookCommandSuffix        = " journal context --from-hook --codex-hook"
	codexJournalHookCommandTemplate      = codexPathLoafCommandName + codexJournalHookCommandSuffix
	codexJournalGuidanceStart            = "<!-- loaf:managed:codex-basic-commands:start -->"
	codexJournalGuidanceEnd              = "<!-- loaf:managed:codex-basic-commands:end -->"
	codexLegacyGuidanceStart             = "<!-- loaf:managed:codex-auto-journal:start -->"
	codexLegacyGuidanceEnd               = "<!-- loaf:managed:codex-auto-journal:end -->"
	codexBasicRulesPlaceholder           = "{{LOAF_BASIC_RULES}}"
)

type codexManagedRuleManifest struct {
	Version int                    `json:"version"`
	Files   []codexManagedRuleFile `json:"files"`
}

type codexManagedRuleFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type codexRuleInstallOperations struct {
	lookPath       func(string) (string, error)
	forbiddenRoots []string
}

func installCodexJournalRule(options targetInstallOptions, codexHome string) error {
	return installCodexJournalRuleWithOperations(options, codexHome, nil)
}

func installCodexJournalRuleWithOperations(options targetInstallOptions, codexHome string, operations *codexRuleInstallOperations) error {
	options.CodexHome = codexHome
	if operations != nil {
		options.CodexRuleOperations = operations
	}
	decisions, err := planCodexJournalRule(options)
	if err != nil {
		return err
	}
	for _, decision := range decisions {
		if decision.Action == planActionConflict {
			return fmt.Errorf("%s: %s", decision.ID, decision.Detail)
		}
	}
	if !scopedPolicyDecisionsNeedApply(decisions) {
		return nil
	}
	txn := newScopedTxn()
	if err := txn.backup(codexPolicyOwnershipManifestPath(options)); err != nil {
		return err
	}
	for _, decision := range decisions {
		if scopedPolicyActionWrites(decision.Action) {
			if err := txn.backup(decision.Destination); err != nil {
				return err
			}
		}
	}
	if err := publishSelectedCodexPolicyArtifacts(options, decisions); err != nil {
		if rollbackErr := txn.rollback(); rollbackErr != nil {
			return fmt.Errorf("%w; rollback: %v", err, rollbackErr)
		}
		return err
	}
	return txn.cleanup()
}

func detectLegacyCodexJournalCapability(rulePath string) (bool, error) {
	rule, ruleExists, err := readOptionalInstallFile(rulePath, "installed Codex journal rule")
	if err != nil {
		return false, err
	}
	if ruleExists && strings.Count(string(rule), "prefix_rule(") == 1 && strings.Contains(string(rule), `"journal", "log", "--execpolicy-safe"`) {
		return true, nil
	}
	return false, nil
}

// readOptionalInstallFile reads a file under the harness's own config home —
// the installed rule, and legacy AGENTS.md guidance eligible for retirement.
// Absence is a fact the callers act on; anything else is a refusal, because
// every one of them decides a rewrite of the same path from what comes back.
func readOptionalInstallFile(path string, description string) ([]byte, bool, error) {
	body, err := readRegularFile(path, projectFileReadLimit)
	if err == nil {
		return body, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("read %s: %w", description, refuseProjectFileRead(err))
}

func renderCodexJournalRule(template string) (string, error) {
	rendered := template
	if strings.Contains(rendered, codexBasicRulesPlaceholder) {
		rendered = strings.ReplaceAll(rendered, codexBasicRulesPlaceholder, renderCodexBasicRules())
	}
	if strings.Contains(rendered, codexJournalExecutablePlaceholder) {
		rendered = strings.ReplaceAll(rendered, codexJournalExecutablePlaceholder, strconv.Quote(codexPathLoafCommandName))
	}
	if rendered == template {
		return "", fmt.Errorf("Codex journal rule template is missing %s or %s", codexBasicRulesPlaceholder, codexJournalExecutablePlaceholder)
	}
	if strings.Contains(rendered, codexBasicRulesPlaceholder) || strings.Contains(rendered, codexJournalExecutablePlaceholder) {
		return "", fmt.Errorf("Codex journal rule template contains an unresolved placeholder")
	}
	return rendered, nil
}

func renderCodexBasicRules() string {
	var rendered strings.Builder
	for _, prefix := range BasicCommandAuthorityPrefixes() {
		rendered.WriteString("prefix_rule(\n    pattern = [")
		rendered.WriteString(strconv.Quote(codexPathLoafCommandName))
		for _, token := range prefix {
			rendered.WriteString(", ")
			rendered.WriteString(strconv.Quote(token))
		}
		rendered.WriteString("],\n    decision = \"allow\",\n)\n")
	}
	return rendered.String()
}

type codexJournalGuidanceRange struct{ start, end int }

func findCodexJournalGuidance(content string) (codexJournalGuidanceRange, bool) {
	startMarker := codexJournalGuidanceStart
	endMarker := codexJournalGuidanceEnd
	markerStart := strings.Index(content, startMarker)
	if markerStart < 0 {
		startMarker = codexLegacyGuidanceStart
		endMarker = codexLegacyGuidanceEnd
		markerStart = strings.Index(content, startMarker)
	}
	if markerStart < 0 {
		return codexJournalGuidanceRange{}, false
	}
	start := markerStart
	if start > 0 && content[start-1] == '\n' {
		start--
	}
	endStart := strings.Index(content[markerStart+len(startMarker):], endMarker)
	if endStart < 0 {
		return codexJournalGuidanceRange{}, false
	}
	end := markerStart + len(startMarker) + endStart + len(endMarker)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return codexJournalGuidanceRange{start: start, end: end}, true
}

func validateCodexJournalGuidanceStructure(content string) error {
	starts := strings.Count(content, codexJournalGuidanceStart) + strings.Count(content, codexLegacyGuidanceStart)
	ends := strings.Count(content, codexJournalGuidanceEnd) + strings.Count(content, codexLegacyGuidanceEnd)
	if starts == 0 && ends == 0 {
		return nil
	}
	if starts != 1 || ends != 1 {
		return fmt.Errorf("expected exactly one complete Loaf-managed block, found %d starts and %d ends", starts, ends)
	}
	startMarker := codexJournalGuidanceStart
	endMarker := codexJournalGuidanceEnd
	start := strings.Index(content, startMarker)
	if start < 0 {
		startMarker = codexLegacyGuidanceStart
		endMarker = codexLegacyGuidanceEnd
		start = strings.Index(content, startMarker)
	}
	end := strings.Index(content, endMarker)
	if start < 0 || end < start+len(startMarker) {
		return fmt.Errorf("Loaf-managed guidance end marker precedes its start marker")
	}
	return nil
}

func removeCodexJournalGuidance(content string, r codexJournalGuidanceRange) string {
	return content[:r.start] + content[r.end:]
}

func sha256Bytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (m codexManagedRuleManifest) ownedDigest(path string) (string, bool) {
	for _, file := range m.Files {
		if file.Path == path {
			return file.SHA256, true
		}
	}
	return "", false
}

func (m *codexManagedRuleManifest) set(path string, digest string) {
	for i := range m.Files {
		if m.Files[i].Path == path {
			m.Files[i].SHA256 = digest
			return
		}
	}
	m.Files = append(m.Files, codexManagedRuleFile{Path: path, SHA256: digest})
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
}

func (m *codexManagedRuleManifest) remove(path string) {
	files := m.Files[:0]
	for _, file := range m.Files {
		if file.Path != path {
			files = append(files, file)
		}
	}
	m.Files = files
}

func readCodexManagedRuleManifest(path string) (codexManagedRuleManifest, error) {
	body, err := readRegularFile(path, projectFileReadLimit)
	if err != nil {
		if os.IsNotExist(err) {
			return codexManagedRuleManifest{Version: 1}, nil
		}
		return codexManagedRuleManifest{}, fmt.Errorf("read Codex rule ownership manifest: %w", refuseProjectFileRead(err))
	}
	var manifest codexManagedRuleManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return codexManagedRuleManifest{}, fmt.Errorf("decode Codex rule ownership manifest: %w", err)
	}
	if manifest.Version != 1 {
		return codexManagedRuleManifest{}, fmt.Errorf("unsupported Codex rule ownership manifest version %d", manifest.Version)
	}
	seen := make(map[string]bool, len(manifest.Files))
	for _, file := range manifest.Files {
		if file.Path == "" || filepath.Base(file.Path) != file.Path || filepath.IsAbs(file.Path) || file.Path != filepath.Clean(file.Path) || file.SHA256 == "" {
			return codexManagedRuleManifest{}, fmt.Errorf("invalid Codex rule ownership manifest path or digest %q", file.Path)
		}
		if seen[file.Path] {
			return codexManagedRuleManifest{}, fmt.Errorf("duplicate Codex rule ownership manifest path %q", file.Path)
		}
		seen[file.Path] = true
		if _, err := hex.DecodeString(file.SHA256); err != nil || len(file.SHA256) != sha256.Size*2 {
			return codexManagedRuleManifest{}, fmt.Errorf("invalid Codex rule ownership digest for %q", file.Path)
		}
	}
	return manifest, nil
}

func scopedPolicyDecisionsNeedApply(decisions []artifactPlanDecision) bool {
	for _, decision := range decisions {
		if scopedPolicyActionWrites(decision.Action) {
			return true
		}
	}
	return false
}

func codexPolicyOwnershipManifestPath(options targetInstallOptions) string {
	return filepath.Join(effectiveCodexHome(options), "rules", codexJournalRuleManifest)
}

func publishSelectedCodexPolicyArtifacts(options targetInstallOptions, decisions []artifactPlanDecision) error {
	var selected []artifactPlanDecision
	for _, decision := range decisions {
		if !isCodexPolicyDecision(decision) || !scopedPolicyActionWrites(decision.Action) {
			continue
		}
		selected = append(selected, decision)
	}
	if len(selected) == 0 {
		return nil
	}
	manifestPath := codexPolicyOwnershipManifestPath(options)
	manifest, err := readCodexManagedRuleManifest(manifestPath)
	if err != nil {
		return err
	}
	for _, decision := range selected {
		if decision.Action == planActionRetire {
			if err := retireSelectedCodexPolicyArtifact(options, decision, &manifest); err != nil {
				return err
			}
			continue
		}
		if err := publishSelectedCodexPolicyArtifact(options, decision, &manifest); err != nil {
			return err
		}
	}
	if err := writeCodexManagedRuleManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("record Codex journal rule ownership: %w", err)
	}
	return nil
}

func publishSelectedCodexPolicyArtifact(options targetInstallOptions, decision artifactPlanDecision, manifest *codexManagedRuleManifest) error {
	path, err := resolveDecisionPath(options, decision)
	if err != nil {
		return err
	}
	live := ""
	if current, exists, err := readOptionalInstallFile(path, "selected Codex policy"); err != nil {
		return err
	} else if exists {
		live = string(current)
	}
	desired, err := desiredCodexPolicyContent(options, decision, live)
	if err != nil {
		return err
	}
	if err := writeFileAtomically(path, desired, 0o644); err != nil {
		return fmt.Errorf("publish selected Codex policy %s: %w", decision.ID, err)
	}
	switch decision.Kind {
	case "codex-rule":
		manifest.set(codexJournalRuleRelativePath, sha256Bytes(desired))
	case "codex-guidance":
		return fmt.Errorf("global Codex guidance is retired and cannot be published")
	default:
		return fmt.Errorf("unsupported Codex policy kind %q", decision.Kind)
	}
	return nil
}

// Read and validate the owned bytes again at apply time; a retirement plan is
// not authority to remove foreign or subsequently modified policy.
func readOwnedCodexPolicyForRetirement(path string, kind string, manifest codexManagedRuleManifest) ([]byte, bool, error) {
	name, description := codexJournalRuleRelativePath, "rule"
	switch kind {
	case "codex-rule":
	case "codex-guidance":
		name, description = codexJournalGuidanceRelativePath, "guidance block"
	default:
		return nil, false, fmt.Errorf("unsupported Codex policy kind %q", kind)
	}
	digest, owned := manifest.ownedDigest(name)
	if !owned {
		return nil, false, nil
	}
	current, exists, err := readOptionalInstallFile(path, "selected Codex policy retirement")
	if err != nil {
		return nil, true, err
	}
	if kind == "codex-rule" && !exists {
		return nil, true, nil
	}
	ownedContent := current
	if kind == "codex-guidance" {
		if err := validateCodexJournalGuidanceStructure(string(current)); err != nil {
			return nil, true, err
		}
		guidanceRange, found := findCodexJournalGuidance(string(current))
		if !found {
			return nil, true, fmt.Errorf("refusing to remove modified Loaf-owned Codex guidance block in %s", path)
		}
		ownedContent = current[guidanceRange.start:guidanceRange.end]
	}
	if sha256Bytes(ownedContent) != digest {
		return nil, true, fmt.Errorf("refusing to remove modified Loaf-owned Codex %s in %s", description, path)
	}
	return current, true, nil
}

func retireSelectedCodexPolicyArtifact(options targetInstallOptions, decision artifactPlanDecision, manifest *codexManagedRuleManifest) error {
	path, err := resolveDecisionPath(options, decision)
	if err != nil {
		return err
	}
	current, owned, err := readOwnedCodexPolicyForRetirement(path, decision.Kind, *manifest)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}
	switch decision.Kind {
	case "codex-rule":
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("retire selected Codex rule %s: %w", decision.ID, err)
		}
		manifest.remove(codexJournalRuleRelativePath)
	case "codex-guidance":
		text := string(current)
		if guidanceRange, ok := findCodexJournalGuidance(text); ok {
			if err := writeFileAtomically(path, []byte(removeCodexJournalGuidance(text, guidanceRange)), 0o644); err != nil {
				return fmt.Errorf("retire selected Codex guidance %s: %w", decision.ID, err)
			}
		}
		manifest.remove(codexJournalGuidanceRelativePath)
	default:
		return fmt.Errorf("unsupported Codex policy kind %q", decision.Kind)
	}
	return nil
}

func writeCodexManagedRuleManifest(path string, manifest codexManagedRuleManifest) error {
	if manifest.Version == 0 {
		manifest.Version = 1
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeFileAtomically(path, body, 0o644)
}

func writeFileAtomically(path string, body []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".loaf-install-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(body); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func validateCodexJournalExecutable(projectRoot string) error {
	_, err := trustedCodexJournalExecutable(projectRoot, nil)
	return err
}

// trustedCodexJournalExecutable resolves the loaf PATH entry for fail-closed
// install/upgrade of the Codex basic-command policy and for recording a
// trusted path that still recognizes older absolute-pin installs. Generated
// hooks and rules render the PATH command name `loaf`, not this
// absolute entrypoint. Validation still requires the PATH binary to exist
// outside forbidden roots, and both the entrypoint and canonical paths must
// be guidance-safe.
func trustedCodexJournalExecutable(projectRoot string, operations *codexRuleInstallOperations) (string, error) {
	if projectRoot == "" {
		projectRoot, _ = os.Getwd()
	}
	lookPath := exec.LookPath
	forbiddenRoots := codexJournalForbiddenExecutableRoots(projectRoot)
	if operations != nil {
		if operations.lookPath != nil {
			lookPath = operations.lookPath
		}
		if operations.forbiddenRoots != nil {
			forbiddenRoots = operations.forbiddenRoots
		}
	}
	path, err := lookPath("loaf")
	if err != nil {
		return "", fmt.Errorf("cannot trust Codex basic command policy: loaf executable is not on PATH: %w", err)
	}
	render, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot trust Codex basic command policy: absolutize loaf executable %s: %w", path, err)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("cannot trust Codex basic command policy: resolve loaf executable %s: %w", path, err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return "", fmt.Errorf("cannot trust Codex basic command policy: canonicalize loaf executable: %w", err)
	}
	if strings.ContainsAny(render, "`\r\n") || strings.ContainsAny(canonical, "`\r\n") {
		return "", fmt.Errorf("cannot trust Codex basic command policy: executable path contains unsupported guidance characters")
	}
	for _, forbidden := range forbiddenRoots {
		if forbidden == "" {
			continue
		}
		forbiddenPath, resolveErr := filepath.EvalSymlinks(forbidden)
		if resolveErr != nil {
			forbiddenPath = forbidden
		}
		forbiddenPath, resolveErr = filepath.Abs(forbiddenPath)
		if resolveErr != nil {
			continue
		}
		if pathWithinInstall(forbiddenPath, canonical) {
			return "", fmt.Errorf("cannot trust Codex basic command policy: loaf executable %s is inside forbidden path %s", canonical, forbiddenPath)
		}
		// The render path is what rendered policy ultimately trusts; an
		// entrypoint inside a forbidden root is rejected even when its
		// canonical target is not.
		if absForbidden, absErr := filepath.Abs(forbidden); absErr == nil && pathWithinInstall(absForbidden, render) {
			return "", fmt.Errorf("cannot trust Codex basic command policy: loaf executable %s is inside forbidden path %s", render, absForbidden)
		}
		if pathWithinInstall(forbiddenPath, render) {
			return "", fmt.Errorf("cannot trust Codex basic command policy: loaf executable %s is inside forbidden path %s", render, forbiddenPath)
		}
	}
	return render, nil
}

func codexJournalForbiddenExecutableRoots(projectRoot string) []string {
	roots := []string{projectRoot, os.TempDir(), "/tmp", "/private/tmp", "/var/tmp"}
	if runtime.GOOS == "darwin" {
		roots = append(roots, "/var/folders", "/private/var/folders")
	}
	if runtime.GOOS == "linux" {
		roots = append(roots, "/dev/shm", "/run/user", "/run/lock", "/run/shm")
		if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(runtimeDir) {
			roots = append(roots, runtimeDir)
		}
	}
	return roots
}

func pathWithinInstall(parent string, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
