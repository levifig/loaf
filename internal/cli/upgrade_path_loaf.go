package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Generated Cursor, OpenCode, Amp, and Claude Code hook commands stay `loaf …`
// and resolve through PATH. Scoped apply does not publish
// {XDG_DATA_HOME}/loaf/pinned/loaf or rewrite those commands to an absolute
// path. Missing, unusable, or too-old PATH loaf fails closed with
// install/upgrade guidance instead of copying a private binary.
//
// Codex generated hooks, basic-command rules, and CODEX_HOME guidance render
// PATH `loaf` plus classified subcommand prefixes. They do not pin an
// absolute executable or grant a bare `loaf` namespace. Missing or
// incompatible PATH loaf still fails closed with install/upgrade guidance.

var loafCheckHookCommandPattern = regexp.MustCompile("(?:^|[\\s\"'`=|])loaf check --hook ([a-z0-9-]+)")

var loafCheckCommandPattern = regexp.MustCompile("(?:^|[\\s\"'`=|])loaf check ([a-z0-9-]+)")

var loafJournalContextCommandPattern = regexp.MustCompile("(?:^|[\\s\"'`=|])loaf (journal context --from-hook(?: --[a-z0-9-]+)?)")

// pathLoafProbeBudget is the hard wall-clock limit for one PATH loaf
// subprocess. Dry-run and the apply-time recheck both probe the same
// executable; without a deadline a stuck launcher holds the reconcile lock
// indefinitely.
const pathLoafProbeBudget = 2 * time.Second

// pathLoafProbeWaitDelay bounds CombinedOutput after CommandContext
// cancels the immediate launcher. Descendants can keep stdout/stderr open;
// without WaitDelay, Wait blocks until those inherited pipes close.
const pathLoafProbeWaitDelay = 100 * time.Millisecond

var lookPathLoaf = exec.LookPath

var runPathLoaf = func(executable string, args ...string) ([]byte, error) {
	return runPathLoafWithin(context.Background(), pathLoafProbeBudget, executable, args...)
}

func runPathLoafWithin(ctx context.Context, budget time.Duration, executable string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = strings.NewReader("")
	cmd.WaitDelay = pathLoafProbeWaitDelay
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return out, fmt.Errorf("PATH loaf probe timed out after %s: %w", budget, ctx.Err())
	}
	return out, err
}

func isPathLoafProbeTimeout(err error) bool {
	return err != nil && errors.Is(err, context.DeadlineExceeded)
}

type pathLoafProbe struct {
	Path        string
	Version     string
	Help        string
	JournalHelp string
}

type pathLoafRequirements struct {
	hooks     []string
	operators []string
	journals  []string
}

func (req pathLoafRequirements) empty() bool {
	return len(req.hooks) == 0 && len(req.operators) == 0 && len(req.journals) == 0
}

func (req pathLoafRequirements) names() []string {
	names := make([]string, 0, len(req.hooks)+len(req.operators)+len(req.journals))
	for _, id := range req.hooks {
		names = append(names, "loaf check --hook "+id)
	}
	for _, name := range req.operators {
		names = append(names, "loaf check "+name)
	}
	for _, leaf := range req.journals {
		names = append(names, "loaf "+leaf)
	}
	return names
}

func scopedPathLoafPreflightError(plan installDryRunPlan) error {
	required := requiredPathLoafCapabilities(plan)
	if required.empty() {
		return nil
	}
	path, err := lookPathLoaf("loaf")
	if err != nil {
		return pathLoafMissingError(required)
	}
	probe, err := probePathLoafForRequirements(path, required)
	if err != nil {
		if isPathLoafProbeTimeout(err) {
			return fmt.Errorf("PATH loaf at %s timed out during capability probe: %w", path, err)
		}
		return pathLoafUnsupportedError(pathLoafProbe{Path: path}, required.names())
	}
	missing := missingPathLoafCapabilities(probe, required)
	if len(missing) == 0 {
		return nil
	}
	return pathLoafUnsupportedError(probe, missing)
}

func requiredPathLoafCheckHooks(plan installDryRunPlan) []string {
	return requiredPathLoafCapabilities(plan).hooks
}

