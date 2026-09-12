package devtool

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// DevLinkStatus is the outcome of a launcher-pointer refresh.
type DevLinkStatus string

const (
	DevLinkLinked   DevLinkStatus = "linked"
	DevLinkConflict DevLinkStatus = "conflict"
	DevLinkSkipped  DevLinkStatus = "skipped"
	DevLinkFailed   DevLinkStatus = "failed"
)

// DevLinkResult names what happened and where.
type DevLinkResult struct {
	Status  DevLinkStatus
	Link    string
	Pointer string
	Err     error
}

// DevLinkOptions parameterize RefreshDevBuildLink; tests set Home and
// DataHome, production reads them from Env.
type DevLinkOptions struct {
	Env      Env
	Home     string
	DataHome string
	Warn     func(string)
	// BeforeClaimPublic runs between observing the PATH name and creating it;
	// tests use it to inject the race the create-exclusive claim guards against.
	BeforeClaimPublic func()
	// BeforeReplacePointer runs immediately before the pointer is published;
	// tests use it to inject a late foreign public claim after the old pointer
	// is still the active target.
	BeforeReplacePointer func()
	// Symlink and MkdirAll default to the os functions; tests inject failures.
	Symlink  func(target, link string) error
	MkdirAll func(path string, perm os.FileMode) error
}

