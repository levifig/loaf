package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type installSymlinkOptions struct {
	Prompt         func(question string) bool
	NonInteractive bool
	AssumeYes      bool
}

type installSymlinkResult struct {
	Action     string
	Message    string
	BackupPath string
	Merged     bool
	Error      string
	// Refused marks a migration step that stopped because a whole-file read
	// failed. Every failure carries the same abort semantics: the project part
	// fails, the fenced and MCP writers that follow are skipped, and the run
	// exits non-zero. Dry-run and apply agree on that reading.
	Refused bool
}

var agentsMDInstallTargets = map[string]bool{
	"cursor":   true,
	"codex":    true,
	"opencode": true,
	"amp":      true,
}

// claudeInstructionsPath is the project-relative Claude Code instruction path
// that shadows root AGENTS.md whenever it holds anything but a link to it.
const claudeInstructionsPath = ".claude/CLAUDE.md"

// symlinkedClaudeDirTarget reports where a symlinked .claude directory points
// when something exists at .claude/CLAUDE.md through it. Loaf never inspects or
// repairs .claude/CLAUDE.md through a symlinked directory: a backup, merge
// source, or removal there could land outside the project, and the lexical
// ../AGENTS.md check would no longer name root AGENTS.md. A symlinked .claude
// with nothing at CLAUDE.md shadows nothing and needs no refusal.
func symlinkedClaudeDirTarget(projectRoot string) (string, bool) {
	claudeDir := filepath.Join(projectRoot, ".claude")
	if !installIsSymlink(claudeDir) {
		return "", false
	}
	if !installPathExists(filepath.Join(projectRoot, filepath.FromSlash(claudeInstructionsPath))) {
		return "", false
	}
	target := resolveInstallSymlinkTarget(claudeDir)
	if target == "" {
		target = "<unreadable>"
	}
	return target, true
}

func symlinkedClaudeDirMessage(target string) string {
	return fmt.Sprintf(".claude is a symlink to %s; Loaf will not inspect or repair .claude/CLAUDE.md through it. Replace .claude with a real directory, or move CLAUDE.md out of it, then rerun", target)
}

// ensureInstallClaudeInstructions keeps Claude Code reading root AGENTS.md
// natively. It never creates or relinks .claude/CLAUDE.md: an absent path and a
// link to root AGENTS.md are both correct and left alone. A link elsewhere
// (including a dangling one) is removed, leaving its target untouched. A real
// file is merged into root AGENTS.md, moved to a collision-safe backup, and the
// path is left absent. Both repairs need consent, as every project-file repair
// does.
func ensureInstallClaudeInstructions(projectRoot string, options installSymlinkOptions) installSymlinkResult {
	claudePath := filepath.Join(projectRoot, filepath.FromSlash(claudeInstructionsPath))
	canonical := filepath.Join(projectRoot, "AGENTS.md")
	if target, symlinked := symlinkedClaudeDirTarget(projectRoot); symlinked {
		message := symlinkedClaudeDirMessage(target)
		result := installSymlinkError("error", message, errors.New(message))
		result.Refused = true
		return result
	}
	if !installPathExists(claudePath) {
		return installSymlinkResult{Action: "already-correct", Message: "No .claude/CLAUDE.md; Claude Code reads root AGENTS.md"}
	}

	if installIsSymlink(claudePath) {
		if installSymlinkPointsTo(claudePath, canonical) {
			return installSymlinkResult{Action: "already-correct", Message: ".claude/CLAUDE.md already points to ../AGENTS.md"}
		}
		actualTarget := resolveInstallSymlinkTarget(claudePath)
		if actualTarget == "" {
			actualTarget = "<unreadable>"
		}
		approved := options.AssumeYes
		if !approved {
			if options.NonInteractive {
				return installSymlinkResult{
					Action:  "skipped-no-tty",
					Message: fmt.Sprintf(".claude/CLAUDE.md points to %s, which Claude Code reads instead of root AGENTS.md; skipped in non-interactive mode", actualTarget),
				}
			}
			if options.Prompt != nil {
				approved = options.Prompt(fmt.Sprintf("  .claude/CLAUDE.md points to %s, so Claude Code reads it instead of root AGENTS.md. Remove the link? [y/N] ", actualTarget))
			}
		}
		if !approved {
			return installSymlinkResult{Action: "declined-remove", Message: fmt.Sprintf("Left .claude/CLAUDE.md pointing at %s (Claude Code will not read root AGENTS.md)", actualTarget)}
		}
		if err := os.Remove(claudePath); err != nil {
			return installSymlinkError("error", fmt.Sprintf("Failed to remove .claude/CLAUDE.md: %v", err), err)
		}
		return installSymlinkResult{Action: "removed", Message: fmt.Sprintf("Removed .claude/CLAUDE.md symlink to %s; Claude Code reads root AGENTS.md", actualTarget)}
	}

	approved := options.AssumeYes
	if !approved {
		if options.NonInteractive {
			return installSymlinkResult{
				Action:  "skipped-no-tty",
				Message: ".claude/CLAUDE.md exists as a real file, which stops Claude Code from reading root AGENTS.md; skipped in non-interactive mode",
			}
		}
		if options.Prompt != nil {
			approved = options.Prompt("  .claude/CLAUDE.md exists as a regular file, which stops Claude Code from reading root AGENTS.md. Merge its content into root AGENTS.md and move it to a backup? [y/N] ")
		}
	}
	if !approved {
		return installSymlinkResult{Action: "declined-replace", Message: "Left .claude/CLAUDE.md as a regular file (Claude Code will not read root AGENTS.md)"}
	}

	body, err := readRegularFileNoFollow(claudePath, projectFileReadLimit)
	if err != nil {
		return installSymlinkReadRefusal("Failed to migrate .claude/CLAUDE.md", refuseProjectFileRead(err))
	}
	backup, merged, err := retireClaudeInstructionsFile(claudePath, body, canonical, claudeInstructionsPath)
	if err != nil {
		return installSymlinkReadRefusal("Failed to migrate .claude/CLAUDE.md", err)
	}
	suffix := ""
	if merged {
		suffix = " and merged its content into root AGENTS.md"
	}
	return installSymlinkResult{
		Action:     "migrated",
		Message:    fmt.Sprintf("Moved .claude/CLAUDE.md to %s%s; Claude Code reads root AGENTS.md", projectRelativeSlashPath(projectRoot, backup), suffix),
		BackupPath: backup,
		Merged:     merged,
	}
}