func requiredPathLoafCapabilities(plan installDryRunPlan) pathLoafRequirements {
	hooks := map[string]bool{}
	operators := map[string]bool{}
	journals := map[string]bool{}
	knownOperators := knownCheckOperatorNames()
	addHook := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || hooks[id] {
			return
		}
		hooks[id] = true
	}
	addOperator := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || !knownOperators[name] || operators[name] {
			return
		}
		operators[name] = true
	}
	addJournal := func(leaf string) {
		leaf = strings.TrimSpace(leaf)
		if leaf == "" || journals[leaf] {
			return
		}
		journals[leaf] = true
	}
	addFromText := func(text string) {
		for _, match := range loafCheckHookCommandPattern.FindAllStringSubmatch(text, -1) {
			addHook(match[1])
		}
		for _, match := range loafCheckCommandPattern.FindAllStringSubmatch(text, -1) {
			addOperator(match[1])
		}
		for _, match := range loafJournalContextCommandPattern.FindAllStringSubmatch(text, -1) {
			addJournal(match[1])
		}
	}
	for _, skill := range plan.Skills {
		if !planActionWritesDesiredContent(skill.Action) {
			continue
		}
		addFromText(capabilityText(skill))
	}
	for _, target := range plan.Targets {
		for _, artifact := range target.Artifacts {
			if !planActionWritesDesiredContent(artifact.Action) {
				continue
			}
			addFromText(capabilityText(artifact))
			if hookID, ok := scopedCheckHookID(artifact.ID); ok {
				addHook(hookID)
			}
		}
	}
	return pathLoafRequirements{
		hooks:     sortedKeys(hooks),
		operators: sortedKeys(operators),
		journals:  sortedKeys(journals),
	}
}

func capabilityText(decision artifactPlanDecision) string {
	if strings.TrimSpace(decision.Desired) != "" {
		return decision.Desired
	}
	return addedDiffContent(decision.Diff)
}

func addedDiffContent(diff string) string {
	var b strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			b.WriteString(line[1:])
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func knownCheckOperatorNames() map[string]bool {
	names := map[string]bool{}
	for _, item := range checkOperatorHelpItems() {
		names[item.Name] = true
	}
	return names
}

func planActionWritesDesiredContent(action string) bool {
	switch action {
	case planActionCreate, planActionUpdate, hookActionAdd:
		return true
	default:
		return false
	}
}

func scopedCheckHookID(artifactID string) (string, bool) {
	rest, ok := strings.CutPrefix(artifactID, "hook:")
	if !ok {
		return "", false
	}
	_, hookID, ok := strings.Cut(rest, "/")
	if !ok || hookID == "" || !validCheckHooks[hookID] {
		return "", false
	}
	return hookID, true
}

func probePathLoaf() (pathLoafProbe, error) {
	path, err := lookPathLoaf("loaf")
	if err != nil {
		return pathLoafProbe{}, err
	}
	return probePathLoafAt(path)
}

func probePathLoafAt(path string) (pathLoafProbe, error) {
	return probePathLoafForRequirements(path, pathLoafRequirements{hooks: []string{"_"}})
}

func probePathLoafForRequirements(path string, required pathLoafRequirements) (pathLoafProbe, error) {
	version, err := probePathLoafVersion(path)
	if err != nil {
		return pathLoafProbe{}, fmt.Errorf("PATH loaf at %s failed version probe: %w", path, err)
	}
	probe := pathLoafProbe{Path: path, Version: version}
	if len(required.hooks) > 0 || len(required.operators) > 0 {
		helpOut, err := runReadOnlyPathLoaf(path, "check", "--help")
		if err != nil {
			return pathLoafProbe{}, fmt.Errorf("PATH loaf at %s failed check-help probe: %w", path, err)
		}
		probe.Help = helpOut
	}
	if len(required.journals) > 0 {
		helpOut, err := runReadOnlyPathLoaf(path, "journal", "context", "--help")
		if err != nil {
			return pathLoafProbe{}, fmt.Errorf("PATH loaf at %s failed journal-context-help probe: %w", path, err)
		}
		probe.JournalHelp = helpOut
	}
	return probe, nil
}

func probePathLoafVersion(path string) (string, error) {
	firstOut, firstErr := runReadOnlyPathLoaf(path, "version")
	if isPathLoafProbeTimeout(firstErr) {
		return "", firstErr
	}
	if identity, ok := pathLoafVersionIdentity(firstOut); ok {
		return identity, nil
	}
	secondOut, secondErr := runReadOnlyPathLoaf(path, "--version")
	if isPathLoafProbeTimeout(secondErr) {
		return "", secondErr
	}
	if identity, ok := pathLoafVersionIdentity(secondOut); ok {
		return identity, nil
	}
	if firstErr != nil && secondErr != nil {
		return "", firstErr
	}
	sample := strings.TrimSpace(firstOut)
	if sample == "" {
		sample = strings.TrimSpace(secondOut)
	}
	if sample == "" {
		return "", fmt.Errorf("inconclusive version identity")
	}
	return "", fmt.Errorf("inconclusive version identity: %s", summarizePathLoafOutput(sample))
}

func pathLoafVersionIdentity(output string) (string, bool) {
	parsed := parsePathLoafVersion(output)
	if parsed == "" {
		return "", false
	}
	if _, ok := parseUpgradeSemver(parsed); !ok {
		return "", false
	}
	return parsed, true
}

func runReadOnlyPathLoaf(executable string, args ...string) (string, error) {
	out, err := runPathLoaf(executable, args...)
	text := string(out)
	if err != nil {
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			return "", fmt.Errorf("%w (%s)", err, summarizePathLoafOutput(trimmed))
		}
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("inconclusive empty output")
	}
	if pathLoafOutputLooksUnsupported(text) {
		return "", fmt.Errorf("inconclusive capability output: %s", summarizePathLoafOutput(text))
	}
	return text, nil
}

