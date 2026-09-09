package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// install_plan.go builds a deterministic, byte-for-byte non-mutating plan for
// `loaf upgrade --dry-run`. Every function here reuses the same read-only
// ownership, content-digest, deprecation-manifest, MCP, and project-file
// primitives the apply path uses, but records intended
// creates/updates/removals/preservations/conflicts without writing files,
// manifests, config, or state. The apply path is left untouched; a plan/apply
// parity test guards against drift.

const installPlanContractVersion = 1

// Artifact/action verbs shared by the plan surfaces.
const (
	planActionCreate   = "create"
	planActionUpdate   = "update"
	planActionPreserve = "preserve"
	planActionRetire   = "retire"
	planActionConflict = "conflict"
	planActionNone     = "none"
)

type installDryRunPlan struct {
	ContractVersion  int                      `json:"contract_version"`
	Command          string                   `json:"command"`
	DryRun           bool                     `json:"dry_run"`
	Targets          []targetDistributionPlan `json:"targets"`
	Skills           []artifactPlanDecision   `json:"skills"`
	Deprecations     []deprecationPlanEntry   `json:"deprecations"`
	ProjectPart      *projectPartPlan         `json:"project_part,omitempty"`
	ProjectFiles     []projectFilePlanEntry   `json:"project_files"`
	Mcp              []mcpPlanEntry           `json:"mcp"`
	FollowUpCommands []string                 `json:"follow_up_commands"`
	ConsentRequired  bool                     `json:"consent_required"`
	Selections       []string                 `json:"selections,omitempty"`
	VersionStamp     string                   `json:"version_stamp,omitempty"`
}

// projectPartPlan reports the Loaf-repo detector gate that decides whether the
// project half of the work runs at all, so a consumer can tell an empty
// project_files list that means "nothing to do" from one that means "not a Loaf
// project". Callers that plan project files unconditionally leave it nil.
type projectPartPlan struct {
	InScope              bool     `json:"in_scope"`
	Tier                 string   `json:"tier"`
	ConfirmationRequired bool     `json:"confirmation_required"`
	Bases                []string `json:"bases,omitempty"`
}

type targetDistributionPlan struct {
	Target    string                 `json:"target"`
	ConfigDir string                 `json:"config_dir"`
	Installed bool                   `json:"installed"`
	Blocked   bool                   `json:"blocked"`
	Note      string                 `json:"note,omitempty"`
	Artifacts []artifactPlanDecision `json:"artifacts"`
}

type artifactPlanDecision struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Destination   string `json:"destination"`
	Action        string `json:"action"`
	Detail        string `json:"detail,omitempty"`
	Diff          string `json:"diff,omitempty"`
	Desired       string `json:"-"`
	LiveSHA256    string `json:"live_sha256,omitempty"`
	DesiredSHA256 string `json:"desired_sha256,omitempty"`
}

type deprecationPlanEntry struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	Path            string `json:"path"`
	Action          string `json:"action"`
	ConsentRequired bool   `json:"consent_required"`
	Reason          string `json:"reason,omitempty"`
	Since           string `json:"since,omitempty"`
	Window          string `json:"window,omitempty"`
	Signoff         string `json:"signoff,omitempty"`
}

type projectFilePlanEntry struct {
	Target string `json:"target,omitempty"`
	Path   string `json:"path"`
	Action string `json:"action"`
	Detail string `json:"detail,omitempty"`
}

type mcpPlanEntry struct {
	ID         string `json:"id"`
	Target     string `json:"target"`
	Configured bool   `json:"configured"`
	Scope      string `json:"scope,omitempty"`
	Action     string `json:"action"`
	Notice     string `json:"notice,omitempty"`
}

// runInstallDryRun renders the plan document.
//
// Schema decision (the plan surface moving from install to upgrade): the
// document stays field-compatible at contract version 1. Every field a consumer
// reads today keeps its name, type, and meaning; the only value change is
// `command`, which now names `upgrade` because that is the command that applies
// the plan. `project_part` is a new optional object, omitted entirely when the
// caller plans project files unconditionally, so the previously documented
// document is emitted byte-for-byte unchanged for those callers. Adding an
// omitted-by-default field is additive, so the contract version does not move.
//
// projectPart nil means "plan project files unconditionally"; non-nil carries
// the detector gate and suppresses project-file planning when it is closed, so
// the plan never promises writes the apply path would refuse to make.
func (r Runner) runInstallDryRun(options installOptions, out io.Writer, loafRoot string, projectRoot string, version string, distRoot string, tools []detectedInstallTool, hasClaudeCode bool, assumeYes bool, projectPart *projectPartPlan) error {
	plan, err := r.buildInstallDryRunPlan(options, loafRoot, projectRoot, version, distRoot, tools, hasClaudeCode, assumeYes, projectPart)
	if err != nil {
		return err
	}
	if options.json {
		return emitInstallDryRunJSON(out, plan)
	}
	writeInstallDryRunHuman(out, plan)
	return nil
}