func ensureProjectInstallSymlinks(projectRoot string, selectedTargets []string, hasClaudeCode bool, options installSymlinkOptions) map[string]installSymlinkResult {
	results := map[string]installSymlinkResult{}
	wantClaude := hasClaudeCode || containsString(selectedTargets, "claude-code")
	wantRootAgents := needsRootInstallAgentsFile(selectedTargets)
	if !wantClaude && !wantRootAgents {
		return results
	}

	rootResult := ensureRootInstallAgentsFile(projectRoot, options)
	results["./AGENTS.md"] = rootResult
	if rootResult.Error != "" {
		return results
	}

	if wantClaude {
		results[claudeInstructionsPath] = ensureInstallClaudeInstructions(projectRoot, options)
	}
	return results
}

func needsRootInstallAgentsFile(targets []string) bool {
	for _, target := range targets {
		if agentsMDInstallTargets[target] {
			return true
		}
	}
	return false
}

func ensureRootInstallAgentsFile(projectRoot string, options installSymlinkOptions) installSymlinkResult {
	canonical := filepath.Join(projectRoot, "AGENTS.md")
	legacy := filepath.Join(projectRoot, ".agents", "AGENTS.md")
	legacyExists := installFileExists(legacy) && !installIsSymlink(legacy)
	if installIsDirectory(canonical) {
		err := fmt.Errorf("./AGENTS.md is a directory; expected a canonical real file")
		return installSymlinkError("error", err.Error(), err)
	}

	if installIsSymlink(canonical) && legacyExists && installSymlinkPointsTo(canonical, legacy) {
		linkBackup := collisionSafeInstallBackupPath(canonical)
		if err := os.Rename(canonical, linkBackup); err != nil {
			return installSymlinkError("error", fmt.Sprintf("Failed to migrate ./AGENTS.md: %v", err), err)
		}
		if err := os.Rename(legacy, canonical); err != nil {
			_ = os.Rename(linkBackup, canonical)
			return installSymlinkError("error", fmt.Sprintf("Failed to migrate .agents/AGENTS.md: %v", err), err)
		}
		_ = os.Remove(linkBackup)
		return installSymlinkResult{Action: "migrated", Message: "Migrated .agents/AGENTS.md to canonical ./AGENTS.md"}
	}

	if !installPathExists(canonical) {
		if legacyExists {
			if err := os.Rename(legacy, canonical); err != nil {
				return installSymlinkError("error", fmt.Sprintf("Failed to migrate .agents/AGENTS.md: %v", err), err)
			}
			return installSymlinkResult{Action: "migrated", Message: "Migrated .agents/AGENTS.md to canonical ./AGENTS.md"}
		}
		if err := os.WriteFile(canonical, []byte{}, 0o644); err != nil {
			return installSymlinkError("error", fmt.Sprintf("Failed to create ./AGENTS.md: %v", err), err)
		}
		return installSymlinkResult{Action: "created", Message: "Created canonical ./AGENTS.md"}
	}

	if installIsSymlink(canonical) {
		approved := options.AssumeYes
		if !approved {
			if options.NonInteractive {
				return installSymlinkResult{Action: "skipped-no-tty", Message: "./AGENTS.md is a symlink; skipped conversion to a canonical real file in non-interactive mode"}
			}
			if options.Prompt != nil {
				approved = options.Prompt("  ./AGENTS.md is a symlink. Preserve its current content and replace it with the canonical real file? [y/N] ")
			}
		}
		if !approved {
			return installSymlinkResult{Action: "declined-replace", Message: "Left ./AGENTS.md as a symlink"}
		}
		body, err := readRegularFile(canonical, projectFileReadLimit)
		if err != nil && !os.IsNotExist(err) {
			return installSymlinkReadRefusal("Failed to read ./AGENTS.md", refuseProjectFileRead(err))
		}
		backup := collisionSafeInstallBackupPath(canonical)
		if err := os.Rename(canonical, backup); err != nil {
			return installSymlinkError("error", fmt.Sprintf("Failed to back up ./AGENTS.md: %v", err), err)
		}
		if err := os.WriteFile(canonical, body, 0o644); err != nil {
			_ = os.Rename(backup, canonical)
			return installSymlinkError("error", fmt.Sprintf("Failed to create canonical ./AGENTS.md: %v", err), err)
		}
		return installSymlinkResult{Action: "replaced-file", Message: "Backed up the old ./AGENTS.md symlink and created a canonical real file", BackupPath: backup}
	}

	if legacyExists {
		approved := options.AssumeYes
		if !approved {
			if options.NonInteractive {
				return installSymlinkResult{Action: "skipped-no-tty", Message: "Both ./AGENTS.md and .agents/AGENTS.md are real files; skipped merge in non-interactive mode"}
			}
			if options.Prompt != nil {
				approved = options.Prompt("  Both ./AGENTS.md and .agents/AGENTS.md are real files. Merge legacy user content into root AGENTS.md and retire the legacy file with a backup? [y/N] ")
			}
		}
		if !approved {
			return installSymlinkResult{Action: "declined-replace", Message: "Left both ./AGENTS.md and .agents/AGENTS.md unchanged"}
		}
		body, err := readRegularFile(legacy, projectFileReadLimit)
		if err != nil {
			return installSymlinkReadRefusal("Failed to migrate .agents/AGENTS.md", refuseProjectFileRead(err))
		}
		stripped := stripDoctorLoafFence(string(body))
		backup := collisionSafeInstallBackupPath(legacy)
		if err := os.Rename(legacy, backup); err != nil {
			return installSymlinkError("error", fmt.Sprintf("Failed to back up .agents/AGENTS.md: %v", err), err)
		}
		merged, err := mergeLegacyAgentsContentIntoCanonical(canonical, stripped, ".agents/AGENTS.md")
		if err != nil {
			if rollbackErr := os.Rename(backup, legacy); rollbackErr != nil {
				err = fmt.Errorf("%w (rollback failed: %v)", err, rollbackErr)
			}
			return installSymlinkReadRefusal("Failed to migrate .agents/AGENTS.md", err)
		}
		return installSymlinkResult{Action: "migrated", Message: "Migrated legacy .agents/AGENTS.md into canonical ./AGENTS.md", BackupPath: backup, Merged: merged}
	}

	return installSymlinkResult{Action: "already-correct", Message: "Canonical ./AGENTS.md already exists"}
}