func summarizePathLoafOutput(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return text
	}
	if len(fields) > 12 {
		return strings.Join(fields[:12], " ")
	}
	return strings.Join(fields, " ")
}

func parsePathLoafVersion(output string) string {
	stripped := stripSimpleANSI(output)
	for _, line := range strings.Split(stripped, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "loaf" {
			return strings.TrimSuffix(fields[1], ",")
		}
	}
	return ""
}

func stripSimpleANSI(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == 0x1b && i+1 < len(value) && value[i+1] == '[' {
			i += 2
			for i < len(value) && (value[i] < 0x40 || value[i] > 0x7e) {
				i++
			}
			continue
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

func missingPathLoafCapabilities(probe pathLoafProbe, required pathLoafRequirements) []string {
	var missing []string
	for _, id := range required.hooks {
		if pathLoafHelpListsToken(probe.Help, id) {
			continue
		}
		missing = append(missing, "loaf check --hook "+id)
	}
	for _, name := range required.operators {
		if pathLoafRecognizesCheckOperator(probe, name) {
			continue
		}
		missing = append(missing, "loaf check "+name)
	}
	for _, leaf := range required.journals {
		if pathLoafRecognizesJournalLeaf(probe, leaf) {
			continue
		}
		missing = append(missing, "loaf "+leaf)
	}
	return missing
}

func pathLoafRecognizesJournalLeaf(probe pathLoafProbe, leaf string) bool {
	if strings.TrimSpace(probe.JournalHelp) == "" {
		return false
	}
	for _, token := range strings.Fields(leaf) {
		if !strings.HasPrefix(token, "--") {
			continue
		}
		if !pathLoafHelpListsToken(probe.JournalHelp, token) {
			return false
		}
	}
	return true
}

func pathLoafRecognizesCheckOperator(probe pathLoafProbe, name string) bool {
	if pathLoafHelpListsToken(probe.Help, name) {
		return true
	}
	out, err := runReadOnlyPathLoaf(probe.Path, "check", name, "--help")
	if err != nil {
		return false
	}
	return strings.Contains(out, "loaf check "+name)
}

func pathLoafHelpListsHook(help string, hookID string) bool {
	return pathLoafHelpListsToken(help, hookID)
}

func pathLoafHelpListsToken(help string, token string) bool {
	for _, field := range strings.FieldsFunc(help, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ':'
	}) {
		if field == token {
			return true
		}
	}
	return false
}

func pathLoafOutputLooksUnsupported(output string) bool {
	combined := string(output)
	switch {
	case strings.Contains(combined, "Unknown hook"):
		return true
	case strings.Contains(combined, "unknown check option"):
		return true
	case strings.Contains(combined, "unknown check subcommand"):
		return true
	case strings.Contains(combined, "unknown command"):
		return true
	case strings.Contains(combined, "This worktree has unmigrated agentic state"):
		return true
	case strings.Contains(combined, "dyld: Library not loaded"):
		return true
	default:
		return false
	}
}

func formatRequiredCapabilities(req pathLoafRequirements) string {
	return strings.Join(req.names(), ", ")
}

func pathLoafInstallUpgradeGuidance() string {
	native := fmt.Sprintf("bin/native/%s-%s/loaf", runtime.GOOS, runtime.GOARCH)
	return "Install or upgrade the loaf binary on PATH (Homebrew: `brew tap levifig/tap && brew install loaf`, then `brew upgrade loaf` after releases; or rebuild this checkout and put " + native + " on PATH). Generated hook commands stay `loaf …` and resolve through PATH; Loaf will not copy a private binary into XDG as a substitute."
}

func pathLoafMissingError(required pathLoafRequirements) error {
	return fmt.Errorf("PATH loaf is missing; selected artifacts require %s. %s", formatRequiredCapabilities(required), pathLoafInstallUpgradeGuidance())
}

func pathLoafUnsupportedError(probe pathLoafProbe, missing []string) error {
	ident := probe.Path
	if probe.Version != "" {
		ident = fmt.Sprintf("%s (%s)", probe.Path, probe.Version)
	}
	return fmt.Errorf("PATH loaf at %s is too old or lacks %s. %s", ident, strings.Join(missing, ", "), pathLoafInstallUpgradeGuidance())
}
