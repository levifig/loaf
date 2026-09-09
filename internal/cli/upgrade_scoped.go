package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

var scopedApplyTestFault func(phase string) error

func scopedApplyFault(phase string) error {
	if scopedApplyTestFault == nil {
		return nil
	}
	return scopedApplyTestFault(phase)
}

func (r Runner) runScopedUpgrade(options upgradeOptions, out io.Writer, runtimeRoot string) error {
	refs := options.selections
	if len(refs) == 0 {
		return fmt.Errorf("scoped upgrade requires --select target/id")
	}
	loafRoot, err := r.resolveInstalledDistributionRoot()
	if err != nil {
		return err
	}
	projectRoot, err := project.ResolveRoot(runtimeRoot)
	if err != nil {
		return err
	}
	version := packageVersion(loafRoot)
	distRoot := filepath.Join(loafRoot, "dist")
	tools := detectInstallTools()
	hasClaudeCode := installCommandExists("claude")
	planOptions := options.installPlanOptions()
	assumeYes := installAssumeYes(planOptions)
	plan, err := r.buildScopedUpgradePlan(planOptions, loafRoot, projectRoot.Path(), version, distRoot, tools, hasClaudeCode, assumeYes, refs)
	if err != nil {
		return err
	}
	if options.dryRun {
		if options.json {
			return emitInstallDryRunJSON(out, plan)
		}
		writeInstallDryRunHuman(out, plan)
		return nil
	}
	if err := selectedConflictsError(plan); err != nil {
		return err
	}
	return r.applyScopedUpgrade(out, plan, planOptions, loafRoot, projectRoot.Path(), version, distRoot, tools, refs)
}

func (r Runner) buildScopedUpgradePlan(options installOptions, loafRoot string, projectRoot string, version string, distRoot string, tools []detectedInstallTool, hasClaudeCode bool, assumeYes bool, refs []scopedArtifactRef) (installDryRunPlan, error) {
	options.selections = refs
	plan, err := r.buildInstallDryRunPlan(options, loafRoot, projectRoot, version, distRoot, tools, hasClaudeCode, assumeYes, nil)
	if err != nil {
		return installDryRunPlan{}, err
	}
	plan, err = filterPlanToSelections(plan, refs)
	if err != nil {
		return installDryRunPlan{}, err
	}
	plan.VersionStamp = "none"
	plan.FollowUpCommands = []string{scopedFollowUpCommand(refs)}
	checkedSkillStores := map[string]bool{}
	for _, skill := range plan.Skills {
		dest := filepath.Dir(skill.Destination)
		if checkedSkillStores[dest] {
			continue
		}
		checkedSkillStores[dest] = true
		previous, err := readManagedSkillsState(dest)
		if err != nil {
			return installDryRunPlan{}, err
		}
		if err := validateSelectedLegacySkills(previous, selectedSkillNames(refs)); err != nil {
			return installDryRunPlan{}, fmt.Errorf("skills at %s: %w", dest, err)
		}
	}
	hookState, releaseHookState := r.hookStateForPlan(projectRoot)
	defer releaseHookState()
	if err := enrichScopedPlanDiffs(&plan, hookState, projectRoot, version, distRoot, tools, refs); err != nil {
		return installDryRunPlan{}, err
	}
	if err := scopedRetirementDependenciesError(plan, refs, hookState, projectRoot, version, distRoot, tools); err != nil {
		return installDryRunPlan{}, err
	}
	if err := scopedPathLoafPreflightError(plan); err != nil {
		return installDryRunPlan{}, err
	}
	return plan, nil
}