func (r Runner) buildInstallDryRunPlan(options installOptions, loafRoot string, projectRoot string, version string, distRoot string, tools []detectedInstallTool, hasClaudeCode bool, assumeYes bool, projectPart *projectPartPlan) (installDryRunPlan, error) {
	plan := installDryRunPlan{
		ContractVersion: installPlanContractVersion,
		Command:         planCommandName(options),
		DryRun:          true,
		Targets:         []targetDistributionPlan{},
		Skills:          []artifactPlanDecision{},
		Deprecations:    []deprecationPlanEntry{},
		ProjectPart:     projectPart,
		ProjectFiles:    []projectFilePlanEntry{},
		Mcp:             []mcpPlanEntry{},
	}

	selectedTargets, err := r.selectedInstallTargets(options, tools, hasClaudeCode, io.Discard)
	if err != nil {
		return installDryRunPlan{}, err
	}

	// Deprecation cleanup is always analyzed with allowDestructive=false, which
	// is guaranteed non-mutating; the destructive branches only surface as
	// "confirmation-required". Consent for destructive deprecation cleanup is
	// governed solely by explicit --yes in the apply path (never by tty-detected
	// assumeYes), so the plan interprets it the same way.
	explicitYes := options.yes != nil && *options.yes
	deprecations, err := planInstallDeprecations(loafRoot, explicitYes)
	if err != nil {
		return installDryRunPlan{}, err
	}
	plan.Deprecations = deprecations

	toolByKey := installToolsByKey(tools)
	defaults, layoutHome := resolveInstallLayout(projectRoot)
	// A plan reads hook enablement and never creates it: a dry run that brought
	// a state database into existence would be a write, and the plan promises
	// none.
	hookState, releaseHookState := r.hookStateForPlan(projectRoot)
	defer releaseHookState()
	buildNeeded := false
	var plannedOptions []targetInstallOptions
	for _, target := range selectedTargets {
		if target == claudeCodeInstallTarget {
			if entry, include := r.planClaudeCodeTarget(loafRoot, options.upgrade, hasClaudeCode); include {
				plan.Targets = append(plan.Targets, entry)
			}
			continue
		}
		distDir := filepath.Join(distRoot, target)
		configDir := defaults[target]
		if tool, ok := toolByKey[target]; ok && tool.configDir != "" {
			configDir = tool.configDir
		}
		targetPlan := targetDistributionPlan{
			Target:    target,
			ConfigDir: configDir,
			Installed: containsInstallToolInstalled(tools, target),
			Artifacts: []artifactPlanDecision{},
		}
		if !dirExistsForInstall(distDir) {
			targetPlan.Note = "no build output found; run loaf build first"
			buildNeeded = true
			plan.Targets = append(plan.Targets, targetPlan)
			continue
		}
		installOpts := targetInstallOptions{
			Target:              target,
			DistDir:             distDir,
			ConfigDir:           configDir,
			Upgrade:             options.upgrade,
			CodexBasicCommands:  options.codexBasicCommands,
			Version:             version,
			HomeDir:             layoutHome,
			CodexHome:           resolveInstallCodexHome(configDir),
			ProjectRoot:         projectRoot,
			HookState:           hookState,
			SelectedHookIDs:     selectedHookIDsForTarget(options.selections, target),
			SelectedArtifactIDs: selectedArtifactIDsForTarget(options.selections, target),
		}
		plannedOptions = append(plannedOptions, installOpts)
		decisions, err := planTargetDistribution(installOpts)
		if err != nil {
			return installDryRunPlan{}, err
		}
		targetPlan.Artifacts = decisions
		for _, decision := range decisions {
			if decision.Action == planActionConflict {
				targetPlan.Blocked = true
			}
		}
		plan.Targets = append(plan.Targets, targetPlan)
	}
	sort.Slice(plan.Targets, func(i, j int) bool { return plan.Targets[i].Target < plan.Targets[j].Target })

	// A managed-skill conflict blocks the whole shared-content cohort. Apply
	// leaves target adapters and markers stale so a later reconcile can retry.
	skills, err := planCanonicalManagedSkills(plannedOptions)
	if err != nil {
		return installDryRunPlan{}, err
	}
	plan.Skills = skills
	for _, skill := range skills {
		if skill.Action != planActionConflict {
			continue
		}
		for i := range plan.Targets {
			plan.Targets[i].Blocked = true
		}
		break
	}

	// Project files mirror enforceInstallProjectFiles: symlinks first, then the
	// managed fenced section for every target that carries a project file, then
	// upgrade's own MCP-recommendation record. A closed detector gate skips all
	// of it, exactly as the apply path does. Upgrade (the caller that supplies a
	// project part) refreshes the surfaces of every installed target regardless
	// of --to, so the plan reports that same unfiltered set.
	targetsInScope := withoutString(selectedTargets, claudeCodeInstallTarget)
	if projectPart != nil {
		targetsInScope = installedUpgradeTargets(tools)
	}
	if projectPart == nil || projectPart.InScope {
		plan.ProjectFiles = planInstallProjectFiles(projectRoot, targetsInScope, hasClaudeCode, assumeYes, version)
		if projectPart != nil {
			plan.ProjectFiles = append(plan.ProjectFiles, planUpgradeMcpRecord(projectRoot))
		}
	}

	// MCP recommendations do not run during --upgrade; report read-only
	// detection so the plan is informative while making it explicit that
	// upgrade applies no MCP changes.
	mcpTargets := append([]string{}, targetsInScope...)
	if hasClaudeCode {
		mcpTargets = append(mcpTargets, claudeCodeInstallTarget)
	}
	plan.Mcp = planInstallMcp(projectRoot, mcpTargets)

	plan.ConsentRequired = installPlanConsentRequired(plan)
	plan.FollowUpCommands = installPlanFollowUpCommands(options, plan, buildNeeded)
	return plan, nil
}

func containsInstallToolInstalled(tools []detectedInstallTool, target string) bool {
	for _, tool := range tools {
		if tool.key == target {
			return tool.installed
		}
	}
	return false
}

// planTargetDistribution mirrors installTargetDistribution's branching without
// writing anything.
func planTargetDistribution(options targetInstallOptions) ([]artifactPlanDecision, error) {
	var decisions []artifactPlanDecision
	// Skills are planned once per destination by planCanonicalManagedSkills;
	// including them here would re-report shared conflicts once per target.

	hasAdapterManifest := fileExistsForInstall(filepath.Join(options.DistDir, targetBuildManifestFile))
	if hasAdapterManifest {
		adapters, err := planTargetAdapterArtifacts(options)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, adapters...)
	} else {
		legacy, err := planLegacyHookArtifacts(options)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, legacy...)
	}
	if options.Target == "opencode" {
		commands, err := planOpenCodeCommandsGuard(options)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, commands...)
	}
	if options.Target == "codex" {
		codex, err := planCodexJournalRule(options)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, codex...)
	}
	sort.SliceStable(decisions, func(i, j int) bool { return decisions[i].ID < decisions[j].ID })
	return decisions, nil
}

// planOpenCodeCommandsGuard mirrors syncOpenCodeCommandsDir's fail-closed skills
// store requirement. Commands sync runs whenever a real commands directory is
// present; when the canonical store is absent and no skills tree will create
// it, apply fails after adapter work — so the plan must block rather than look
// preservable.
func planOpenCodeCommandsGuard(options targetInstallOptions) ([]artifactPlanDecision, error) {
	src := filepath.Join(options.DistDir, "commands")
	info, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	// Symlinked or non-directory sources are refused on apply before the store
	// check; this guard covers only the store failure mode.
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, nil
	}
	skillsDest := installSkillsDestination(options)
	if dirExistsForInstall(skillsDest) {
		return nil, nil
	}
	// Canonical skills sync creates the store when a skills source tree exists,
	// and that sync precedes command install on the apply path.
	if dirExistsForInstall(filepath.Join(options.DistDir, "skills")) {
		return nil, nil
	}
	return []artifactPlanDecision{{
		ID:          "opencode:commands",
		Kind:        "commands",
		Destination: filepath.Join(options.ConfigDir, "commands"),
		Action:      planActionConflict,
		Detail:      fmt.Sprintf("cannot install OpenCode commands: canonical skills store %s does not exist", skillsDest),
	}}, nil
}

