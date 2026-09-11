package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"syscall"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

const worktreeBackPointerFile = ".moved-to"
const worktreePartialSuffix = ".partial.loaf-migrate"
const debugResolveEnv = "LOAF_DEBUG_RESOLVE"

const preA3RefusalMessageNative = `Reconcile the named storage paths manually, or inspect a dry-run preview:

    loaf migrate worktree-storage

That explicit migration covers the entire .agents/ tree, including configuration.
Review its conflict choices before --apply; applying can replace or remove copies.
(Tip: set LOAF_DEBUG_RESOLVE=1 to see git probe diagnostics.)`

type worktreeConflictPolicy string

const (
	worktreeConflictNewer    worktreeConflictPolicy = "newer"
	worktreeConflictMain     worktreeConflictPolicy = "main"
	worktreeConflictWorktree worktreeConflictPolicy = "worktree"
)

type worktreeMigrationOptions struct {
	apply          bool
	conflictPolicy worktreeConflictPolicy
	help           bool
}

type worktreeMigrationMove struct {
	from             string
	to               string
	rel              string
	conflict         bool
	resolution       string
	resolutionReason string
}

type worktreeMigrationPlan struct {
	worktreeAgents  string
	mainAgents      string
	mainRoot        string
	moves           []worktreeMigrationMove
	backPointerPath string
}

type worktreeMigrationResult struct {
	status   string
	message  string
	plan     *worktreeMigrationPlan
	partials []string
}

func (r Runner) runMigrateWorktreeStorage(args []string, out io.Writer, runtimeRoot string) error {
	options, err := parseWorktreeMigrationArgs(args)
	if err != nil {
		return err
	}
	if options.help {
		writeWorktreeMigrationHelp(out)
		return nil
	}
	result := runWorktreeStorageMigration(runtimeRoot, options)
	switch result.status {
	case "not-in-git", "main-missing", "partial-leftover", "symlink-unsupported":
		return fmt.Errorf("%s", result.message)
	default:
		fmt.Fprintln(out, formatWorktreeMigrationResult(result, options.apply))
		return nil
	}
}

func parseWorktreeMigrationArgs(args []string) (worktreeMigrationOptions, error) {
	options := worktreeMigrationOptions{conflictPolicy: worktreeConflictNewer}
	for _, arg := range args {
		switch arg {
		case "--apply":
			options.apply = true
		case "--force-from-worktree":
			if options.conflictPolicy == worktreeConflictMain {
				return worktreeMigrationOptions{}, fmt.Errorf("--force-from-worktree and --force-from-main are mutually exclusive")
			}
			options.conflictPolicy = worktreeConflictWorktree
		case "--force-from-main":
			if options.conflictPolicy == worktreeConflictWorktree {
				return worktreeMigrationOptions{}, fmt.Errorf("--force-from-worktree and --force-from-main are mutually exclusive")
			}
			options.conflictPolicy = worktreeConflictMain
		case "--help", "-h":
			options.help = true
		default:
			return worktreeMigrationOptions{}, fmt.Errorf("unknown migrate worktree-storage option %q", arg)
		}
	}
	return options, nil
}

func writeWorktreeMigrationHelp(out io.Writer) {
	fmt.Fprintln(out, strings.Join([]string{
		"Usage: loaf migrate worktree-storage [options]",
		"",
		"Move a linked worktree's local .agents/ into the main worktree.",
		"dry-run by default; pass --apply to mutate.",
		"",
		"Options:",
		"  --apply                Perform the migration",
		"  --force-from-worktree  On conflict, always keep the worktree-local copy",
		"  --force-from-main      On conflict, always keep the main-worktree copy",
		"  -h, --help             Show help",
		"",
		"Set LOAF_DEBUG_RESOLVE=1 to surface git probe diagnostics.",
	}, "\n"))
}

