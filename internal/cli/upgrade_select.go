package cli

import (
	"fmt"
	"sort"
	"strings"
)

const scopedSkillsTarget = "skills"

// scopedArtifactRef is one upgrade selection. Artifact IDs are not globally
// unique, so a selection is always a target (or the shared skills store) plus
// the plan ID. The first '/' separates the qualifier from the ID, so
// cursor/hook:preToolUse/validate-sql-safety stays one identity.
type scopedArtifactRef struct {
	Target string
	ID     string
}

func (r scopedArtifactRef) String() string {
	return r.Target + "/" + r.ID
}

func parseScopedArtifactRef(value string) (scopedArtifactRef, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return scopedArtifactRef{}, fmt.Errorf("--select requires target/id (IDs are not globally unique)")
	}
	target, id, ok := strings.Cut(value, "/")
	if !ok || target == "" || id == "" {
		return scopedArtifactRef{}, fmt.Errorf("selection %q must be target/id (for example skills/skill:foundations or cursor/hook:postToolUse/kb-staleness-nudge)", value)
	}
	if target != scopedSkillsTarget && !isValidInstallTarget(target) {
		return scopedArtifactRef{}, fmt.Errorf("unknown selection target %q in %q (valid: %s, %s)", target, value, scopedSkillsTarget, strings.Join(installValidTargets, ", "))
	}
	return scopedArtifactRef{Target: target, ID: id}, nil
}

func parseScopedArtifactRefs(values []string) ([]scopedArtifactRef, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	refs := make([]scopedArtifactRef, 0, len(values))
	for _, value := range values {
		ref, err := parseScopedArtifactRef(value)
		if err != nil {
			return nil, err
		}
		key := ref.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Target != refs[j].Target {
			return refs[i].Target < refs[j].Target
		}
		return refs[i].ID < refs[j].ID
	})
	return refs, nil
}

func (r scopedArtifactRef) matchesSkill(decision artifactPlanDecision) bool {
	return r.Target == scopedSkillsTarget && decision.Kind == "skill" && decision.ID == r.ID
}

func (r scopedArtifactRef) matchesTarget(target string, decision artifactPlanDecision) bool {
	return r.Target == target && decision.ID == r.ID
}

func selectedHookIDsForTarget(refs []scopedArtifactRef, target string) map[string]bool {
	ids := map[string]bool{}
	for _, ref := range refs {
		if ref.Target == target && strings.HasPrefix(ref.ID, "hook:") {
			ids[ref.ID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func isCodexPolicyArtifactID(id string) bool {
	return strings.HasPrefix(id, "codex-rule:")
}

func isCodexPolicyDecision(decision artifactPlanDecision) bool {
	return decision.Kind == "codex-rule" || decision.Kind == "codex-guidance"
}

func selectedAdapterIDsForTarget(refs []scopedArtifactRef, target string) []string {
	var ids []string
	for _, ref := range refs {
		if ref.Target == target && !strings.HasPrefix(ref.ID, "hook:") && !isCodexPolicyArtifactID(ref.ID) {
			ids = append(ids, ref.ID)
		}
	}
	return ids
}

func selectedCodexPolicyDecisions(artifacts []artifactPlanDecision) []artifactPlanDecision {
	var decisions []artifactPlanDecision
	for _, decision := range artifacts {
		if isCodexPolicyDecision(decision) {
			decisions = append(decisions, decision)
		}
	}
	return decisions
}

func scopedPolicyActionWrites(action string) bool {
	switch action {
	case planActionCreate, planActionUpdate, planActionRetire:
		return true
	default:
		return false
	}
}

func selectedSkillNames(refs []scopedArtifactRef) []string {
	var names []string
	for _, ref := range refs {
		if ref.Target != scopedSkillsTarget {
			continue
		}
		name, ok := strings.CutPrefix(ref.ID, "skill:")
		if !ok || name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func filterPlanToSelections(plan installDryRunPlan, refs []scopedArtifactRef) (installDryRunPlan, error) {
	matched := map[string]bool{}

	var skills []artifactPlanDecision
	for _, decision := range plan.Skills {
		for _, ref := range refs {
			if ref.matchesSkill(decision) {
				skills = append(skills, decision)
				matched[ref.String()] = true
			}
		}
	}

	var targets []targetDistributionPlan
	for _, target := range plan.Targets {
		var artifacts []artifactPlanDecision
		blocked := false
		for _, decision := range target.Artifacts {
			for _, ref := range refs {
				if !ref.matchesTarget(target.Target, decision) {
					continue
				}
				if decision.Kind == "hook-legacy" {
					return installDryRunPlan{}, fmt.Errorf("selection %s is a legacy whole-target artifact; run `loaf build` before selecting individual artifacts", ref.String())
				}
				artifacts = append(artifacts, decision)
				matched[ref.String()] = true
				if decision.Action == planActionConflict {
					blocked = true
				}
			}
		}
		if len(artifacts) == 0 {
			continue
		}
		target.Artifacts = artifacts
		target.Blocked = blocked
		target.Note = ""
		targets = append(targets, target)
	}

	var missing []string
	for _, ref := range refs {
		if !matched[ref.String()] {
			missing = append(missing, ref.String())
		}
	}
	if len(missing) > 0 {
		return installDryRunPlan{}, fmt.Errorf("unknown scoped selection(s): %s", strings.Join(missing, ", "))
	}

	plan.Skills = skills
	plan.Targets = targets
	plan.Deprecations = nil
	plan.ProjectPart = nil
	plan.ProjectFiles = nil
	plan.Mcp = nil
	plan.ConsentRequired = false
	plan.Selections = make([]string, 0, len(refs))
	for _, ref := range refs {
		plan.Selections = append(plan.Selections, ref.String())
	}
	plan.VersionStamp = "none"
	return plan, nil
}

func scopedFollowUpCommand(refs []scopedArtifactRef) string {
	parts := []string{"loaf", upgradeCommandName}
	for _, ref := range refs {
		parts = append(parts, "--select", ref.String())
	}
	return strings.Join(parts, " ")
}

func selectedArtifactIDsForTarget(refs []scopedArtifactRef, target string) map[string]bool {
	ids := map[string]bool{}
	for _, ref := range refs {
		if ref.Target == target {
			ids[ref.ID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}