func selectedConflictsError(plan installDryRunPlan) error {
	var conflicts []string
	for _, skill := range plan.Skills {
		if skill.Action == planActionConflict {
			conflicts = append(conflicts, scopedSkillsTarget+"/"+skill.ID)
		}
	}
	for _, target := range plan.Targets {
		for _, artifact := range target.Artifacts {
			if artifact.Action == planActionConflict {
				conflict := target.Target + "/" + artifact.ID
				if artifact.Detail != "" {
					conflict += ": " + artifact.Detail
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("selected conflict(s) refuse application: %s", strings.Join(conflicts, ", "))
}

func (r Runner) applyScopedUpgrade(out io.Writer, planned installDryRunPlan, options installOptions, loafRoot string, projectRoot string, version string, distRoot string, tools []detectedInstallTool, refs []scopedArtifactRef) error {
	lock, err := acquireHarnessReconcileLock(managedContentLockDir(installLayoutHome(projectRoot)), managedContentLockWait)
	if err != nil {
		return err
	}
	defer lock.release()

	hasClaudeCode := installCommandExists("claude")
	assumeYes := installAssumeYes(options)
	rechecked, err := r.buildScopedUpgradePlan(options, loafRoot, projectRoot, version, distRoot, tools, hasClaudeCode, assumeYes, refs)
	if err != nil {
		return err
	}
	if err := selectedConflictsError(rechecked); err != nil {
		return err
	}
	if err := scopedPlanDriftError(planned, rechecked); err != nil {
		return err
	}

	txn := newScopedTxn()
	defer func() {
		if txn.failed != nil {
			_ = txn.rollback()
		}
	}()

	defaults, layoutHome := resolveInstallLayout(projectRoot)
	toolByKey := installToolsByKey(tools)
	hookState, releaseHookState := r.hookStateForApply(projectRoot)
	defer releaseHookState()

	skillNames := selectedSkillNames(refs)
	if len(skillNames) > 0 {
		groups, err := groupSkillsInstallByDestination(scopedTargetOptions(refs, defaults, toolByKey, distRoot, version, layoutHome, projectRoot, hookState))
		if err != nil {
			txn.failed = err
			return err
		}
		for _, group := range groups {
			source, err := selectCanonicalSkillsSource(group.Options)
			if err != nil {
				txn.failed = err
				return err
			}
			src := filepath.Join(source.DistDir, "skills")
			if err := txn.backup(filepath.Join(group.Destination, loafSkillManifestFile)); err != nil {
				txn.failed = err
				return err
			}
			for _, name := range skillNames {
				if err := txn.backup(filepath.Join(group.Destination, name)); err != nil {
					txn.failed = err
					return err
				}
			}
			if err := syncSelectedManagedSkills(src, group.Destination, skillNames); err != nil {
				txn.failed = err
				return err
			}
			if err := scopedApplyFault("skills"); err != nil {
				txn.failed = err
				return err
			}
		}
	}

	for _, targetPlan := range rechecked.Targets {
		opts := targetInstallOptions{
			Target:          targetPlan.Target,
			DistDir:         filepath.Join(distRoot, targetPlan.Target),
			ConfigDir:       targetPlan.ConfigDir,
			Upgrade:         true,
			Version:         version,
			HomeDir:         layoutHome,
			CodexHome:       resolveInstallCodexHome(targetPlan.ConfigDir),
			ProjectRoot:     projectRoot,
			SkipSkillsSync:  true,
			HookState:       hookState,
			SelectedHookIDs: selectedHookIDsForTarget(refs, targetPlan.Target),
		}
		adapterIDs := selectedAdapterIDsForTarget(refs, targetPlan.Target)
		if len(adapterIDs) > 0 {
			if err := txn.backup(filepath.Join(opts.ConfigDir, targetInstallManifestFile)); err != nil {
				txn.failed = err
				return err
			}
			for _, decision := range targetPlan.Artifacts {
				if strings.HasPrefix(decision.ID, "hook:") || isCodexPolicyDecision(decision) {
					continue
				}
				path := decision.Destination
				if !filepath.IsAbs(path) {
					path = filepath.Join(opts.ConfigDir, filepath.FromSlash(path))
				}
				if decision.Kind == "hook-legacy" || filepath.Clean(path) == filepath.Clean(opts.ConfigDir) {
					err := fmt.Errorf("refusing whole-target snapshot for scoped artifact %s/%s", targetPlan.Target, decision.ID)
					txn.failed = err
					return err
				}
				if err := txn.backup(path); err != nil {
					txn.failed = err
					return err
				}
			}
			if err := syncSelectedTargetAdapterArtifacts(opts, adapterIDs); err != nil {
				txn.failed = err
				return err
			}
			if err := scopedApplyFault(targetPlan.Target + "-adapters"); err != nil {
				txn.failed = err
				return err
			}
		}
		policyDecisions := selectedCodexPolicyDecisions(targetPlan.Artifacts)
		if scopedPolicyDecisionsNeedApply(policyDecisions) {
			if err := txn.backup(codexPolicyOwnershipManifestPath(opts)); err != nil {
				txn.failed = err
				return err
			}
			for _, decision := range policyDecisions {
				if !scopedPolicyActionWrites(decision.Action) {
					continue
				}
				path, err := resolveDecisionPath(opts, decision)
				if err != nil {
					txn.failed = err
					return err
				}
				if err := txn.backup(path); err != nil {
					txn.failed = err
					return err
				}
			}
			if err := publishSelectedCodexPolicyArtifacts(opts, policyDecisions); err != nil {
				txn.failed = err
				return err
			}
			if err := scopedApplyFault(targetPlan.Target + "-policy"); err != nil {
				txn.failed = err
				return err
			}
		}
		if opts.SelectedHookIDs != nil {
			if err := txn.backup(targetHookFilePath(opts)); err != nil {
				txn.failed = err
				return err
			}
			if err := applySelectedHookEntries(opts); err != nil {
				txn.failed = err
				return err
			}
			if err := scopedApplyFault(targetPlan.Target + "-hooks"); err != nil {
				txn.failed = err
				return err
			}
		}
	}

	if err := txn.cleanup(); err != nil {
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, ansiBold("loaf upgrade --select"))
	fmt.Fprintln(out)
	for _, skill := range rechecked.Skills {
		if skill.Action == planActionPreserve || skill.Action == planActionNone {
			continue
		}
		fmt.Fprintf(out, "  %s %s %s\n", planActionGlyph(skill.Action), skill.Action, skill.ID)
	}
	for _, target := range rechecked.Targets {
		for _, artifact := range target.Artifacts {
			if artifact.Action == planActionPreserve || artifact.Action == planActionNone {
				continue
			}
			fmt.Fprintf(out, "  %s %s %s/%s\n", planActionGlyph(artifact.Action), artifact.Action, target.Target, artifact.ID)
		}
		if fileExistsForInstall(filepath.Join(target.ConfigDir, loafInstallMarkerFile)) {
			fmt.Fprintf(out, "  %s %s .loaf-version left unchanged\n", ansiGray("○"), installDisplayName(target.Target))
		}
	}
	fmt.Fprintln(out)
	return nil
}

func scopedPlanDriftError(before installDryRunPlan, after installDryRunPlan) error {
	if summarizeScopedPlan(before) != summarizeScopedPlan(after) {
		return fmt.Errorf("selected artifacts changed while waiting for the reconcile lock; rerun --dry-run")
	}
	return nil
}

func summarizeScopedPlan(plan installDryRunPlan) string {
	var b strings.Builder
	for _, skill := range plan.Skills {
		fmt.Fprintf(&b, "S %s %s %s %s\n", skill.ID, skill.Action, skill.LiveSHA256, skill.DesiredSHA256)
	}
	for _, target := range plan.Targets {
		for _, artifact := range target.Artifacts {
			fmt.Fprintf(&b, "T %s %s %s %s %s\n", target.Target, artifact.ID, artifact.Action, artifact.LiveSHA256, artifact.DesiredSHA256)
		}
	}
	return b.String()
}

func scopedTargetOptions(refs []scopedArtifactRef, defaults map[string]string, toolByKey map[string]detectedInstallTool, distRoot string, version string, home string, projectRoot string, hookState hookStateResolver) []targetInstallOptions {
	seen := map[string]bool{}
	var options []targetInstallOptions
	for _, ref := range refs {
		if ref.Target == scopedSkillsTarget {
			continue
		}
		if seen[ref.Target] {
			continue
		}
		seen[ref.Target] = true
		configDir := defaults[ref.Target]
		if tool, ok := toolByKey[ref.Target]; ok && tool.configDir != "" {
			configDir = tool.configDir
		}
		options = append(options, targetInstallOptions{
			Target:         ref.Target,
			DistDir:        filepath.Join(distRoot, ref.Target),
			ConfigDir:      configDir,
			Upgrade:        true,
			Version:        version,
			HomeDir:        home,
			CodexHome:      resolveInstallCodexHome(configDir),
			ProjectRoot:    projectRoot,
			SkipSkillsSync: true,
			HookState:      hookState,
		})
	}
	if len(options) == 0 {
		// Skills-only selections still need a destination group. Use cursor if
		// present, otherwise the first installed target, so the shared store
		// resolves the same way upgrade does.
		for _, target := range []string{"cursor", "opencode", "codex", "amp"} {
			configDir := defaults[target]
			if tool, ok := toolByKey[target]; ok && tool.configDir != "" {
				configDir = tool.configDir
			}
			if configDir == "" {
				continue
			}
			options = append(options, targetInstallOptions{
				Target:         target,
				DistDir:        filepath.Join(distRoot, target),
				ConfigDir:      configDir,
				Upgrade:        true,
				Version:        version,
				HomeDir:        home,
				ProjectRoot:    projectRoot,
				SkipSkillsSync: true,
				HookState:      hookState,
			})
			break
		}
	}
	return options
}

func applySelectedHookEntries(options targetInstallOptions) error {
	reconciler, err := newHookReconciler(options)
	if err != nil {
		return err
	}
	if reconciler == nil {
		return nil
	}
	_, err = reconciler.applySelected(context.Background())
	return err
}

func (r *hookReconciler) applySelected(ctx context.Context) ([]hookAction, error) {
	if err := r.beginSelected(ctx); err != nil {
		return nil, err
	}
	return r.complete(ctx)
}

func (r *hookReconciler) beginSelected(ctx context.Context) error {
	lock, err := acquireHookFileLock(r.path, r.lockWait)
	if err != nil {
		return err
	}
	r.lock = lock
	store, err := r.openState()
	if err != nil {
		_ = r.release()
		return err
	}
	if store == nil {
		_ = r.release()
		return fmt.Errorf("cannot reconcile %s: hook enablement state is unavailable", r.path)
	}
	if err := r.recordTrustedExecutable(ctx, store); err != nil {
		_ = r.release()
		return err
	}
	return nil
}

type scopedTxn struct {
	backups []scopedBackup
	failed  error
}

type scopedBackup struct {
	path    string
	existed bool
	isDir   bool
	body    []byte
	mode    fs.FileMode
	dirCopy string
}

func newScopedTxn() *scopedTxn {
	return &scopedTxn{}
}

func (t *scopedTxn) backup(path string) error {
	if t == nil || path == "" {
		return nil
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		t.backups = append(t.backups, scopedBackup{path: path})
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to snapshot symlink %s", path)
	}
	if info.IsDir() {
		dirCopy, err := os.MkdirTemp(filepath.Dir(path), "."+filepath.Base(path)+".loaf-scoped-bak-")
		if err != nil {
			return err
		}
		if err := copyDirContentsForInstall(path, dirCopy); err != nil {
			_ = os.RemoveAll(dirCopy)
			return err
		}
		t.backups = append(t.backups, scopedBackup{path: path, existed: true, isDir: true, dirCopy: dirCopy, mode: info.Mode().Perm()})
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	t.backups = append(t.backups, scopedBackup{path: path, existed: true, body: body, mode: info.Mode().Perm()})
	return nil
}

func (t *scopedTxn) rollback() error {
	if t == nil {
		return nil
	}
	var failures []string
	for i := len(t.backups) - 1; i >= 0; i-- {
		backup := t.backups[i]
		if err := backup.restore(); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("scoped apply rollback failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (t *scopedTxn) cleanup() error {
	if t == nil {
		return nil
	}
	var failures []string
	for _, backup := range t.backups {
		if backup.dirCopy == "" {
			continue
		}
		if err := os.RemoveAll(backup.dirCopy); err != nil {
			failures = append(failures, err.Error())
		}
		backup.dirCopy = ""
	}
	if len(failures) > 0 {
		return fmt.Errorf("scoped apply backup cleanup failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (b scopedBackup) restore() error {
	if !b.existed {
		if err := os.RemoveAll(b.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if b.isDir {
		if err := os.RemoveAll(b.path); err != nil {
			return err
		}
		if err := os.MkdirAll(b.path, b.mode); err != nil {
			return err
		}
		err := copyDirContentsForInstall(b.dirCopy, b.path)
		_ = os.RemoveAll(b.dirCopy)
		return err
	}
	return writeFileAtomically(b.path, b.body, b.mode)
}

func scopedRetirementDependenciesError(plan installDryRunPlan, refs []scopedArtifactRef, hookState hookStateResolver, projectRoot string, version string, distRoot string, tools []detectedInstallTool) error {
	selected := map[string]bool{}
	for _, ref := range refs {
		selected[ref.String()] = true
	}
	var missing []string
	seenMissing := map[string]bool{}
	addMissing := func(id string) {
		if id == "" || selected[id] || seenMissing[id] {
			return
		}
		seenMissing[id] = true
		missing = append(missing, id)
	}
	defaults, layoutHome := resolveInstallLayout(projectRoot)
	toolByKey := installToolsByKey(tools)
	for _, targetPlan := range plan.Targets {
		var retiring []artifactPlanDecision
		for _, artifact := range targetPlan.Artifacts {
			if artifact.Action == planActionRetire && strings.HasPrefix(artifact.ID, "hook-file:") {
				retiring = append(retiring, artifact)
			}
		}
		if len(retiring) == 0 {
			continue
		}
		configDir := targetPlan.ConfigDir
		if configDir == "" {
			configDir = defaults[targetPlan.Target]
			if tool, ok := toolByKey[targetPlan.Target]; ok && tool.configDir != "" {
				configDir = tool.configDir
			}
		}
		opts := targetInstallOptions{
			Target:          targetPlan.Target,
			DistDir:         filepath.Join(distRoot, targetPlan.Target),
			ConfigDir:       configDir,
			Upgrade:         true,
			Version:         version,
			HomeDir:         layoutHome,
			CodexHome:       resolveInstallCodexHome(configDir),
			ProjectRoot:     projectRoot,
			SkipSkillsSync:  true,
			HookState:       hookState,
			SelectedHookIDs: selectedHookIDsForTarget(refs, targetPlan.Target),
		}
		paths := make([]string, 0, len(retiring))
		relPaths := make([]string, 0, len(retiring))
		for _, artifact := range retiring {
			rel := strings.TrimPrefix(artifact.ID, "hook-file:")
			if artifact.Destination != "" {
				rel = artifact.Destination
			}
			relPaths = append(relPaths, rel)
			path := artifact.Destination
			if path == "" {
				path = rel
			}
			if !filepath.IsAbs(path) {
				abs, err := targetAdapterDestination(opts, targetAdapterArtifact{ID: artifact.ID, Kind: "hook-file", Destination: rel})
				if err != nil {
					return err
				}
				path = abs
			}
			paths = append(paths, path)
		}
		if err := collectHookFileCompanionIDs(opts, targetPlan.Target, relPaths, paths, addMissing); err != nil {
			return err
		}
		if err := collectPluginCompanionIDs(opts, targetPlan.Target, selected, relPaths, paths, addMissing); err != nil {
			return err
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("incomplete retirement selection: live callers remain; also select %s", strings.Join(missing, ", "))
}

// collectHookFileCompanionIDs inspects every hook entry that would remain after
// this apply, including entries computeProjection would add when the live hooks
// file is absent. Selection membership is not enough: a selected companion is
// safe only when its projected command no longer names a retiring file, and a
// preserved foreign caller that still names one blocks retirement without being
// claimed or rewritten.
func collectHookFileCompanionIDs(opts targetInstallOptions, target string, relPaths []string, paths []string, add func(string)) error {
	reconciler, err := newHookReconciler(opts)
	if err != nil || reconciler == nil {
		return err
	}
	records, err := loadHookRecordsForDiff(opts, reconciler)
	if err != nil {
		return err
	}
	if reconciler.selectedIDs == nil {
		// Apply will not rewrite the hooks file. Project that fact rather than
		// simulating a full reconcile that would replace unselected hooks.
		reconciler.selectedIDs = map[string]bool{}
	}
	projection, err := reconciler.computeProjection(records)
	if err != nil {
		return err
	}
	recognition := reconciler.recognition(records)
	needles := hookFileReferenceNeedles(paths, relPaths)
	var selectedBroken []string
	seenSelected := map[string]bool{}
	preservedCaller := false
	for _, event := range reconciler.reconciledEvents(projection.file) {
		entries, err := projection.file.eventEntries(event)
		if err != nil {
			return err
		}
		outcome, err := pairHookEventEntries(recognition, event, entries)
		if err != nil {
			return err
		}
		foreign := make(map[int]bool, len(outcome.foreign))
		for _, index := range outcome.foreign {
			foreign[index] = true
		}
		ownedID := make(map[int]string, len(outcome.paired)+len(outcome.duplicates))
		for _, pairing := range outcome.paired {
			ownedID[pairing.index] = pairing.hookID
		}
		for _, pairing := range outcome.duplicates {
			ownedID[pairing.index] = pairing.hookID
		}
		for index, entry := range entries {
			if !projectedHookEntryReferencesFiles(entry, needles) {
				continue
			}
			if foreign[index] {
				preservedCaller = true
				continue
			}
			if hookID := ownedID[index]; hookID != "" {
				companion := target + "/hook:" + event + "/" + hookID
				if opts.SelectedHookIDs["hook:"+event+"/"+hookID] {
					if !seenSelected[companion] {
						seenSelected[companion] = true
						selectedBroken = append(selectedBroken, companion)
					}
					continue
				}
				add(companion)
				continue
			}
			if len(paths) > 0 {
				add(target + "/hook:" + event + "/" + filepath.Base(paths[0]))
			}
		}
	}
	if preservedCaller {
		return fmt.Errorf("incomplete retirement selection: preserved hook entries still reference retiring hook files")
	}
	if len(selectedBroken) > 0 {
		sort.Strings(selectedBroken)
		return fmt.Errorf("incomplete retirement selection: selected %s still references retiring hook files", strings.Join(selectedBroken, ", "))
	}
	return nil
}

func projectedHookEntryReferencesFiles(entry map[string]any, needles []string) bool {
	body, err := json.Marshal(entry)
	if err != nil {
		return false
	}
	return adapterBodyReferencesHookFiles(body, needles)
}

func collectPluginCompanionIDs(opts targetInstallOptions, target string, selected map[string]bool, relPaths []string, absPaths []string, add func(string)) error {
	installed, ok, err := readInstalledHookManifest(opts)
	if err != nil || !ok {
		return err
	}
	needles := hookFileReferenceNeedles(absPaths, relPaths)
	for _, artifact := range installed.Artifacts {
		if artifact.Kind != "plugin" {
			continue
		}
		companion := target + "/" + artifact.ID
		path, err := targetAdapterDestination(opts, artifact)
		if err != nil {
			return err
		}
		if selected[companion] {
			desired, _, err := desiredAdapterContent(opts, artifactPlanDecision{ID: artifact.ID})
			if err != nil {
				return err
			}
			if adapterBodyReferencesHookFiles(desired, needles) {
				return fmt.Errorf("incomplete retirement selection: selected %s still references retiring hook files", companion)
			}
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if adapterBodyReferencesHookFiles(body, needles) {
			add(companion)
		}
	}
	return nil
}

func hookFileReferenceNeedles(absPaths []string, relPaths []string) []string {
	seen := map[string]bool{}
	var needles []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		needles = append(needles, value)
	}
	for _, path := range absPaths {
		add(path)
		add(filepath.ToSlash(path))
		add(filepath.Base(path))
	}
	for _, rel := range relPaths {
		slash := filepath.ToSlash(rel)
		add(slash)
		add(filepath.Base(slash))
		if idx := strings.Index(slash, "hooks/"); idx >= 0 {
			add(slash[idx:])
		}
	}
	return needles
}

func adapterBodyReferencesHookFiles(body []byte, needles []string) bool {
	for _, needle := range needles {
		if needle != "" && bytes.Contains(body, []byte(needle)) {
			return true
		}
	}
	return false
}

func desiredAdapterBody(options targetInstallOptions, artifact targetAdapterArtifact) ([]byte, error) {
	path := filepath.Join(options.DistDir, filepath.FromSlash(artifact.SourcePath))
	return os.ReadFile(path)
}

func enrichScopedPlanDiffs(plan *installDryRunPlan, hookState hookStateResolver, projectRoot string, version string, distRoot string, tools []detectedInstallTool, refs []scopedArtifactRef) error {
	defaults, layoutHome := resolveInstallLayout(projectRoot)
	toolByKey := installToolsByKey(tools)
	for i, skill := range plan.Skills {
		name, _ := strings.CutPrefix(skill.ID, "skill:")
		src := scopedSkillSource(refs, defaults, toolByKey, distRoot, version, layoutHome, projectRoot, name)
		dest := skill.Destination
		diff, liveHash, desiredHash, err := skillTreeDiff(src, dest)
		if err != nil {
			return err
		}
		plan.Skills[i].Diff = diff
		plan.Skills[i].LiveSHA256 = liveHash
		plan.Skills[i].DesiredSHA256 = desiredHash
		desiredText, err := desiredTextFromTree(src)
		if err != nil {
			return err
		}
		plan.Skills[i].Desired = desiredText
	}
	for i, target := range plan.Targets {
		opts := targetInstallOptions{
			Target:          target.Target,
			DistDir:         filepath.Join(distRoot, target.Target),
			ConfigDir:       target.ConfigDir,
			Upgrade:         true,
			Version:         version,
			HomeDir:         layoutHome,
			CodexHome:       resolveInstallCodexHome(target.ConfigDir),
			ProjectRoot:     projectRoot,
			SkipSkillsSync:  true,
			HookState:       hookState,
			SelectedHookIDs: selectedHookIDsForTarget(refs, target.Target),
		}
		for j, artifact := range target.Artifacts {
			diff, liveHash, desiredHash, err := scopedArtifactDiff(opts, artifact)
			if err != nil {
				return err
			}
			plan.Targets[i].Artifacts[j].Diff = diff
			plan.Targets[i].Artifacts[j].LiveSHA256 = liveHash
			plan.Targets[i].Artifacts[j].DesiredSHA256 = desiredHash
			desiredText, err := scopedArtifactDesiredText(opts, artifact)
			if err != nil {
				return err
			}
			plan.Targets[i].Artifacts[j].Desired = desiredText
		}
	}
	return nil
}

func scopedArtifactDesiredText(options targetInstallOptions, decision artifactPlanDecision) (string, error) {
	if !planActionWritesDesiredContent(decision.Action) {
		return "", nil
	}
	if strings.HasPrefix(decision.ID, "hook:") {
		return hookEntryDesiredText(options, decision)
	}
	body, _, err := desiredScopedArtifactContent(options, decision, "")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func hookEntryDesiredText(options targetInstallOptions, decision artifactPlanDecision) (string, error) {
	reconciler, err := newHookReconciler(options)
	if err != nil || reconciler == nil {
		return "", err
	}
	event, hookID, ok := strings.Cut(strings.TrimPrefix(decision.ID, "hook:"), "/")
	if !ok {
		return "", nil
	}
	for _, entry := range reconciler.catalog.Entries {
		if entry.Event != event || entry.HookID != hookID {
			continue
		}
		desired, err := reconciler.desiredEntry(entry)
		if err != nil {
			return "", err
		}
		return prettyJSONBytes(desired), nil
	}
	return "", nil
}

func desiredTextFromTree(root string) (string, error) {
	if root == "" {
		return "", nil
	}
	if _, err := os.Lstat(root); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(body) {
			return nil
		}
		b.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			b.WriteByte('\n')
		}
		return nil
	})
	return b.String(), err
}
func scopedSkillSource(refs []scopedArtifactRef, defaults map[string]string, toolByKey map[string]detectedInstallTool, distRoot string, version string, home string, projectRoot string, skill string) string {
	options := scopedTargetOptions(refs, defaults, toolByKey, distRoot, version, home, projectRoot, nil)
	if len(options) == 0 {
		return ""
	}
	source, err := selectCanonicalSkillsSource(options)
	if err != nil {
		return filepath.Join(options[0].DistDir, "skills", skill)
	}
	return filepath.Join(source.DistDir, "skills", skill)
}

func skillTreeDiff(src string, dest string) (string, string, string, error) {
	desiredHash, err := hashInstallSkillTree(src)
	if err != nil && !os.IsNotExist(err) {
		return "", "", "", err
	}
	if os.IsNotExist(err) {
		desiredHash = ""
	}
	liveHash, err := hashInstallSkillTree(dest)
	if err != nil && !os.IsNotExist(err) {
		return "", "", "", err
	}
	if os.IsNotExist(err) {
		liveHash = ""
	}
	diff, err := directoryUnifiedDiff(dest, src)
	if err != nil {
		return "", liveHash, desiredHash, err
	}
	return diff, liveHash, desiredHash, nil
}

func scopedArtifactDiff(options targetInstallOptions, decision artifactPlanDecision) (string, string, string, error) {
	if decision.Kind == "codex-guidance" && decision.Action == planActionPreserve {
		// Preserved global instructions may be symlinked or unreadable. They
		// are neither a publication target nor a dependency of rule updates.
		return "", "", "", nil
	}
	if strings.HasPrefix(decision.ID, "hook:") {
		return hookEntryDiff(options, decision)
	}
	path, err := resolveDecisionPath(options, decision)
	if err != nil {
		return "", "", "", err
	}
	if path == "" || path == options.ConfigDir {
		return "", "", "", nil
	}
	info, statErr := os.Lstat(path)
	if statErr == nil && info.IsDir() {
		return "", "", "", nil
	}
	liveSnap, err := readTargetAdapterSnapshot(path)
	if err != nil {
		return "", "", "", err
	}
	liveHash := ""
	liveBody := ""
	if liveSnap.exists {
		liveHash = sha256Bytes(liveSnap.body)
		liveBody = string(liveSnap.body)
	}
	if decision.Action == planActionRetire {
		if decision.Kind == "codex-guidance" {
			text := liveBody
			if guidanceRange, ok := findCodexJournalGuidance(text); ok {
				text = removeCodexJournalGuidance(text, guidanceRange)
			}
			return unifiedDiff(path, liveBody, text), liveHash, sha256Bytes([]byte(text)), nil
		}
		return unifiedDiff(path, liveBody, ""), liveHash, "", nil
	}
	if isCodexPolicyDecision(decision) && !scopedPolicyActionWrites(decision.Action) {
		return "", liveHash, liveHash, nil
	}
	desired, desiredHash, err := desiredScopedArtifactContent(options, decision, liveBody)
	if err != nil {
		return "", liveHash, "", err
	}
	return unifiedDiff(path, liveBody, string(desired)), liveHash, desiredHash, nil
}

func desiredScopedArtifactContent(options targetInstallOptions, decision artifactPlanDecision, liveBody string) ([]byte, string, error) {
	if decision.Kind == "codex-rule" || decision.Kind == "codex-guidance" {
		body, err := desiredCodexPolicyContent(options, decision, liveBody)
		if err != nil {
			return nil, "", err
		}
		return body, sha256Bytes(body), nil
	}
	return desiredAdapterContent(options, decision)
}

func desiredCodexPolicyContent(options targetInstallOptions, decision artifactPlanDecision, liveBody string) ([]byte, error) {
	switch decision.Kind {
	case "codex-rule":
		templatePath := filepath.Join(options.DistDir, ".codex", "rules", codexJournalRuleTemplateRelativePath)
		templateBody, err := os.ReadFile(templatePath)
		if err != nil {
			return nil, fmt.Errorf("read generated Codex journal rule template: %w", err)
		}
		rendered, err := renderCodexJournalRule(string(templateBody))
		if err != nil {
			return nil, err
		}
		return []byte(rendered), nil
	case "codex-guidance":
		return nil, fmt.Errorf("global Codex guidance is retired and cannot be published")
	default:
		return nil, fmt.Errorf("unsupported Codex policy kind %q", decision.Kind)
	}
}

func resolveDecisionPath(options targetInstallOptions, decision artifactPlanDecision) (string, error) {
	if filepath.IsAbs(decision.Destination) {
		return decision.Destination, nil
	}
	return filepath.Join(options.ConfigDir, filepath.FromSlash(decision.Destination)), nil
}

func desiredAdapterContent(options targetInstallOptions, decision artifactPlanDecision) ([]byte, string, error) {
	buildPath := filepath.Join(options.DistDir, targetBuildManifestFile)
	desired, err := readTargetAdapterManifest(buildPath)
	if err != nil {
		return nil, "", err
	}
	for _, artifact := range desired.Artifacts {
		if artifact.ID != decision.ID {
			continue
		}
		body, err := desiredAdapterBody(options, artifact)
		if err != nil {
			return nil, "", err
		}
		return body, sha256Bytes(body), nil
	}
	return nil, "", nil
}

func hookEntryDiff(options targetInstallOptions, decision artifactPlanDecision) (string, string, string, error) {
	reconciler, err := newHookReconciler(options)
	if err != nil || reconciler == nil {
		return "", "", "", err
	}
	live, err := readHookFile(reconciler.path)
	if err != nil {
		return "", "", "", err
	}
	event, hookID, ok := strings.Cut(strings.TrimPrefix(decision.ID, "hook:"), "/")
	if !ok {
		return "", "", "", nil
	}
	liveJSON := ""
	for _, entry := range reconciler.catalog.Entries {
		if entry.Event != event || entry.HookID != hookID {
			continue
		}
		desired, err := reconciler.desiredEntry(entry)
		if err != nil {
			return "", "", "", err
		}
		desiredPretty := prettyJSONBytes(desired)
		if live.exists {
			if entries, err := live.eventEntries(event); err == nil {
				records, recErr := loadHookRecordsForDiff(options, reconciler)
				if recErr == nil {
					outcome, pairErr := pairHookEventEntries(reconciler.recognition(records), event, entries)
					if pairErr == nil {
						for _, pairing := range outcome.paired {
							if pairing.hookID == hookID {
								liveJSON = prettyJSONBytes(live.entries[event][pairing.index])
							}
						}
					}
				}
			}
		}
		return unifiedDiff(decision.ID, liveJSON, desiredPretty), sha256Hex(liveJSON), sha256Hex(desiredPretty), nil
	}
	return "", "", "", nil
}

func loadHookRecordsForDiff(options targetInstallOptions, reconciler *hookReconciler) (hookRecords, error) {
	if options.HookState == nil {
		return hookRecords{enablement: map[string]state.HookEnablement{}}, nil
	}
	store, err := options.HookState()
	if err != nil {
		return hookRecords{}, err
	}
	return loadHookRecords(context.Background(), store, reconciler.target)
}

func prettyJSONBytes(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	buf.WriteByte('\n')
	return buf.String()
}

func directoryUnifiedDiff(liveRoot string, desiredRoot string) (string, error) {
	files := map[string]bool{}
	collect := func(root string) error {
		if root == "" {
			return nil
		}
		if _, err := os.Lstat(root); os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
		}
		return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = true
			return nil
		})
	}
	if err := collect(liveRoot); err != nil {
		return "", err
	}
	if err := collect(desiredRoot); err != nil {
		return "", err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		live := readOptionalFile(filepath.Join(liveRoot, filepath.FromSlash(name)))
		desired := readOptionalFile(filepath.Join(desiredRoot, filepath.FromSlash(name)))
		if live == desired {
			continue
		}
		b.WriteString(unifiedDiff(name, live, desired))
	}
	return b.String(), nil
}

func readOptionalFile(path string) string {
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(body)
}

func unifiedDiff(name, before, after string) string {
	if before == after {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s (live)\n+++ %s (desired)\n", name, name)
	beforeLines := splitDiffLines(before)
	afterLines := splitDiffLines(after)
	fmt.Fprintf(&b, "@@ -1,%d +1,%d @@\n", len(beforeLines), len(afterLines))
	for _, line := range beforeLines {
		b.WriteString("-")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	for _, line := range afterLines {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func splitDiffLines(value string) []string {
	if value == "" {
		return nil
	}
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return []string{""}
	}
	return strings.Split(value, "\n")
}