func runWorktreeStorageMigration(cwd string, options worktreeMigrationOptions) worktreeMigrationResult {
	mainRoot := findMainWorktreeRootNative(cwd)
	if mainRoot == "" {
		wtRoot := firstNonEmpty(findWorktreeRootNative(cwd), cwd)
		if pointerMainRoot := readGitdirPointerMainRootNative(wtRoot); pointerMainRoot != "" {
			exists, isDir := probeDirectory(pointerMainRoot)
			if !exists || !isDir {
				return worktreeMigrationResult{status: "main-missing", message: buildMainMissingMessageNative(pointerMainRoot, exists)}
			}
		}
		if !isInGitContextNative(cwd) {
			return worktreeMigrationResult{
				status:  "not-in-git",
				message: "loaf migrate worktree-storage: not in a git repository - this command is only meaningful inside a linked git worktree.",
			}
		}
		return worktreeMigrationResult{status: "not-in-worktree", message: "Nothing to migrate - already in the main worktree."}
	}
	exists, isDir := probeDirectory(mainRoot)
	if !exists || !isDir {
		return worktreeMigrationResult{status: "main-missing", message: buildMainMissingMessageNative(mainRoot, exists)}
	}

	wtRoot := firstNonEmpty(findWorktreeRootNative(cwd), cwd)
	localAgents := filepath.Join(wtRoot, ".agents")
	mainAgents := filepath.Join(mainRoot, ".agents")

	partials := findWorktreePartialPaths(mainAgents)
	if len(partials) > 0 {
		lines := []string{
			"Refusing to migrate: found leftover staging paths from a previous interrupted run.",
			"Resolve each path manually (delete it, or rename it into place if you trust the staged copy),",
			"then re-run the migrate command:",
			"",
		}
		for _, partial := range partials {
			lines = append(lines, "  "+partial)
		}
		lines = append(lines,
			"",
			fmt.Sprintf("These paths end in '%s' and were created by an EXDEV cross-filesystem", worktreePartialSuffix),
			"stage that did not complete the atomic rename to the final destination.",
		)
		return worktreeMigrationResult{status: "partial-leftover", message: strings.Join(lines, "\n"), partials: partials}
	}

	pointer := readBackPointerNative(localAgents)
	if pointer == mainRoot && !worktreeAgentsHasContentNative(localAgents) {
		return worktreeMigrationResult{status: "already-migrated", message: "Nothing to do - already migrated."}
	}
	if !worktreeAgentsHasContentNative(localAgents) {
		return worktreeMigrationResult{status: "no-local-agents", message: "Nothing to migrate - worktree has no local .agents/ content."}
	}

	symlinks := findWorktreeSymlinkPaths(localAgents)
	if len(symlinks) > 0 {
		lines := []string{
			"Refusing to migrate: found symlinks under worktree-local .agents/.",
			"Handle these paths manually, then re-run the migrate command:",
			"",
		}
		for _, symlink := range symlinks {
			lines = append(lines, "  "+symlink)
		}
		return worktreeMigrationResult{status: "symlink-unsupported", message: strings.Join(lines, "\n")}
	}

	plan := &worktreeMigrationPlan{
		worktreeAgents:  localAgents,
		mainAgents:      mainAgents,
		mainRoot:        mainRoot,
		moves:           planWorktreeMoves(localAgents, mainAgents, options.conflictPolicy),
		backPointerPath: filepath.Join(localAgents, worktreeBackPointerFile),
	}
	if !options.apply {
		return worktreeMigrationResult{status: "planned", message: "Dry run - re-run with --apply to perform the migration.", plan: plan}
	}

	if err := os.MkdirAll(mainAgents, 0o755); err != nil {
		return worktreeMigrationResult{status: "apply-error", message: err.Error(), plan: plan}
	}
	for _, move := range plan.moves {
		if err := applyWorktreeMove(move); err != nil {
			return worktreeMigrationResult{status: "apply-error", message: err.Error(), plan: plan}
		}
	}
	if err := os.MkdirAll(filepath.Dir(plan.backPointerPath), 0o755); err != nil {
		return worktreeMigrationResult{status: "apply-error", message: err.Error(), plan: plan}
	}
	if err := os.WriteFile(plan.backPointerPath, []byte(mainRoot+"\n"), 0o644); err != nil {
		return worktreeMigrationResult{status: "apply-error", message: err.Error(), plan: plan}
	}
	return worktreeMigrationResult{
		status:  "applied",
		message: fmt.Sprintf("Migrated %d file(s) to %s.", len(plan.moves), mainAgents),
		plan:    plan,
	}
}