// planCanonicalManagedSkills plans managed-skill actions once per resolved
// destination across the selected targets. Divergent source trees fail loudly.
// The planned source is selectCanonicalSkillsSource's pick, matching the write.
// Conflicted skills are listed with action "conflict"; the caller blocks the
// target cohort so marker truth remains retryable.
func planCanonicalManagedSkills(options []targetInstallOptions) ([]artifactPlanDecision, error) {
	groups, err := groupSkillsInstallByDestination(options)
	if err != nil {
		return nil, err
	}
	var decisions []artifactPlanDecision
	for _, group := range groups {
		source, err := selectCanonicalSkillsSource(group.Options)
		if err != nil {
			return nil, err
		}
		src := filepath.Join(source.DistDir, "skills")
		skills, err := planManagedSkills(src, group.Destination)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, skills...)
	}
	sort.SliceStable(decisions, func(i, j int) bool {
		if decisions[i].Destination != decisions[j].Destination {
			return decisions[i].Destination < decisions[j].Destination
		}
		return decisions[i].ID < decisions[j].ID
	})
	return decisions, nil
}

// planManagedSkills mirrors the read-only preflight and classification of
// syncManagedSkillsDirIfExists.
func planManagedSkills(src string, dest string) ([]artifactPlanDecision, error) {
	if !dirExistsForInstall(src) {
		return nil, nil
	}
	sourceSkills, err := listInstallSkillDirs(src)
	if err != nil {
		return nil, err
	}
	previous, err := readManagedSkillsState(dest)
	if err != nil {
		return nil, err
	}
	current := map[string]string{}
	for _, skill := range sourceSkills {
		digest, err := hashInstallSkillTree(filepath.Join(src, skill))
		if err != nil {
			return nil, fmt.Errorf("hash source skill %q: %w", skill, err)
		}
		current[skill] = digest
	}

	conflicts := map[string]string{}
	// Modified previously-managed skills (mirror the ownership preflight).
	for skill, recordedDigest := range previous.digests {
		if previous.legacy {
			continue
		}
		actual, err := hashInstallSkillTree(filepath.Join(dest, skill))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("managed skill %q cannot be verified: %w", skill, err)
		}
		if actual != recordedDigest && actual != current[skill] {
			conflicts[skill] = "managed skill was modified; refusing to overwrite or remove"
		}
	}
	// Unowned destinations that collide with a source skill.
	for _, skill := range sourceSkills {
		if _, owned := previous.digests[skill]; owned {
			continue
		}
		if _, err := os.Lstat(filepath.Join(dest, skill)); err == nil {
			conflicts[skill] = "skill destination already exists and is not managed by Loaf"
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var decisions []artifactPlanDecision
	for _, skill := range sourceSkills {
		id := "skill:" + skill
		destination := filepath.Join(dest, skill)
		if reason, bad := conflicts[skill]; bad {
			decisions = append(decisions, artifactPlanDecision{ID: id, Kind: "skill", Destination: destination, Action: planActionConflict, Detail: reason})
			continue
		}
		installedHash, hashErr := hashInstallSkillTree(destination)
		switch {
		case hashErr == nil && installedHash == current[skill]:
			decisions = append(decisions, artifactPlanDecision{ID: id, Kind: "skill", Destination: destination, Action: planActionPreserve})
		case hashErr == nil:
			decisions = append(decisions, artifactPlanDecision{ID: id, Kind: "skill", Destination: destination, Action: planActionUpdate})
		case os.IsNotExist(hashErr):
			decisions = append(decisions, artifactPlanDecision{ID: id, Kind: "skill", Destination: destination, Action: planActionCreate})
		default:
			return nil, fmt.Errorf("verify managed skill %q: %w", skill, hashErr)
		}
	}
	// Retire previously-managed skills that the source no longer ships.
	for skill := range previous.digests {
		if _, keep := current[skill]; keep {
			continue
		}
		destination := filepath.Join(dest, skill)
		if _, err := os.Lstat(destination); os.IsNotExist(err) {
			// The directory is gone, but its ownership entry still needs to
			// retire and remain addressable by an explicit scoped selection.
			decisions = append(decisions, artifactPlanDecision{ID: "skill:" + skill, Kind: "skill", Destination: destination, Action: planActionRetire, Detail: "retire ownership of absent skill"})
			continue
		} else if err != nil {
			return nil, err
		}
		action := planActionRetire
		detail := ""
		if reason, bad := conflicts[skill]; bad {
			action = planActionConflict
			detail = reason
		}
		decisions = append(decisions, artifactPlanDecision{ID: "skill:" + skill, Kind: "skill", Destination: destination, Action: action, Detail: detail})
	}
	sort.SliceStable(decisions, func(i, j int) bool { return decisions[i].ID < decisions[j].ID })
	return decisions, nil
}

// planTargetAdapterArtifacts mirrors the read-only analysis of
// syncTargetAdapterManifest.
func planTargetAdapterArtifacts(options targetInstallOptions) ([]artifactPlanDecision, error) {
	buildPath := filepath.Join(options.DistDir, targetBuildManifestFile)
	desired, err := readTargetAdapterManifest(buildPath)
	if err != nil {
		return nil, err
	}
	if desired.Target != options.Target {
		return nil, fmt.Errorf("target adapter manifest target %q does not match install target %q", desired.Target, options.Target)
	}
	installedPath := filepath.Join(options.ConfigDir, targetInstallManifestFile)
	installed := targetAdapterManifest{}
	if _, err := os.Lstat(installedPath); err == nil {
		installed, err = readTargetAdapterManifest(installedPath)
		if err != nil {
			return nil, err
		}
		if installed.Target != options.Target {
			return nil, fmt.Errorf("installed target adapter manifest target %q does not match %q", installed.Target, options.Target)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	desiredByID := targetAdapterArtifactsByID(desired.Artifacts)
	installedByID := targetAdapterArtifactsByID(installed.Artifacts)

	var decisions []artifactPlanDecision
	desiredDestinations := map[string]bool{}

	// Hook entries are planned per identity by the reconciler, from the same
	// computation apply runs. It reads state and the live file and writes
	// nothing.
	reconciler, err := newHookReconciler(options)
	if err != nil {
		return nil, err
	}
	if reconciler != nil {
		actions, err := reconciler.plan(context.Background())
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, hookActionPlanDecisions(reconciler, actions)...)
		decisions = append(decisions, selectedHookPreserveDecisions(reconciler, options.SelectedHookIDs, decisions)...)
	}

	for _, artifact := range desired.Artifacts {
		if skipTargetAdapterArtifact(artifact) {
			continue
		}
		if err := verifyTargetAdapterSource(options, artifact); err != nil {
			return nil, err
		}
		path, err := targetAdapterDestination(options, artifact)
		if err != nil {
			return nil, err
		}
		desiredDestinations[path] = true
		snapshot, err := readTargetAdapterSnapshot(path)
		if err != nil {
			return nil, err
		}
		_, owned := installedByID[artifact.ID]
		decision := artifactPlanDecision{ID: artifact.ID, Kind: artifact.Kind, Destination: artifact.Destination}
		if !snapshot.exists {
			decision.Action = planActionCreate
			decisions = append(decisions, decision)
			continue
		}
		matchesDesired, err := targetAdapterSnapshotMatchesDesired(options, artifact, snapshot)
		if err != nil {
			return nil, err
		}
		if owned {
			matchesInstalled := targetAdapterSnapshotMatchesArtifact(installedByID[artifact.ID], snapshot)
			switch {
			case !matchesInstalled && !matchesDesired:
				decision.Action = planActionConflict
				decision.Detail = "managed target artifact was modified; refusing to overwrite or remove"
			case matchesDesired:
				decision.Action = planActionPreserve
			default:
				decision.Action = planActionUpdate
			}
			decisions = append(decisions, decision)
			continue
		}
		// Unowned destination that already exists: mirror the migration checks.
		switch {
		case matchesDesired:
			decision.Action = planActionPreserve
		case targetAdapterLegacyOwnership(options.Target, artifact, snapshot.body), ampModesPluginExactPredecessor(options.Target, artifact, snapshot.body):
			decision.Action = planActionUpdate
			decision.Detail = "adopting legacy Loaf-owned content"
		default:
			decision.Action = planActionConflict
			decision.Detail = "destination exists and is not managed by Loaf"
		}
		decisions = append(decisions, decision)
	}

	// Retire installed artifacts the desired manifest no longer ships.
	for _, artifact := range installed.Artifacts {
		if skipTargetAdapterArtifact(artifact) {
			continue
		}
		if _, keep := desiredByID[artifact.ID]; keep {
			continue
		}
		path, err := targetAdapterDestination(options, artifact)
		if err != nil {
			return nil, err
		}
		if desiredDestinations[path] {
			continue
		}
		snapshot, err := readTargetAdapterSnapshot(path)
		if err != nil {
			return nil, err
		}
		if !snapshot.exists {
			continue
		}
		decision := artifactPlanDecision{ID: artifact.ID, Kind: artifact.Kind, Destination: artifact.Destination, Action: planActionRetire}
		if !targetAdapterSnapshotMatchesArtifact(artifact, snapshot) {
			decision.Action = planActionConflict
			decision.Detail = "managed target artifact was modified; refusing to remove"
		}
		decisions = append(decisions, decision)
	}
	preserved, err := selectedRetiredAdapterPreserveDecisions(options, desiredByID, installedByID, installed.RetiredArtifactIDs, decisions)
	if err != nil {
		return nil, err
	}
	decisions = append(decisions, preserved...)
	sort.SliceStable(decisions, func(i, j int) bool { return decisions[i].ID < decisions[j].ID })
	return decisions, nil
}

// planCodexJournalRule plans permission rules independently of retired global
// guidance. It performs read-only ownership checks and fails closed when rule
// convergence requires an unavailable PATH loaf.
func planCodexJournalRule(options targetInstallOptions) ([]artifactPlanDecision, error) {
	codexHome := effectiveCodexHome(options)
	ruleDest := filepath.Join(codexHome, "rules", codexJournalRuleRelativePath)
	manifest, err := readCodexManagedRuleManifest(codexPolicyOwnershipManifestPath(options))
	if err != nil {
		return nil, err
	}
	guidance := planRetiredCodexGuidance(filepath.Join(codexHome, codexJournalGuidanceRelativePath), manifest)
	rule := artifactPlanDecision{ID: "codex-rule:loaf.rules", Kind: "codex-rule", Destination: ruleDest}
	ownedRuleSHA, ownedRule := manifest.ownedDigest(codexJournalRuleRelativePath)
	// Guidance ownership never elects permission-rule installation.
	if !options.CodexBasicCommands && !(options.Upgrade && ownedRule) {
		rule.Action = planActionPreserve
		return []artifactPlanDecision{rule, guidance}, nil
	}
	templatePath := filepath.Join(options.DistDir, ".codex", "rules", codexJournalRuleTemplateRelativePath)
	templateBody, templateErr := os.ReadFile(templatePath)
	if templateErr != nil && !os.IsNotExist(templateErr) {
		return nil, fmt.Errorf("read generated Codex journal rule template: %w", templateErr)
	}
	legacy, err := detectLegacyCodexJournalCapability(ruleDest)
	if err != nil {
		return nil, err
	}
	if (os.IsNotExist(templateErr) || legacy) && options.Upgrade && !options.CodexBasicCommands {
		retired := planCodexPolicyRetirements([]artifactPlanDecision{rule}, manifest)
		return append(retired, guidance), nil
	}
	if os.IsNotExist(templateErr) {
		rule.Action, rule.Detail = planActionConflict, "generated Codex journal rule template is missing"
		return []artifactPlanDecision{rule, guidance}, nil
	}
	if _, err := trustedCodexJournalExecutable(options.ProjectRoot, options.CodexRuleOperations); err != nil {
		rule.Action = planActionConflict
		rule.Detail = err.Error() + ". " + pathLoafInstallUpgradeGuidance()
		return []artifactPlanDecision{rule, guidance}, nil
	}
	renderedRule, err := renderCodexJournalRule(string(templateBody))
	if err != nil {
		return nil, err
	}
	rule, err = planCodexRuleFile(ruleDest, options, ownedRule, ownedRuleSHA, sha256Bytes([]byte(renderedRule)))
	if err != nil {
		return nil, err
	}
	return []artifactPlanDecision{rule, guidance}, nil
}

// Global instructions are user-owned. Only an intact, recorded block in a
// regular file can be retired; guidance drift must not block independent rules.
func planRetiredCodexGuidance(path string, manifest codexManagedRuleManifest) artifactPlanDecision {
	decision := artifactPlanDecision{ID: "codex-rule:AGENTS.md", Kind: "codex-guidance", Destination: path, Action: planActionPreserve}
	if _, owned := manifest.ownedDigest(codexJournalGuidanceRelativePath); !owned {
		return decision
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		decision.Detail = "preserving absent, unreadable, or non-regular global instructions; guidance is no longer managed"
		return decision
	}
	if _, _, err := readOwnedCodexPolicyForRetirement(path, decision.Kind, manifest); err != nil {
		decision.Detail = "preserving changed global guidance; guidance is no longer managed"
		return decision
	}
	decision.Action = planActionRetire
	decision.Detail = "remove only the unchanged Loaf-owned global guidance block"
	return decision
}

func planCodexPolicyRetirements(decisions []artifactPlanDecision, manifest codexManagedRuleManifest) []artifactPlanDecision {
	for i := range decisions {
		_, owned, err := readOwnedCodexPolicyForRetirement(decisions[i].Destination, decisions[i].Kind, manifest)
		switch {
		case err != nil:
			decisions[i].Action = planActionConflict
			decisions[i].Detail = err.Error()
		case !owned:
			decisions[i].Action = planActionPreserve
			decisions[i].Detail = "preserving unowned Codex policy"
		default:
			decisions[i].Action = planActionRetire
		}
	}
	return decisions
}

func planCodexRuleFile(ruleDest string, options targetInstallOptions, ownedRule bool, ownedRuleSHA string, newRuleSHA string) (artifactPlanDecision, error) {
	decision := artifactPlanDecision{ID: "codex-rule:loaf.rules", Kind: "codex-rule", Destination: ruleDest}
	currentRule, ruleExists, err := readOptionalInstallFile(ruleDest, "installed Codex journal rule")
	if err != nil {
		return decision, err
	}
	if !ruleExists {
		decision.Action = planActionCreate
		return decision, nil
	}
	currentSHA := sha256Bytes(currentRule)
	switch {
	case currentSHA == newRuleSHA && ((ownedRule && ownedRuleSHA != newRuleSHA) || (!ownedRule && options.CodexBasicCommands)):
		decision.Action = planActionUpdate
	case currentSHA == newRuleSHA:
		decision.Action = planActionPreserve
	case ownedRule && currentSHA == ownedRuleSHA:
		decision.Action = planActionUpdate
	case !ownedRule && options.CodexBasicCommands:
		decision.Action = planActionConflict
		decision.Detail = "refusing to overwrite unowned Codex rule"
	case ownedRule:
		decision.Action = planActionConflict
		decision.Detail = "refusing to overwrite modified Loaf-owned Codex rule"
	default:
		decision.Action = planActionNone
	}
	return decision, nil
}

// planLegacyHookArtifacts reports the no-manifest hook refresh for the targets
// that still have one: a distribution predating the target adapter manifest
// refreshes plugins wholesale on apply.
func planLegacyHookArtifacts(options targetInstallOptions) ([]artifactPlanDecision, error) {
	// A target whose entries are reconciled has no legacy path left to promise:
	// apply refuses a build output that predates the catalog, so the plan says
	// the same thing rather than describing a refresh that will not happen.
	if targetReconcilesHookEntries(options) {
		_, err := newHookReconciler(options)
		return []artifactPlanDecision{{
			ID:          "hooks",
			Kind:        "hook-legacy",
			Destination: targetHookFilePath(options),
			Action:      planActionConflict,
			Detail:      err.Error(),
		}}, nil
	}
	return []artifactPlanDecision{{
		ID:          "hooks",
		Kind:        "hook-legacy",
		Destination: options.ConfigDir,
		Action:      planActionUpdate,
		Detail:      "legacy build output without a target adapter manifest; hooks/plugins refreshed on apply",
	}}, nil
}

// planInstallDeprecations reuses applyInstallDeprecationCleanup with
// allowDestructive=false, which is guaranteed non-mutating, then interprets the
// classified result. explicitYes reflects an explicit --yes; destructive
// deprecation cleanup requires it, matching the apply contract.
func planInstallDeprecations(loafRoot string, explicitYes bool) ([]deprecationPlanEntry, error) {
	manifest, found, err := loadInstallDeprecationManifest(loafRoot)
	if err != nil {
		return nil, err
	}
	if !found || manifest.isEmpty() {
		return []deprecationPlanEntry{}, nil
	}
	result, err := applyInstallDeprecationCleanup(manifest, installPathContext(), false)
	if err != nil {
		return nil, err
	}
	var entries []deprecationPlanEntry
	appendEntry := func(action installDeprecationCleanupAction, planAction string, consentRequired bool) {
		entries = append(entries, deprecationPlanEntry{
			Kind:            action.Kind,
			Name:            action.Name,
			Path:            action.Path,
			Action:          planAction,
			ConsentRequired: consentRequired,
			Reason:          action.Reason,
			Since:           action.Since,
			Window:          action.Window,
			Signoff:         action.Signoff,
		})
	}
	for _, action := range result.Aliases {
		appendEntry(action, "alias", false)
	}
	// "missing"/"unmarked" are not recorded on the cleanup result; the plan stays
	// silent for those cases if any legacy caller still emits them.
	for _, action := range result.Skipped {
		switch action.Action {
		case "confirmation-required":
			appendEntry(action, destructiveDeprecationAction(action.Kind), !explicitYes)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Path < entries[j].Path
	})
	if entries == nil {
		entries = []deprecationPlanEntry{}
	}
	return entries, nil
}

func destructiveDeprecationAction(kind string) string {
	if kind == "path" {
		return "relocate"
	}
	return "remove"
}

// planInstallProjectFiles mirrors enforceInstallProjectFiles: project symlinks
// followed by the managed fenced section, all read-only.
func planInstallProjectFiles(projectRoot string, selectedTargets []string, hasClaudeCode bool, assumeYes bool, version string) []projectFilePlanEntry {
	entries := planInstallProjectSymlinks(projectRoot, selectedTargets, hasClaudeCode, assumeYes)
	fencedTargets := append([]string{}, selectedTargets...)
	if hasClaudeCode {
		fencedTargets = append([]string{"claude-code"}, fencedTargets...)
	}
	entries = append(entries, planInstallFencedSections(fencedTargets, projectRoot, version)...)
	if entries == nil {
		entries = []projectFilePlanEntry{}
	}
	return entries
}

func planInstallProjectSymlinks(projectRoot string, selectedTargets []string, hasClaudeCode bool, assumeYes bool) []projectFilePlanEntry {
	var entries []projectFilePlanEntry
	wantClaude := hasClaudeCode || containsString(selectedTargets, "claude-code")
	wantRootAgents := needsRootInstallAgentsFile(selectedTargets)
	if !wantClaude && !wantRootAgents {
		return entries
	}
	canonical := filepath.Join(projectRoot, "AGENTS.md")
	rootAction, rootDetail, rootErr := planRootInstallAgentsFile(projectRoot, assumeYes)
	entries = append(entries, projectFilePlanEntry{Path: "./AGENTS.md", Action: rootAction, Detail: rootDetail})
	if rootErr {
		return entries
	}
	if wantClaude {
		linkPath := filepath.Join(projectRoot, ".claude", "CLAUDE.md")
		relTarget := relativeInstallLinkTarget(linkPath, canonical)
		action, detail := planInstallSymlink(linkPath, relTarget, ".claude/CLAUDE.md", canonical, assumeYes)
		entries = append(entries, projectFilePlanEntry{Target: "claude-code", Path: ".claude/CLAUDE.md", Action: action, Detail: detail})
	}
	return entries
}

// planProjectFileReadable answers, for the plan, the question the apply path
// answers by reading: will a whole-file read of this path be refused? The plan
// has to ask it the same way, because a migration the plan promises and apply
// refuses is worse than either outcome on its own. Absence is not a refusal —
// the callers already branch on it — and the read costs what the apply read
// costs, which is what planFencedSection already pays.
func planProjectFileReadable(path string) error {
	if _, err := readRegularFile(path, projectFileReadLimit); err != nil && !os.IsNotExist(err) {
		return refuseProjectFileRead(err)
	}
	return nil
}

// planRootInstallAgentsFile mirrors the read-only branch decisions of
// ensureRootInstallAgentsFile. Returns (action, detail, isError).
func planRootInstallAgentsFile(projectRoot string, assumeYes bool) (string, string, bool) {
	canonical := filepath.Join(projectRoot, "AGENTS.md")
	legacy := filepath.Join(projectRoot, ".agents", "AGENTS.md")
	legacyExists := installFileExists(legacy) && !installIsSymlink(legacy)
	if installIsDirectory(canonical) {
		return "error", "./AGENTS.md is a directory; expected a canonical real file", true
	}
	if installIsSymlink(canonical) && legacyExists && installSymlinkPointsTo(canonical, legacy) {
		return "migrated", "Migrate .agents/AGENTS.md to canonical ./AGENTS.md", false
	}
	if !installPathExists(canonical) {
		if legacyExists {
			return "migrated", "Migrate .agents/AGENTS.md to canonical ./AGENTS.md", false
		}
		return "created", "Create canonical ./AGENTS.md", false
	}
	if installIsSymlink(canonical) {
		if !assumeYes {
			return "skipped-no-tty", "./AGENTS.md is a symlink; skipped conversion in non-interactive mode", false
		}
		if err := planProjectFileReadable(canonical); err != nil {
			return "error", fmt.Sprintf("Failed to read ./AGENTS.md: %v", err), true
		}
		return "replaced-file", "Back up the ./AGENTS.md symlink and create a canonical real file", false
	}
	if legacyExists {
		if !assumeYes {
			return "skipped-no-tty", "Both ./AGENTS.md and .agents/AGENTS.md are real files; skipped merge in non-interactive mode", false
		}
		if err := planProjectFileReadable(legacy); err != nil {
			return "error", fmt.Sprintf("Failed to migrate .agents/AGENTS.md: %v", err), true
		}
		if err := planProjectFileReadable(canonical); err != nil {
			return "error", fmt.Sprintf("Failed to migrate .agents/AGENTS.md: %v", err), true
		}
		return "migrated", "Merge legacy .agents/AGENTS.md into canonical ./AGENTS.md", false
	}
	return "already-correct", "Canonical ./AGENTS.md already exists", false
}

// planInstallSymlink mirrors the read-only branch decisions of
// ensureInstallSymlink.
func planInstallSymlink(linkPath string, relativeTarget string, description string, canonicalPath string, assumeYes bool) (string, string) {
	expectedAbs := filepath.Clean(filepath.Join(filepath.Dir(linkPath), relativeTarget))
	if !installPathExists(linkPath) {
		return "created", fmt.Sprintf("Create %s -> %s", description, relativeTarget)
	}
	if installIsSymlink(linkPath) {
		if installSymlinkPointsTo(linkPath, expectedAbs) {
			return "already-correct", fmt.Sprintf("%s already points to %s", description, relativeTarget)
		}
		if !assumeYes {
			return "skipped-no-tty", fmt.Sprintf("%s points to the wrong target; skipped in non-interactive mode", description)
		}
		return "relinked", fmt.Sprintf("Relink %s -> %s", description, relativeTarget)
	}
	if !assumeYes {
		return "skipped-no-tty", fmt.Sprintf("%s exists as a real file; skipped in non-interactive mode", description)
	}
	if err := planProjectFileReadable(linkPath); err != nil {
		return "error", fmt.Sprintf("Failed to replace %s: %v", description, err)
	}
	if canonicalPath != "" {
		if err := planProjectFileReadable(canonicalPath); err != nil {
			return "error", fmt.Sprintf("Failed to replace %s: %v", description, err)
		}
	}
	return "replaced-file", fmt.Sprintf("Back up %s and replace with a symlink -> %s", description, relativeTarget)
}

// planInstallFencedSections mirrors installFencedSectionsForTargets +
// installFencedSection without writing.
func planInstallFencedSections(targets []string, projectRoot string, version string) []projectFilePlanEntry {
	var entries []projectFilePlanEntry
	writtenPaths := map[string]string{}
	for _, target := range targets {
		relPath, ok := fencedTargetFiles[target]
		if !ok {
			entries = append(entries, projectFilePlanEntry{Target: target, Path: "", Action: "error", Detail: "Unknown target: " + target})
			continue
		}
		targetFile := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		canonicalBefore := canonicalInstallPath(targetFile)
		if _, ok := writtenPaths[canonicalBefore]; ok {
			entries = append(entries, projectFilePlanEntry{Target: target, Path: relPath, Action: "skipped", Detail: "shared canonical project file already planned"})
			continue
		}
		action, detail := planFencedSection(targetFile, version)
		entries = append(entries, projectFilePlanEntry{Target: target, Path: relPath, Action: action, Detail: detail})
		writtenPaths[canonicalBefore] = version
	}
	return entries
}

func planFencedSection(targetFile string, version string) (string, string) {
	canonicalTarget, err := canonicalFenceWritePath(targetFile)
	if err != nil {
		return "error", err.Error()
	}
	targetFile = canonicalTarget
	if version == "" {
		version = "0.0.0"
	}
	body, err := readRegularFile(targetFile, projectFileReadLimit)
	fileExisted := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "error", refuseProjectFileRead(err).Error()
	}
	content := string(body)
	if err := validateFencedStructure(content); err != nil {
		return "error", err.Error()
	}
	section, hasSection := findFencedSectionRange(content)
	newContent := generateFencedContent()
	switch {
	case hasSection:
		if section.malformedHeader {
			return "error", "managed Loaf section has a malformed start marker; refusing to overwrite"
		}
		if content[section.start:section.end] == newContent {
			return "skipped", "Loaf framework section already current (v" + version + ")"
		}
		return "updated", "Update Loaf framework section (v" + version + ")"
	case fileExisted:
		return "appended", "Add Loaf framework section to project file"
	default:
		return "created", "Create project file with Loaf framework section"
	}
}

func planInstallMcp(projectRoot string, availableTargets []string) []mcpPlanEntry {
	targets := uniqueInstallTargets(availableTargets)
	entries := []mcpPlanEntry{}
	for _, def := range installMcpDefinitions {
		for _, target := range targets {
			status := detectInstallMcpForTarget(projectRoot, target, def.id)
			entries = append(entries, mcpPlanEntry{
				ID:         def.id,
				Target:     target,
				Configured: status.configured,
				Scope:      status.scope,
				Action:     planActionNone,
				Notice:     status.notice,
			})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].ID != entries[j].ID {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].Target < entries[j].Target
	})
	return entries
}