func installSymlinkError(action string, message string, err error) installSymlinkResult {
	return installSymlinkResult{Action: action, Message: message, Error: err.Error()}
}

// installSymlinkReadRefusal reports a migration step that stopped because a
// whole-file read failed — not a regular file, too large, permission denied,
// or any other I/O failure. The step is abandoned before it writes anything,
// and the run treats the result as a failed project part rather than a note in
// passing: the fenced writers that follow would meet the same path, so the
// project part reports itself failed and upgrade exits non-zero.
func installSymlinkReadRefusal(prefix string, err error) installSymlinkResult {
	result := installSymlinkError("error", fmt.Sprintf("%s: %v", prefix, err), err)
	result.Refused = true
	return result
}

func installPathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func installFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func installIsDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

func collisionSafeInstallBackupPath(path string) string {
	base := path + ".bak"
	if !installPathExists(base) {
		return base
	}
	for index := 1; ; index++ {
		candidate := base + "." + strconv.Itoa(index)
		if !installPathExists(candidate) {
			return candidate
		}
	}
}

func mergeLegacyAgentsContentIntoCanonical(canonical string, stripped string, relSource string) (bool, error) {
	if stripped == "" {
		return false, nil
	}
	existing, err := readRegularFile(canonical, projectFileReadLimit)
	if err != nil {
		return false, refuseProjectFileRead(err)
	}
	trimmedExisting := strings.TrimRight(string(existing), " \t\r\n")
	merged := trimmedExisting + "\n\n## Migrated from " + relSource + "\n\n" + stripped + "\n"
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(canonical); statErr == nil {
		mode = info.Mode().Perm()
	}
	temp, err := os.CreateTemp(filepath.Dir(canonical), ".loaf-agents-md-*")
	if err != nil {
		return false, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return false, err
	}
	if _, err := temp.WriteString(merged); err != nil {
		_ = temp.Close()
		return false, err
	}
	if err := temp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tempPath, canonical); err != nil {
		return false, err
	}
	return true, nil
}

func installIsSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func resolveInstallSymlinkTarget(linkPath string) string {
	target, err := os.Readlink(linkPath)
	if err != nil {
		return ""
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(linkPath), target))
}

func installSymlinkPointsTo(linkPath string, expectedAbs string) bool {
	resolved := resolveInstallSymlinkTarget(linkPath)
	return resolved != "" && filepath.Clean(resolved) == filepath.Clean(expectedAbs)
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