// Only commands that consume canonical Markdown storage need the legacy gate.
// SQLite continuity and Git-authored configuration are separate authorities.
func worktreeCommandStoragePaths(args []string) []string {
	if len(args) < 2 {
		return nil
	}
	switch args[0] {
	case "task":
		switch args[1] {
		case "list", "show", "status", "refresh", "sync":
			return []string{"tasks", "specs", "TASKS.json"}
		}
	case "report":
		if args[1] == "generate" {
			if options, err := parseReportGenerateArgs(args[2:]); err == nil && options.kind == state.ExportKindReleaseReadiness {
				return []string{"reports"}
			}
		}
		switch args[1] {
		case "list", "show", "render", "create", "edit", "finalize", "archive":
			return []string{"reports"}
		}
	case "council":
		switch args[1] {
		case "new", "show", "list":
			return []string{"councils"}
		}
	case "search":
		if options, err := parseSearchArgs(args[1:]); err == nil && !options.allProjects {
			return []string{"reports"}
		}
	case "trace":
		return []string{"reports", "councils", "drafts"}
	case "render":
		if args[1] == "sweep" {
			return []string{"specs", "reports"}
		}
	case "migrate":
		if args[1] == "markdown" || args[1] == "vnext-rehearsal" {
			return legacyMarkdownStoragePaths()
		}
	case "state":
		if len(args) > 2 && args[1] == "export" && args[2] == "release-readiness" {
			return []string{"reports"}
		}
		if args[1] == "restore-ephemerals" || args[1] == "verify-ephemerals" {
			return legacyMarkdownStoragePaths()
		}
		if args[1] == "migrate" && len(args) > 2 && args[2] == "markdown" {
			return legacyMarkdownStoragePaths()
		}
	}
	return nil
}

func legacyMarkdownStoragePaths() []string {
	return []string{"tasks", "specs", "ideas", "sessions", "drafts", "reports", "sparks", "brainstorms", "TASKS.json"}
}

func worktreeCommandNeedsRoot(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return false
		}
	}
	if len(args) == 3 && args[0] == "journal" && args[1] == "context" && (args[2] == "for-prompt" || args[2] == "for-compact") {
		return false
	}
	switch args[0] {
	case "help", "version", "--version", "-v", "--agent-help", "build", "serve":
		return false
	case "auth":
		return len(args) > 1 && (args[1] == "attach" || args[1] == "link")
	case "harness":
		return projectEnvironmentActive()
	}
	return knownTopLevelCommandNative(args[0])
}

func shouldRefuseCommandNative(args []string, cwd string) bool {
	return (Runner{}).worktreeStorageRefusal(args, cwd) != ""
}

func (r Runner) worktreeStorageRefusal(args []string, cwd string) string {
	if !worktreeCommandNeedsRoot(args) {
		return ""
	}
	// Root identity remains necessary even for SQLite commands; never register
	// a missing main checkout as a new project rooted in the linked checkout.
	if missing := detectMainMissingForRefusalNative(cwd); missing != "" {
		return missing
	}
	paths := worktreeCommandStoragePaths(args)
	if args[0] == "housekeeping" {
		paths = []string{"reports", "drafts"}
		if root, err := project.ResolveRoot(cwd); err == nil {
			if status, err := state.Inspect(root, state.PathResolver{StateHome: r.StateHome}); err == nil && status.Mode == state.ModeMarkdownOnly {
				paths = nil
				for _, section := range markdownHousekeepingSectionSpecs {
					relative := strings.Split(filepath.ToSlash(section.relativeDir), "/")[0]
					if !slices.Contains(paths, relative) {
						paths = append(paths, relative)
					}
				}
			}
		}
	}
	if len(paths) == 0 {
		return ""
	}
	wtRoot := findWorktreeRootNative(cwd)
	mainRoot := findMainWorktreeRootNative(cwd)
	if wtRoot == "" || mainRoot == "" {
		return ""
	}
	localAgents := filepath.Join(wtRoot, ".agents")
	mainAgents := filepath.Join(mainRoot, ".agents")
	conflicts := worktreeStorageConflicts(localAgents, mainAgents, mainRoot, paths)
	if len(conflicts) == 0 {
		return ""
	}
	if args[0] == "task" {
		root, err := project.ResolveRoot(cwd)
		if err == nil {
			status, err := state.Inspect(root, state.PathResolver{StateHome: r.StateHome})
			// Dispatch owns invalid/unavailable SQLite diagnostics. Only the actual
			// Markdown fallback reads these files.
			if err != nil || status.Mode != state.ModeMarkdownOnly {
				return ""
			}
		}
	}
	return "This command consumes canonical storage that conflicts with this linked worktree:\n  " + strings.Join(conflicts, "\n  ") + "\n\n" + preA3RefusalMessageNative
}