func installPlanConsentRequired(plan installDryRunPlan) bool {
	for _, entry := range plan.Deprecations {
		if entry.ConsentRequired {
			return true
		}
	}
	return false
}

func installPlanFollowUpCommands(options installOptions, plan installDryRunPlan, buildNeeded bool) []string {
	commands := []string{}
	if buildNeeded {
		commands = append(commands, "loaf build")
	}
	if installPlanHasChanges(plan) {
		commands = append(commands, installPlanApplyCommand(options, plan.ConsentRequired))
	}
	commands = dedupeSortedStrings(commands)
	return commands
}

func installPlanHasChanges(plan installDryRunPlan) bool {
	for _, artifact := range plan.Skills {
		switch artifact.Action {
		case planActionCreate, planActionUpdate, planActionRetire, planActionConflict:
			return true
		}
	}
	for _, target := range plan.Targets {
		if target.Note != "" || target.Blocked {
			return true
		}
		for _, artifact := range target.Artifacts {
			switch artifact.Action {
			case planActionCreate, planActionUpdate, planActionRetire, planActionConflict,
				hookActionAdd, hookActionRemove, hookActionAbsorb:
				return true
			}
		}
	}
	for _, entry := range plan.Deprecations {
		switch entry.Action {
		case "remove", "relocate":
			return true
		}
	}
	for _, entry := range plan.ProjectFiles {
		switch entry.Action {
		case "already-correct", "skipped", planActionPreserve, planActionNone:
		default:
			return true
		}
	}
	return false
}

