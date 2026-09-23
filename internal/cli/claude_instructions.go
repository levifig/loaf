package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// Every .claude/CLAUDE.md repair runs through an *os.Root opened at the
// project root, so no Lstat, Readlink, Remove, Rename, or read can resolve
// outside the project — including when .claude is swapped for a symlink after
// the friendly symlinkedClaudeDirTarget refusal already ran. Symlinks that stay
// inside the project are still followed by the Root; the refusal above is what
// keeps a symlinked .claude from being repaired at all in the ordinary case.

// claudeInstructionsName is claudeInstructionsPath in the Root's separator form.
var claudeInstructionsName = filepath.FromSlash(claudeInstructionsPath)

// lstatClaudeInstructions reports the .claude/CLAUDE.md entry through root. A
// missing entry, or a .claude that is not a directory, reports absent.
func lstatClaudeInstructions(root *os.Root) (fs.FileInfo, bool, error) {
	info, err := root.Lstat(claudeInstructionsName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return info, true, nil
}

// claudeInstructionsLinkTarget reads the .claude/CLAUDE.md link through root and
// resolves it lexically against the project's .claude directory, the way
// Claude Code names it from the project.
func claudeInstructionsLinkTarget(root *os.Root, projectRoot string) (string, error) {
	target, err := root.Readlink(claudeInstructionsName)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target), nil
	}
	return filepath.Clean(filepath.Join(projectRoot, ".claude", target)), nil
}

// removeClaudeInstructionsLink removes the .claude/CLAUDE.md symlink itself,
// never what it points at, and refuses if the entry is no longer a symlink.
func removeClaudeInstructionsLink(root *os.Root) error {
	info, exists, err := lstatClaudeInstructions(root)
	if err != nil {
		return err
	}
	if !exists || info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is no longer a symlink; re-run to re-check it", claudeInstructionsPath)
	}
	return root.Remove(claudeInstructionsName)
}

// retireClaudeInstructionsFile retires a real .claude/CLAUDE.md. It moves the
// file to a collision-safe backup first and then reads the backup, so the
// merged content is exactly the backed-up bytes even if the path is edited
// concurrently. The read keeps the project size limit and refuses anything but
// a regular file. The user content (managed fence stripped) is merged into root
// AGENTS.md atomically; if the read or merge fails the backup is renamed back.
// The path is left absent so Claude Code reads root AGENTS.md natively. It
// returns the backup's project-relative slash path and whether content merged.
func retireClaudeInstructionsFile(root *os.Root, canonical string) (string, bool, error) {
	info, exists, err := lstatClaudeInstructions(root)
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, fmt.Errorf("%s no longer exists; re-run to re-check it", claudeInstructionsPath)
	}
	if !info.Mode().IsRegular() {
		return "", false, refuseProjectFileRead(&fs.PathError{Op: "open", Path: claudeInstructionsPath, Err: errNotRegularFile})
	}
	backup, err := collisionSafeRootBackupName(root, claudeInstructionsName)
	if err != nil {
		return "", false, err
	}
	if err := root.Rename(claudeInstructionsName, backup); err != nil {
		return "", false, err
	}
	merged, err := mergeRetiredClaudeInstructions(root, backup, canonical)
	if err != nil {
		if rollbackErr := root.Rename(backup, claudeInstructionsName); rollbackErr != nil {
			err = fmt.Errorf("%w (rollback failed: %v)", err, rollbackErr)
		}
		return "", false, err
	}
	return filepath.ToSlash(backup), merged, nil
}

func mergeRetiredClaudeInstructions(root *os.Root, backup string, canonical string) (bool, error) {
	body, err := readRootRegularFile(root, backup, projectFileReadLimit)
	if err != nil {
		return false, refuseProjectFileRead(err)
	}
	merged, err := mergeDoctorContentIntoCanonical(canonical, stripDoctorLoafFence(string(body)), claudeInstructionsPath)
	if err == nil && !doctorFileExists(canonical) {
		err = writeFileAtomically(canonical, []byte{}, 0o644)
	}
	return merged, err
}

// collisionSafeRootBackupName is collisionSafeInstallBackupPath through root:
// name.bak, or the first free name.bak.N.
func collisionSafeRootBackupName(root *os.Root, name string) (string, error) {
	base := name + ".bak"
	for index := 0; ; index++ {
		candidate := base
		if index > 0 {
			candidate = base + "." + strconv.Itoa(index)
		}
		_, err := root.Lstat(candidate)
		if errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
}

// readRootRegularFile reads a regular file whole through root. It settles the
// type with Lstat, opens without blocking on a FIFO where the platform allows,
// and requires the descriptor to be the same regular file Lstat saw, then
// applies the same size limit as readRegularFile.
func readRootRegularFile(root *os.Root, name string, limit int64) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, &fs.PathError{Op: "open", Path: filepath.ToSlash(name), Err: errNotRegularFile}
	}
	return readOpenedRegularFile(filepath.ToSlash(name), limit, func(string) (*os.File, error) {
		file, err := root.OpenFile(name, rootRegularFileReadFlags, 0)
		if err != nil {
			return nil, err
		}
		opened, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
			file.Close()
			return nil, &fs.PathError{Op: "open", Path: filepath.ToSlash(name), Err: errNotRegularFile}
		}
		return file, nil
	})
}