// Classification never writes a back-pointer, changes sources, or follows
// symlinks. Git tracking is deliberately irrelevant: tracked legacy storage
// can still contain data absent from the canonical checkout.
func worktreeStorageConflicts(localAgents, mainAgents, mainRoot string, paths []string) []string {
	var conflicts []string
	for _, dir := range []string{localAgents, mainAgents} {
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return []string{fmt.Sprintf("%s: cannot inspect storage: %v", dir, err)}
		}
		if !info.IsDir() {
			return []string{dir + ": storage root is not a directory or is a symlink; reconcile manually"}
		}
	}
	pointerPath := filepath.Join(localAgents, worktreeBackPointerFile)
	if info, err := os.Lstat(pointerPath); err == nil {
		if !info.Mode().IsRegular() {
			return []string{pointerPath + ": unsupported migration marker; reconcile manually"}
		}
		pointer, err := os.ReadFile(pointerPath)
		if err != nil || strings.TrimSpace(string(pointer)) == "" || normalizePathForComparisonNative(strings.TrimSpace(string(pointer))) != normalizePathForComparisonNative(mainRoot) {
			return []string{pointerPath + ": invalid or stale migration target; reconcile with " + mainRoot}
		}
	} else if !os.IsNotExist(err) {
		return []string{fmt.Sprintf("%s: cannot read migration marker: %v", pointerPath, err)}
	}
	for _, rel := range paths {
		for _, dir := range []string{localAgents, mainAgents} {
			partial := filepath.Join(dir, rel+worktreePartialSuffix)
			if _, err := os.Lstat(partial); err == nil {
				conflicts = append(conflicts, partial+": interrupted migration staging path; recover manually before retrying")
			}
		}
		// Canonical-only staging paths and symlinks also make this storage
		// unsafe; they must not disappear merely because the local tree is empty.
		canonicalPath := filepath.Join(mainAgents, rel)
		err := filepath.WalkDir(canonicalPath, func(path string, entry fs.DirEntry, err error) error {
			if os.IsNotExist(err) && path == canonicalPath {
				return nil
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(entry.Name(), worktreePartialSuffix) {
				conflicts = append(conflicts, path+": interrupted migration staging path; recover manually before retrying")
				if entry.IsDir() {
					return filepath.SkipDir
				}
			} else if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
				conflicts = append(conflicts, path+": unsupported canonical storage file or symlink; reconcile manually")
			}
			return nil
		})
		if err != nil {
			conflicts = append(conflicts, fmt.Sprintf("%s: cannot inspect canonical storage: %v", canonicalPath, err))
		}
		localPath := filepath.Join(localAgents, rel)
		err = filepath.WalkDir(localPath, func(path string, entry fs.DirEntry, err error) error {
			if os.IsNotExist(err) && path == localPath {
				return nil
			}
			if err != nil {
				return err
			}
			if strings.HasSuffix(entry.Name(), worktreePartialSuffix) {
				conflicts = append(conflicts, path+": interrupted migration staging path; recover manually before retrying")
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
				conflicts = append(conflicts, path+": unsupported storage file or symlink; reconcile manually")
				return nil
			}
			relative, err := filepath.Rel(localAgents, path)
			if err != nil {
				return err
			}
			canonical := filepath.Join(mainAgents, relative)
			info, err := os.Lstat(canonical)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if info != nil && (info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) || info.IsDir() != entry.IsDir()) {
				conflicts = append(conflicts, canonical+": canonical storage type differs or is a symlink; reconcile manually")
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if os.IsNotExist(err) {
				conflicts = append(conflicts, path+": local-only storage (absent from "+canonical+")")
				return nil
			}
			if !filesHaveSameContent(path, canonical) {
				conflicts = append(conflicts, path+": divergent storage (differs from "+canonical+")")
			}
			return nil
		})
		if err != nil {
			conflicts = append(conflicts, fmt.Sprintf("%s: cannot inspect storage: %v", localPath, err))
		}
	}
	return conflicts
}