// planCommandName resolves the command a plan speaks for. Every command string
// the plan emits — JSON envelope, human title, apply and follow-up commands —
// is built from it, so a plan can never advertise an entry point that does not
// exist. An unset command resolves to `upgrade`: the document describes an
// upgrade, and upgrade is the command that applies it.
func planCommandName(options installOptions) string {
	if options.command == "" {
		return upgradeCommandName
	}
	return options.command
}

// installPlanApplyCommand emits only flags the resolved command accepts. The
// Codex basic-command policy is an install-time opt-in, so it is never part of
// an apply command.
func installPlanApplyCommand(options installOptions, consentRequired bool) string {
	if len(options.selections) > 0 {
		return scopedFollowUpCommand(options.selections)
	}
	target := options.target
	if options.upgrade && options.resolvedUpgradeTargets != nil {
		// There is no non-interactive spelling for "no global targets". Never
		// advertise the unfiltered upgrade when the picker selected none.
		if len(options.resolvedUpgradeTargets) == 0 {
			return ""
		}
		target = strings.Join(options.resolvedUpgradeTargets, ",")
	}
	parts := []string{"loaf", planCommandName(options)}
	if target != "" {
		parts = append(parts, "--to", target)
	}
	if consentRequired {
		parts = append(parts, "--yes")
	}
	return strings.Join(parts, " ")
}

func dedupeSortedStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	if result == nil {
		result = []string{}
	}
	return result
}

func emitInstallDryRunJSON(out io.Writer, plan installDryRunPlan) error {
	body, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	_, err = out.Write(body)
	return err
}

func writeInstallDryRunHuman(out io.Writer, plan installDryRunPlan) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, ansiBold("loaf "+plan.Command+" --dry-run"))
	fmt.Fprintf(out, "  %s\n\n", ansiGray("Plan only — no files, manifests, config, or state will change."))
	if len(plan.Selections) > 0 {
		fmt.Fprintf(out, "  %s\n", ansiBold("Scoped selections"))
		for _, sel := range plan.Selections {
			fmt.Fprintf(out, "    %s\n", sel)
		}
		if plan.VersionStamp != "" {
			fmt.Fprintf(out, "  %s %s\n\n", ansiGray("version stamp"), plan.VersionStamp)
		}
	}

	if len(plan.Targets) == 0 {
		fmt.Fprintf(out, "  %s\n", ansiGray("No installed targets to upgrade"))
	}
	if len(plan.Skills) > 0 {
		fmt.Fprintf(out, "  %s\n", ansiBold("Skills"))
		for _, artifact := range plan.Skills {
			fmt.Fprintf(out, "    %s %s %s%s\n", planActionGlyph(artifact.Action), artifact.Action, artifact.ID, planDetailSuffix(artifact.Detail))
			writePlanDiff(out, artifact.Diff)
		}
		fmt.Fprintln(out)
	}
	for _, target := range plan.Targets {
		header := ansiBold(installDisplayName(target.Target))
		if target.Blocked {
			header += " " + ansiRed("(blocked)")
		}
		fmt.Fprintf(out, "  %s %s\n", header, ansiGray(target.ConfigDir))
		if target.Note != "" {
			fmt.Fprintf(out, "    %s %s\n", ansiYellow("⚠"), target.Note)
		}
		for _, artifact := range target.Artifacts {
			fmt.Fprintf(out, "    %s %s %s%s\n", planActionGlyph(artifact.Action), artifact.Action, artifact.ID, planDetailSuffix(artifact.Detail))
			writePlanDiff(out, artifact.Diff)
		}
		fmt.Fprintln(out)
	}

	if len(plan.Deprecations) > 0 {
		fmt.Fprintf(out, "  %s\n", ansiBold("Deprecations"))
		for _, entry := range plan.Deprecations {
			consent := ""
			if entry.ConsentRequired {
				consent = " " + ansiYellow("(needs --yes)")
			}
			fmt.Fprintf(out, "    %s %s %s %s at %s%s\n", planActionGlyph(entry.Action), entry.Action, entry.Kind, entry.Name, ansiGray(entry.Path), consent)
		}
		fmt.Fprintln(out)
	}

	if part := plan.ProjectPart; part != nil && !part.InScope {
		reason := "no Loaf project detected here"
		if part.ConfirmationRequired {
			reason = "legacy Loaf artifacts only; confirmation required"
		}
		fmt.Fprintf(out, "  %s %s\n\n", ansiBold("Project files"), ansiGray("skipped — "+reason))
	}
	if len(plan.ProjectFiles) > 0 {
		fmt.Fprintf(out, "  %s\n", ansiBold("Project files"))
		for _, entry := range plan.ProjectFiles {
			fmt.Fprintf(out, "    %s %s %s%s\n", planActionGlyph(entry.Action), entry.Action, entry.Path, planDetailSuffix(entry.Detail))
		}
		fmt.Fprintln(out)
	}

	// Only the notices are printed, not the MCP detection table: a plan the
	// human reads says what will change, and detection changes nothing. A
	// config that could not be inspected is the exception, because it is the
	// one MCP line that says the plan is incomplete.
	if notices := installMcpPlanNotices(plan.Mcp); len(notices) > 0 {
		fmt.Fprintf(out, "  %s\n", ansiBold("MCP configs"))
		for _, notice := range notices {
			fmt.Fprintf(out, "    %s %s\n", ansiYellow("⚠"), ansiGray(notice))
		}
		fmt.Fprintln(out)
	}

	if len(plan.FollowUpCommands) > 0 {
		fmt.Fprintf(out, "  %s\n", ansiBold("Apply with"))
		for _, command := range plan.FollowUpCommands {
			fmt.Fprintf(out, "    %s %s\n", ansiGray("$"), ansiWhite(command))
		}
		fmt.Fprintln(out)
	}
	if plan.ConsentRequired {
		fmt.Fprintf(out, "  %s Explicit consent (--yes) is required to apply destructive deprecation cleanup.\n", ansiYellow("⚠"))
	}
}