// RefreshDevBuildLink makes the last successful dev build the active CLI
// (docs/architecture/runtime-and-delivery.md): it retargets $XDG_DATA_HOME/loaf/current-dev-launcher at the
// checkout's bin/loaf and creates ~/.local/bin/loaf only when that name is
// absent, as a relative symlink to the pointer. A real file, directory, or any
// other symlink at the PATH name is never replaced. Callers that require
// activation, such as Install, treat conflict, skip, and failure as errors.
func RefreshDevBuildLink(launcher string, options DevLinkOptions) DevLinkResult {
	warn := options.Warn
	if warn == nil {
		warn = func(string) {}
	}
	symlink := options.Symlink
	if symlink == nil {
		symlink = os.Symlink
	}
	mkdirAll := options.MkdirAll
	if mkdirAll == nil {
		mkdirAll = os.MkdirAll
	}
	home := options.Home
	if home == "" {
		home = options.Env["HOME"]
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	dataHome := options.DataHome
	if dataHome == "" {
		if xdg := options.Env["XDG_DATA_HOME"]; xdg != "" && filepath.IsAbs(xdg) {
			dataHome = xdg
		} else {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}
	publicLink := filepath.Join(home, ".local", "bin", "loaf")
	pointer := filepath.Join(dataHome, "loaf", "current-dev-launcher")
	result := DevLinkResult{Link: publicLink, Pointer: pointer}

	if runtime.GOOS == "windows" {
		result.Status = DevLinkSkipped
		return result
	}
	conflict := func(path, reason string) DevLinkResult {
		warn(fmt.Sprintf("WARN: not linking latest dev build; %s", reason))
		result.Status = DevLinkConflict
		result.Err = fmt.Errorf("%s: %s", path, reason)
		return result
	}
	fail := func(err error) DevLinkResult {
		warn(fmt.Sprintf("WARN: failed to link latest dev build (%v)", err))
		result.Status = DevLinkFailed
		result.Err = err
		return result
	}

	absLauncher, err := filepath.Abs(launcher)
	if err != nil {
		return fail(err)
	}
	if err := mkdirAll(filepath.Dir(publicLink), 0o755); err != nil {
		return fail(err)
	}
	if err := mkdirAll(filepath.Dir(pointer), 0o755); err != nil {
		return fail(err)
	}

	lock, err := acquireDevLinkLock(filepath.Join(filepath.Dir(pointer), ".dev-launcher.lock"))
	if err != nil {
		return fail(fmt.Errorf("serialize development launcher activation: %w", err))
	}
	defer releaseDevLinkLock(lock)

	if options.BeforeClaimPublic != nil {
		options.BeforeClaimPublic()
	}

	createdPublic, err := ensureOwnedPublicLink(publicLink, pointer, symlink)
	if err != nil {
		if errors.Is(err, errDevLinkConflict) {
			warn(publicConflictWarning(publicLink, pointer))
			result.Status = DevLinkConflict
			result.Err = fmt.Errorf("%s already exists and was not replaced", publicLink)
			return result
		}
		return fail(err)
	}
	abandonPublic := func() {
		removeCreatedPublicIfStillOwned(publicLink, pointer, createdPublic)
	}

	if info, err := os.Lstat(pointer); err == nil && info.Mode()&os.ModeSymlink == 0 {
		abandonPublic()
		return conflict(pointer, pointer+" is not a symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		abandonPublic()
		return fail(err)
	}

	temporary := uniqueTempName(pointer)
	pointerTarget, err := relativeSymlinkTarget(pointer, absLauncher)
	if err != nil {
		abandonPublic()
		return fail(err)
	}
	if err := symlink(pointerTarget, temporary); err != nil {
		abandonPublic()
		return fail(err)
	}
	if options.BeforeReplacePointer != nil {
		options.BeforeReplacePointer()
	}
	if !publicPointsAt(publicLink, pointer) {
		_ = os.Remove(temporary)
		abandonPublic()
		warn(publicConflictWarning(publicLink, pointer))
		result.Status = DevLinkConflict
		result.Err = fmt.Errorf("%s already exists and was not replaced", publicLink)
		return result
	}
	if info, err := os.Lstat(pointer); err == nil && info.Mode()&os.ModeSymlink == 0 {
		_ = os.Remove(temporary)
		abandonPublic()
		return conflict(pointer, pointer+" is not a symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(temporary)
		abandonPublic()
		return fail(err)
	}
	if err := os.Rename(temporary, pointer); err != nil {
		_ = os.Remove(temporary)
		abandonPublic()
		return fail(err)
	}
	result.Status = DevLinkLinked
	return result
}

var errDevLinkConflict = errors.New("development launcher conflict")

func ensureOwnedPublicLink(publicLink, pointer string, symlink func(target, link string) error) (os.FileInfo, error) {
	if _, err := os.Lstat(publicLink); err == nil {
		if publicPointsAt(publicLink, pointer) {
			return nil, nil
		}
		return nil, errDevLinkConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	publicTarget, err := relativeSymlinkTarget(publicLink, pointer)
	if err != nil {
		return nil, err
	}
	if err := symlink(publicTarget, publicLink); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if publicPointsAt(publicLink, pointer) {
			return nil, nil
		}
		return nil, errDevLinkConflict
	}
	info, err := os.Lstat(publicLink)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink == 0 || !publicPointsAt(publicLink, pointer) {
		return nil, errDevLinkConflict
	}
	return info, nil
}

func removeCreatedPublicIfStillOwned(publicLink, pointer string, created os.FileInfo) {
	if created == nil {
		return
	}
	current, err := os.Lstat(publicLink)
	if err != nil {
		return
	}
	if !os.SameFile(created, current) {
		return
	}
	if !publicPointsAt(publicLink, pointer) {
		return
	}
	_ = os.Remove(publicLink)
}

func uniqueTempName(path string) string {
	return fmt.Sprintf("%s.tmp-%d-%d", path, os.Getpid(), time.Now().UnixNano())
}

func publicPointsAt(publicLink, pointer string) bool {
	info, err := os.Lstat(publicLink)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(publicLink)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		parent, err := physicalDir(filepath.Dir(publicLink))
		if err != nil {
			return false
		}
		target = filepath.Join(parent, target)
	}
	resolvedTarget, err := physicalPathPreserveBase(target)
	if err != nil {
		return false
	}
	resolvedPointer, err := physicalPathPreserveBase(pointer)
	if err != nil {
		return false
	}
	return resolvedTarget == resolvedPointer
}

func relativeSymlinkTarget(link, dest string) (string, error) {
	// Resolve only the parent directories so a /var → /private/var alias
	// cannot produce a broken lexical hop, while the dest basename stays
	// the pointer name rather than the binary it currently refers to.
	linkParent, err := physicalDir(filepath.Dir(link))
	if err != nil {
		return "", err
	}
	destParent, err := physicalDir(filepath.Dir(dest))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(linkParent, destParent)
	if err != nil {
		return "", err
	}
	return filepath.Join(rel, filepath.Base(dest)), nil
}

func physicalDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func physicalPathPreserveBase(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := physicalDir(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

func publicConflictWarning(publicLink, pointer string) string {
	info, err := os.Lstat(publicLink)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return fmt.Sprintf("WARN: not linking latest dev build; %s is not a symlink", publicLink)
	}
	if isLoafCheckoutLink(publicLink) {
		return fmt.Sprintf("WARN: not linking latest dev build; %s already points at a Loaf checkout and will not be replaced. Remove it and run make install to install the last-build pointer at %s", publicLink, pointer)
	}
	return fmt.Sprintf("WARN: not linking latest dev build; %s is not Loaf's launcher pointer", publicLink)
}

// isLoafCheckoutLink recognizes the pre-pointer scheme: a PATH symlink straight
// at some checkout's bin/loaf, identified by the package.json beside bin/.
func isLoafCheckoutLink(link string) bool {
	target, err := os.Readlink(link)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	packageRoot := filepath.Dir(filepath.Dir(target))
	body, err := os.ReadFile(filepath.Join(packageRoot, "package.json"))
	if err != nil {
		return false
	}
	var manifest struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return false
	}
	return manifest.Name == "loaf" && filepath.Clean(target) == filepath.Join(packageRoot, "bin", "loaf")
}