func unknownTopLevelCommandNative(args []string) string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") || args[0] == "help" {
		return ""
	}
	if knownTopLevelCommandNative(args[0]) {
		return ""
	}
	return args[0]
}

func knownTopLevelCommandNative(command string) bool {
	switch command {
	case "attach",
		"auth",
		"brainstorm",
		"build",
		"bundle",
		"check",
		"config",
		"conversation",
		"council",
		"doctor",
		"docs",
		"exploration",
		"handoff",
		"hooks",
		"housekeeping",
		"harness",
		"idea",
		"init",
		"install",
		"intake",
		"intent",
		"issue",
		"journal",
		"kb",
		"link",
		"migrate",
		"plan",
		"project",
		"release",
		"render",
		"report",
		"scratchpad",
		"search",
		"serve",
		"setup",
		"spark",
		"state",
		"sync",
		"tag",
		"task",
		"trace",
		"upgrade",
		"version":
		return true
	default:
		return false
	}
}

func detectMainMissingForRefusalNative(startDir string) string {
	wtRoot := findWorktreeRootNative(startDir)
	if wtRoot == "" {
		return ""
	}
	if mainRoot := findMainWorktreeRootNative(wtRoot); mainRoot != "" {
		exists, isDir := probeDirectory(mainRoot)
		if !exists || !isDir {
			return buildMainMissingMessageNative(mainRoot, exists)
		}
		return ""
	}
	pointerMainRoot := readGitdirPointerMainRootNative(wtRoot)
	if pointerMainRoot == "" {
		return ""
	}
	exists, isDir := probeDirectory(pointerMainRoot)
	if !exists || !isDir {
		return buildMainMissingMessageNative(pointerMainRoot, exists)
	}
	return ""
}

func formatWorktreeMigrationResult(result worktreeMigrationResult, apply bool) string {
	var out []string
	if result.plan != nil {
		mode := "(dry-run)"
		if apply {
			mode = "--apply"
		}
		out = append(out,
			fmt.Sprintf("loaf migrate worktree-storage %s", mode),
			"",
			fmt.Sprintf("  from %s", result.plan.worktreeAgents),
			fmt.Sprintf("  to   %s", result.plan.mainAgents),
			"",
		)
		if len(result.plan.moves) == 0 {
			out = append(out, "  ok Nothing to move.")
		} else {
			for _, move := range result.plan.moves {
				if move.conflict {
					out = append(out, fmt.Sprintf("  [conflict->%s] %s", strings.TrimPrefix(move.resolution, "keep-"), move.rel))
					out = append(out, fmt.Sprintf("      %s", move.resolutionReason))
				} else {
					out = append(out, fmt.Sprintf("  -> %s", move.rel))
				}
			}
		}
		out = append(out, "", fmt.Sprintf("  back-pointer: %s", result.plan.backPointerPath), "")
	}
	prefix := "->"
	if result.status == "applied" {
		prefix = "ok"
	}
	out = append(out, fmt.Sprintf("  %s %s", prefix, result.message))
	return strings.Join(out, "\n")
}