// installMcpPlanNotices collects the distinct unreadable-config notices from
// the MCP plan. One unreadable path is reported once however many servers were
// asked about it.
func installMcpPlanNotices(entries []mcpPlanEntry) []string {
	var notices []string
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Notice == "" || seen[entry.Notice] {
			continue
		}
		seen[entry.Notice] = true
		notices = append(notices, entry.Notice)
	}
	sort.Strings(notices)
	return notices
}

func planActionGlyph(action string) string {
	switch action {
	case planActionCreate, planActionUpdate, hookActionAdd, "created", "appended", "updated", "relinked", "replaced-file", "migrated":
		return ansiGreen("+")
	case planActionRetire, hookActionRemove, "relocate":
		return ansiYellow("-")
	case planActionConflict, "error":
		return ansiRed("✗")
	case planActionPreserve, hookActionAbsorb, "skipped", "already-correct", planActionNone, "absent", "skip-unmarked":
		return ansiGray("○")
	default:
		return ansiGray("•")
	}
}

// writeHookActionLines reports what a reconcile did, nested under the target it
// ran for. A converged target prints nothing, which is the point: the quiet
// upgrade is the normal one.
func writeHookActionLines(out io.Writer, actions []hookAction) {
	for _, action := range actions {
		fmt.Fprintf(out, "    %s %s %s%s\n", planActionGlyph(action.action), action.action, action.id(), planDetailSuffix(action.detail))
	}
}

