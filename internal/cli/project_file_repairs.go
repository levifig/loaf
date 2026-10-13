package cli

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// renameProjectFileNoReplace resolves both parent directories through the
// project root and uses an exclusive rename. Link followed by Remove is not
// equivalent: a writer can replace the source between those two operations.
func renameProjectFileNoReplace(root *os.Root, from, to string) error {
	oldParent, err := root.OpenRoot(filepath.Dir(from))
	if err != nil {
		return err
	}
	defer oldParent.Close()
	newParent, err := root.OpenRoot(filepath.Dir(to))
	if err != nil {
		return err
	}
	defer newParent.Close()
	oldDir, err := oldParent.Open(".")
	if err != nil {
		return err
	}
	defer oldDir.Close()
	newDir, err := newParent.Open(".")
	if err != nil {
		return err
	}
	defer newDir.Close()
	if err := exclusiveRename(oldDir, filepath.Base(from), newDir, filepath.Base(to)); err != nil {
		return fmt.Errorf("refusing to move %s to %s; source retained at %s and destination left intact: %w", filepath.ToSlash(from), filepath.ToSlash(to), filepath.ToSlash(from), err)
	}
	return nil
}

func projectFileInfo(root *os.Root, name string) (fs.FileInfo, bool, error) {
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return info, err == nil, projectFileError(root, name, err)
}

// readProjectFile follows only links that remain inside the opened project.
// The descriptor check preserves the nonblocking special-file read guard.
func readProjectFile(root *os.Root, name string) ([]byte, error) {
	// os.Root intentionally rejects absolute symlinks, including safe links
	// inside this project. Normalize only an absolute leaf target to a relative
	// name; Root still resolves all intermediate components and the open itself.
	readName := name
	leaf, err := root.Lstat(name)
	if err != nil {
		return nil, projectFileError(root, name, err)
	}
	if leaf.Mode()&os.ModeSymlink != 0 {
		target, err := root.Readlink(name)
		if err != nil {
			return nil, projectFileError(root, name, err)
		}
		if filepath.IsAbs(target) {
			relative, err := filepath.Rel(root.Name(), filepath.Clean(target))
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return nil, projectFileError(root, name, fmt.Errorf("absolute link points outside project; refusing to read"))
			}
			readName = relative
		}
	}
	before, err := root.Stat(readName)
	if err != nil {
		return nil, projectFileError(root, name, err)
	}
	if !before.Mode().IsRegular() {
		return nil, projectFileError(root, name, &fs.PathError{Op: "open", Path: name, Err: errNotRegularFile})
	}
	body, err := readOpenedRegularFile(name, projectFileReadLimit, func(string) (*os.File, error) {
		f, err := root.OpenFile(readName, rootRegularFileReadFlags, 0)
		if err != nil {
			return nil, projectFileError(root, name, err)
		}
		opened, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, projectFileError(root, name, err)
		}
		if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
			f.Close()
			return nil, projectFileError(root, name, &fs.PathError{Op: "open", Path: name, Err: errNotRegularFile})
		}
		return f, nil
	})
	return body, projectFileError(root, name, err)
}

func readProjectFileAt(projectRoot, name string) ([]byte, error) {
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return readProjectFile(root, name)
}

func inspectProjectFiles(projectRoot string, names ...string) error {
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, name := range names {
		if _, _, err := projectFileInfo(root, name); err != nil {
			return err
		}
	}
	return nil
}

func createProjectFile(root *os.Root, name string, body []byte, mode fs.FileMode) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("refusing to replace %s: %w", name, err)
	}
	if _, err = f.Write(body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func restoreProjectFile(root *os.Root, backup, name string, cause error) error {
	if err := renameProjectFileNoReplace(root, backup, name); err != nil {
		return fmt.Errorf("%w (rollback refused: %v; recovery retained at %s; destination %s left intact)", cause, err, filepath.ToSlash(backup), filepath.ToSlash(name))
	}
	return cause
}

// removeProjectEntry first captures the checked entry inside the root. If it
// changed type or identity, restore without replacing any newly created path.
func removeProjectEntry(root *os.Root, name string, before fs.FileInfo) error {
	backup, err := collisionSafeRootBackupName(root, name)
	if err != nil {
		return err
	}
	if err := renameProjectFileNoReplace(root, name, backup); err != nil {
		return err
	}
	captured, err := root.Lstat(backup)
	if err != nil {
		return restoreProjectFile(root, backup, name, err)
	}
	if !os.SameFile(before, captured) || before.Mode().Type() != captured.Mode().Type() {
		return restoreProjectFile(root, backup, name, fmt.Errorf("%s changed during inspection; re-run to re-check", name))
	}
	if err := root.Remove(backup); err != nil {
		return restoreProjectFile(root, backup, name, err)
	}
	return nil
}

func mergeRootProjectContent(root *os.Root, canonical, stripped, relSource string) (bool, error) {
	if stripped == "" {
		return false, nil
	}
	existing, err := readProjectFile(root, canonical)
	existed := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, refuseProjectFileRead(err)
	}
	trimmed := strings.TrimRight(string(existing), " \t\r\n")
	body := stripped + "\n"
	// Legacy AGENTS migration retains its source heading even in an empty file.
	if trimmed != "" || existed && relSource == ".agents/AGENTS.md" {
		body = trimmed + "\n\n## Migrated from " + relSource + "\n\n" + stripped + "\n"
	}
	if !existed {
		return true, createProjectFile(root, canonical, []byte(body), 0o644)
	}
	mode := fs.FileMode(0o644)
	if info, err := root.Stat(canonical); err == nil {
		mode = info.Mode().Perm()
	}
	tempName := ".loaf-agents-md-" + rand.Text()
	temp, err := root.OpenFile(tempName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return false, err
	}
	defer root.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return false, err
	}
	if _, err := temp.WriteString(body); err != nil {
		temp.Close()
		return false, err
	}
	if err := temp.Close(); err != nil {
		return false, err
	}
	return true, root.Rename(tempName, canonical)
}

func doctorProjectInspectionFailure(err error) doctorResult {
	return doctorResult{Status: doctorFail, Message: "Project instructions could not be inspected", Detail: err.Error()}
}

func rootProjectFileExists(root *os.Root, name string) bool {
	_, err := root.Stat(name)
	return err == nil
}

func readProjectFileNoFollowAt(projectRoot, name string) ([]byte, error) {
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	body, err := readRootRegularFile(root, name, projectFileReadLimit)
	return body, projectFileError(root, name, err)
}

func moveCheckedProjectFile(root *os.Root, from, to string, before fs.FileInfo) error {
	if err := renameProjectFileNoReplace(root, from, to); err != nil {
		return err
	}
	after, err := root.Lstat(to)
	if err != nil {
		return restoreProjectFile(root, to, from, err)
	}
	if !os.SameFile(before, after) || before.Mode().Type() != after.Mode().Type() {
		return restoreProjectFile(root, to, from, fmt.Errorf("%s changed during inspection; re-run to re-check", from))
	}
	return nil
}

// Root errors carry relative names. Keep the opened-root boundary while
// retaining the absolute path operators need to locate a refused project file.
func projectFileError(root *os.Root, name string, err error) error {
	if err == nil {
		return nil
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return &fs.PathError{Op: pe.Op, Path: filepath.Join(root.Name(), name), Err: pe.Err}
	}
	return fmt.Errorf("%s: %w", filepath.Join(root.Name(), name), err)
}