func findMainWorktreeRootNative(startDir string) string {
	gitDir, err := gitOutput(startDir, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		debugResolveNative(err)
		return ""
	}
	commonDir, err := gitOutput(startDir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		debugResolveNative(err)
		return ""
	}
	gitAbs := absoluteGitProbePath(startDir, strings.TrimSpace(gitDir))
	commonAbs := absoluteGitProbePath(startDir, strings.TrimSpace(commonDir))
	if normalizePathForComparisonNative(gitAbs) == normalizePathForComparisonNative(commonAbs) {
		return ""
	}
	commonCanonical := realpathOrSelfNative(commonAbs)
	if filepath.Base(commonCanonical) == ".git" {
		return filepath.Dir(commonCanonical)
	}
	return ""
}

func absoluteGitProbePath(startDir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(filepath.Join(startDir, path))
	if err != nil {
		return filepath.Join(startDir, path)
	}
	return abs
}

func gitOutput(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func debugResolveNative(err error) {
	if !debugResolveEnabledNative() {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: findMainWorktreeRoot fell back to parent-walk (error: %v)\n", debugResolveEnv, err)
}

func debugResolveEnabledNative() bool {
	switch strings.ToLower(os.Getenv(debugResolveEnv)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func normalizePathForComparisonNative(path string) string {
	return realpathOrSelfNative(path)
}

func realpathOrSelfNative(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func readGitdirPointerMainRootNative(wtRoot string) string {
	body, err := readRegularFile(filepath.Join(wtRoot, ".git"), projectFileReadLimit)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "gitdir:") {
			continue
		}
		gitdir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Clean(filepath.Join(wtRoot, gitdir))
		}
		needle := string(filepath.Separator) + ".git" + string(filepath.Separator) + "worktrees" + string(filepath.Separator)
		idx := strings.LastIndex(gitdir, needle)
		if idx <= 0 {
			return ""
		}
		return gitdir[:idx]
	}
	return ""
}

func probeDirectory(path string) (exists bool, isDirectory bool) {
	info, err := os.Stat(path)
	if err != nil {
		return false, false
	}
	return true, info.IsDir()
}

func buildMainMissingMessageNative(mainPath string, exists bool) string {
	cause := fmt.Sprintf("Main worktree at %s not found.", mainPath)
	if exists {
		cause = fmt.Sprintf("Main worktree at %s is not a directory.", mainPath)
	}
	return strings.Join([]string{
		cause,
		"",
		"The .agents/ migration target is unreachable. This usually means the main",
		"worktree was removed (`git worktree remove`) or its directory was deleted.",
		"Migration cannot proceed without a valid target.",
		"",
		"To resolve:",
		"- Restore the main worktree, OR",
		"- Check `git worktree list` and re-initialize the project layout you expect",
	}, "\n")
}

func findWorktreeRootNative(startDir string) string {
	current := startDir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func isInGitContextNative(startDir string) bool {
	return findWorktreeRootNative(startDir) != ""
}

func readBackPointerNative(worktreeAgentsDir string) string {
	body, err := os.ReadFile(filepath.Join(worktreeAgentsDir, worktreeBackPointerFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func worktreeAgentsHasContentNative(worktreeAgentsDir string) bool {
	files := enumerateWorktreeAgentFiles(worktreeAgentsDir)
	return len(files) > 0
}

func enumerateWorktreeAgentFiles(dir string) []string {
	var files []string
	var walk func(string, string)
	walk = func(current string, rel string) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for _, entry := range entries {
			relPath := entry.Name()
			if rel != "" {
				relPath = filepath.Join(rel, entry.Name())
			}
			if rel == "" && entry.Name() == worktreeBackPointerFile {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				files = append(files, filepath.ToSlash(relPath))
				continue
			}
			if entry.IsDir() {
				walk(filepath.Join(current, entry.Name()), relPath)
				continue
			}
			if entry.Type().IsRegular() {
				files = append(files, filepath.ToSlash(relPath))
			}
		}
	}
	walk(dir, "")
	sort.Strings(files)
	return files
}

func findWorktreeSymlinkPaths(dir string) []string {
	var paths []string
	var walk func(string, string)
	walk = func(current string, rel string) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for _, entry := range entries {
			relPath := entry.Name()
			if rel != "" {
				relPath = filepath.Join(rel, entry.Name())
			}
			if rel == "" && entry.Name() == worktreeBackPointerFile {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				paths = append(paths, filepath.ToSlash(relPath))
				continue
			}
			if entry.IsDir() {
				walk(filepath.Join(current, entry.Name()), relPath)
			}
		}
	}
	walk(dir, "")
	sort.Strings(paths)
	return paths
}

func findWorktreePartialPaths(dir string) []string {
	var paths []string
	var walk func(string)
	walk = func(current string) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for _, entry := range entries {
			abs := filepath.Join(current, entry.Name())
			if strings.HasSuffix(entry.Name(), worktreePartialSuffix) {
				paths = append(paths, abs)
				continue
			}
			if entry.IsDir() {
				walk(abs)
			}
		}
	}
	walk(dir)
	sort.Strings(paths)
	return paths
}

func planWorktreeMoves(worktreeAgents string, mainAgents string, policy worktreeConflictPolicy) []worktreeMigrationMove {
	relFiles := enumerateWorktreeAgentFiles(worktreeAgents)
	moves := make([]worktreeMigrationMove, 0, len(relFiles))
	for _, rel := range relFiles {
		from := filepath.Join(worktreeAgents, filepath.FromSlash(rel))
		to := filepath.Join(mainAgents, filepath.FromSlash(rel))
		move := worktreeMigrationMove{from: from, to: to, rel: rel}
		if _, err := os.Stat(to); err == nil {
			move.conflict = true
			switch {
			case filesHaveSameContent(from, to):
				move.resolution = "keep-main"
				move.resolutionReason = "identical content"
			case policy == worktreeConflictWorktree:
				move.resolution = "keep-worktree"
				move.resolutionReason = "forced by --force-from-worktree"
			case policy == worktreeConflictMain:
				move.resolution = "keep-main"
				move.resolutionReason = "forced by --force-from-main"
			default:
				fromMtime := fileModTimeUnixNano(from)
				toMtime := fileModTimeUnixNano(to)
				if fromMtime > toMtime {
					move.resolution = "keep-worktree"
					move.resolutionReason = "worktree mtime > main mtime"
				} else {
					move.resolution = "keep-main"
					move.resolutionReason = "main mtime >= worktree mtime"
				}
			}
		}
		moves = append(moves, move)
	}
	return moves
}

func filesHaveSameContent(a string, b string) bool {
	left, err := readRegularFile(a, projectFileReadLimit)
	if err != nil {
		return false
	}
	right, err := readRegularFile(b, projectFileReadLimit)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}

func fileModTimeUnixNano(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano()
}

func applyWorktreeMove(move worktreeMigrationMove) error {
	if move.conflict {
		if move.resolution == "keep-main" {
			return os.Remove(move.from)
		}
		if err := os.Remove(move.to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(move.to), 0o755); err != nil {
		return err
	}
	if err := os.Rename(move.from, move.to); err != nil {
		if !isCrossDeviceRename(err) {
			return err
		}
		partial := move.to + worktreePartialSuffix
		if err := os.RemoveAll(partial); err != nil {
			return err
		}
		if err := copyFilePreservingMode(move.from, partial); err != nil {
			return err
		}
		if err := os.Rename(partial, move.to); err != nil {
			return err
		}
		if err := os.Remove(move.from); err != nil {
			return err
		}
	}
	return nil
}

func isCrossDeviceRename(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return errors.Is(linkErr.Err, syscall.EXDEV)
	}
	return false
}

func copyFilePreservingMode(from string, to string) error {
	// Verify regular-file-ness through the descriptor-checked non-blocking
	// open before copying. There is no size cap on the copy itself — the type
	// check is the point — so a non-regular entry is refused with a reported
	// skip rather than hanging the open on a FIFO.
	source, err := openRegularFile(from)
	if err != nil {
		return err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	target, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return err
	}
	return target.Close()
}