func planDetailSuffix(detail string) string {
	if detail == "" {
		return ""
	}
	return " " + ansiGray("— "+detail)
}

func selectedRetiredAdapterPreserveDecisions(options targetInstallOptions, desiredByID map[string]targetAdapterArtifact, installedByID map[string]targetAdapterArtifact, recordedRetired []string, existing []artifactPlanDecision) ([]artifactPlanDecision, error) {
	if options.SelectedArtifactIDs == nil {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, decision := range existing {
		seen[decision.ID] = true
	}
	retired := map[string]bool{}
	for _, id := range recordedRetired {
		retired[id] = true
	}
	var extra []artifactPlanDecision
	ids := make([]string, 0, len(options.SelectedArtifactIDs))
	for id := range options.SelectedArtifactIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if seen[id] || strings.HasPrefix(id, "hook:") {
			continue
		}
		if !strings.HasPrefix(id, "hook-file:") && !strings.HasPrefix(id, "plugin:") {
			continue
		}
		if _, keep := desiredByID[id]; keep {
			continue
		}
		destination := destinationFromAdapterID(id)
		if artifact, owned := installedByID[id]; owned {
			if artifact.Destination != "" {
				destination = artifact.Destination
			}
			path, err := targetAdapterDestination(options, artifact)
			if err != nil {
				return nil, err
			}
			info, err := os.Lstat(path)
			if err == nil && info.Mode().IsRegular() {
				continue
			}
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			extra = append(extra, artifactPlanDecision{ID: id, Kind: artifact.Kind, Destination: destination, Action: planActionPreserve, Detail: "already retired"})
			continue
		}
		if !retired[id] {
			continue
		}
		kind := "hook-file"
		if strings.HasPrefix(id, "plugin:") {
			kind = "plugin"
		}
		extra = append(extra, artifactPlanDecision{ID: id, Kind: kind, Destination: destination, Action: planActionPreserve, Detail: "already retired"})
	}
	return extra, nil
}

func destinationFromAdapterID(id string) string {
	if dest, ok := strings.CutPrefix(id, "hook-file:"); ok {
		return dest
	}
	if dest, ok := strings.CutPrefix(id, "plugin:"); ok {
		return dest
	}
	return ""
}

func writePlanDiff(out io.Writer, diff string) {
	if strings.TrimSpace(diff) == "" {
		return
	}
	for _, line := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		fmt.Fprintf(out, "      %s\n", ansiGray(line))
	}
}

func selectedHookPreserveDecisions(reconciler *hookReconciler, selected map[string]bool, existing []artifactPlanDecision) []artifactPlanDecision {
	if selected == nil {
		return nil
	}
	seen := map[string]bool{}
	for _, decision := range existing {
		seen[decision.ID] = true
	}
	var extra []artifactPlanDecision
	for _, entry := range reconciler.catalog.Entries {
		id := "hook:" + entry.Event + "/" + entry.HookID
		if !selected[id] || seen[id] {
			continue
		}
		extra = append(extra, artifactPlanDecision{
			ID:          id,
			Kind:        "hook-entry",
			Destination: reconciler.path,
			Action:      planActionPreserve,
		})
	}
	return extra
}
