package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

type issueChangelogRequest struct {
	repository, base, head string
	declaration            implementedIssues
}

func writeCheckIssueChangelogHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check issue-changelog --repository owner/repo --base <ref> --head <ref|WORKTREE> (--pr-body-file <file> | --issue <URL>... | --release-only) [--json]", "Require changed Unreleased entries for every explicitly declared implemented Issue. --event-file accepts a GitHub pull_request event instead of base/head/declaration flags. Reads only; WORKTREE includes local staged/unstaged content, while pre-PR and CI check committed HEAD. This validates structural coverage, not tracker identity or semantic accuracy.", "--repository    Explicit GitHub owner/repo\n--base          Actual review base ref or commit\n--head          Candidate ref or WORKTREE\n--pr-body-file  Read the single implemented Issues JSON declaration\n--issue         Local equivalent: repeat a canonical Issue URL for each implemented Issue\n--release-only  Local equivalent for exact promotion of existing notes only\n--event-file    GitHub pull_request event: read exact base/head/body as data\n--json          Output pass/fail and errors as JSON")
}

type changelogIssueFlags []string

func (f *changelogIssueFlags) String() string         { return strings.Join(*f, ",") }
func (f *changelogIssueFlags) Set(value string) error { *f = append(*f, value); return nil }

func (r Runner) runCheckIssueChangelog(args []string, out, errOut io.Writer, root string) error {
	var request issueChangelogRequest
	var bodyFile, eventFile string
	var issues changelogIssueFlags
	var releaseOnly, jsonOutput bool
	flags := flag.NewFlagSet("issue-changelog", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&request.repository, "repository", "", "")
	flags.StringVar(&request.base, "base", "", "")
	flags.StringVar(&request.head, "head", "", "")
	flags.StringVar(&bodyFile, "pr-body-file", "", "")
	flags.StringVar(&eventFile, "event-file", "", "")
	flags.Var(&issues, "issue", "")
	flags.BoolVar(&releaseOnly, "release-only", false, "")
	flags.BoolVar(&jsonOutput, "json", false, "")
	err := uniqueIssueChangelogFlags(args)
	if err == nil {
		err = flags.Parse(args)
	}
	if err == nil && flags.NArg() != 0 {
		err = fmt.Errorf("unexpected positional arguments")
	}
	if err == nil {
		sources := 0
		if bodyFile != "" {
			sources++
		}
		if eventFile != "" {
			sources++
		}
		if len(issues) != 0 || releaseOnly {
			sources++
		}
		if sources != 1 {
			err = fmt.Errorf("choose exactly one declaration input: --pr-body-file, --event-file, or --issue / --release-only")
		}
	}
	if err == nil && eventFile != "" {
		if request.base != "" || request.head != "" {
			err = fmt.Errorf("--event-file supplies exact base/head; do not combine it with --base or --head")
		} else {
			request, err = readIssueChangelogEvent(resolveCheckOperatorPath(root, eventFile), request.repository)
		}
	} else if err == nil && bodyFile != "" {
		var body string
		body, err = readCheckOperatorFile(resolveCheckOperatorPath(root, bodyFile))
		if err == nil {
			request.declaration, err = parseImplementedIssues(body, request.repository)
		}
	} else if err == nil {
		request.declaration = implementedIssues{Issues: []string(issues), ReleaseOnly: releaseOnly}
	}
	if err == nil {
		err = checkIssueChangelog(root, request)
	}
	result := checkResult{Passed: err == nil}
	if err != nil {
		result.Blocked = true
		result.Errors = []string{err.Error()}
	}
	return writeCheckOperatorResult(out, errOut, "issue-changelog", result, jsonOutput)
}

func uniqueIssueChangelogFlags(args []string) error {
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		name, _, hasValue := strings.Cut(args[i], "=")
		if !strings.HasPrefix(name, "--") {
			if strings.HasPrefix(name, "-") {
				return fmt.Errorf("use --long-option spelling; ambiguous option %q is not accepted", name)
			}
			continue
		}
		if name != "--issue" && seen[name] {
			return fmt.Errorf("ambiguous repeated option %s", name)
		}
		seen[name] = true
		if !hasValue && name != "--json" && name != "--release-only" {
			i++
		}
	}
	return nil
}

func readIssueChangelogEvent(path, repository string) (issueChangelogRequest, error) {
	request := issueChangelogRequest{repository: repository}
	body, err := readCheckOperatorFile(path)
	if err != nil {
		return request, err
	}
	var event struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		PullRequest *struct {
			Body *string `json:"body"`
			Base struct {
				SHA  string `json:"sha"`
				Repo struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"base"`
			Head struct {
				SHA string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return request, fmt.Errorf("invalid GitHub event: %w", err)
	}
	if event.PullRequest == nil || event.PullRequest.Body == nil || repository == "" || event.Repository.FullName != repository || event.PullRequest.Base.Repo.FullName != repository {
		return request, fmt.Errorf("event must contain a pull_request body and the exact expected base repository %q", repository)
	}
	request.base, request.head = event.PullRequest.Base.SHA, event.PullRequest.Head.SHA
	// Event metadata must be immutable commit IDs, never arbitrary revision syntax.
	if !changelogCommitIDRE.MatchString(request.base) || !changelogCommitIDRE.MatchString(request.head) {
		return request, fmt.Errorf("event is missing exact base/head commit IDs")
	}
	request.declaration, err = parseImplementedIssues(*event.PullRequest.Body, repository)
	return request, err
}

func checkIssueChangelog(root string, request issueChangelogRequest) error {
	if err := validateImplementedIssues(request.declaration, request.repository); err != nil {
		return err
	}
	if request.base == "" || request.head == "" {
		return fmt.Errorf("explicit review --base and candidate --head are required")
	}
	base, err := resolveChangelogCommit(root, request.base)
	if err != nil {
		return fmt.Errorf("review base unavailable: %w", err)
	}
	head := request.head
	if head != "WORKTREE" {
		head, err = resolveChangelogCommit(root, head)
		if err != nil {
			return fmt.Errorf("candidate unavailable: %w", err)
		}
	}
	before, err := readChangelogCandidateFile(root, base, "CHANGELOG.md")
	if err != nil {
		return fmt.Errorf("review base CHANGELOG.md unavailable: %w", err)
	}
	after, err := readChangelogCandidateFile(root, head, "CHANGELOG.md")
	if err != nil {
		return fmt.Errorf("candidate CHANGELOG.md unavailable: %w", err)
	}
	if request.declaration.ReleaseOnly {
		return validateChangelogReleaseMovement(root, base, head, before, after)
	}
	return validateIssueChangelogCoverage(before, after, request.declaration)
}

func changelogGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed (verify the supplied review context)", args[0])
	}
	if len(output) > int(projectFileReadLimit) {
		return "", fmt.Errorf("Git result exceeds file read limit")
	}
	return string(output), nil
}

func resolveChangelogCommit(root, ref string) (string, error) {
	value, err := changelogGit(root, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func readChangelogCandidateFile(root, candidate, path string) (string, error) {
	if candidate == "WORKTREE" {
		return readCheckOperatorFile(filepath.Join(root, path))
	}
	mode, err := changelogGit(root, "ls-tree", candidate, "--", path)
	if err != nil || !(strings.HasPrefix(mode, "100644 ") || strings.HasPrefix(mode, "100755 ")) {
		return "", fmt.Errorf("%s not found as a regular tracked file", path)
	}
	return changelogGit(root, "show", candidate+":"+path)
}

func validateChangelogReleaseMovement(root, base, head, before, after string) error {
	paths, err := changelogGit(root, "diff", "--name-only", "-z", base, changelogHeadArgument(head), "--")
	if head == "WORKTREE" {
		paths, err = changelogGit(root, "diff", "--name-only", "-z", base, "--")
	}
	if err != nil {
		return err
	}
	if head == "WORKTREE" {
		untracked, err := changelogGit(root, "ls-files", "--others", "--exclude-standard", "-z")
		if err != nil {
			return err
		}
		paths += untracked
	}
	versionChanged, changelogChanged := false, false
	var generatedPaths []string
	for _, path := range strings.Split(paths, "\x00") {
		switch path {
		case "":
		case "CHANGELOG.md":
			changelogChanged = true
		case "package.json":
			versionChanged = true
		default:
			generatedPaths = append(generatedPaths, path)
		}
	}
	if !changelogChanged {
		return fmt.Errorf("release-only requires an actual CHANGELOG.md promotion")
	}
	old, err := splitIssueChangelog(before)
	if err != nil {
		return err
	}
	current, err := splitIssueChangelog(after)
	if err != nil {
		return err
	}
	empty := strings.TrimSpace(current.unreleased)
	if empty != "" && empty != "- _No unreleased changes yet._" {
		return fmt.Errorf("release-only must leave Unreleased empty or its canonical stub")
	}
	hasNotes := false
	for _, entry := range issueChangelogEntries(old.unreleased) {
		prose := normalizedChangelogProse(entry)
		if prose != "" && !strings.HasPrefix(prose, "nounreleasedchanges") {
			hasNotes = true
		}
	}
	if old.prefix != current.prefix || !hasNotes {
		return fmt.Errorf("release-only requires existing landed Unreleased notes and an unchanged changelog prefix")
	}
	line, rest, ok := strings.Cut(current.released, "\n")
	match := changelogReleaseHeadingRE.FindStringSubmatch(line)
	if !ok || match == nil {
		return fmt.Errorf("release-only requires one new versioned release heading")
	}
	// Reconstruct the only permitted movement, preserving note bytes and all
	// older release history. No title/tag/commit-subject exemption exists.
	moved := strings.TrimSpace(old.unreleased)
	want := moved + "\n\n" + strings.TrimSpace(old.released)
	if strings.TrimSpace(rest) != strings.TrimSpace(want) {
		return fmt.Errorf("release-only must move existing Unreleased notes unchanged; new, missing, or rewritten notes are implemented work")
	}
	if versionChanged {
		beforeVersion, err := readChangelogCandidateFile(root, base, "package.json")
		if err != nil {
			return err
		}
		afterVersion, err := readChangelogCandidateFile(root, head, "package.json")
		if err != nil {
			return err
		}
		var oldPackage, newPackage map[string]any
		if json.Unmarshal([]byte(beforeVersion), &oldPackage) != nil || json.Unmarshal([]byte(afterVersion), &newPackage) != nil || oldPackage == nil || newPackage == nil {
			return fmt.Errorf("release-only package.json must be a JSON object")
		}
		if newPackage["version"] != match[1] || oldPackage["version"] == newPackage["version"] {
			return fmt.Errorf("release-only package.json version must change to the new release heading")
		}
		oldVersion, ok := oldPackage["version"].(string)
		if !ok || oldVersion == "" {
			return fmt.Errorf("release-only requires a valid prior package version")
		}
		delete(oldPackage, "version")
		delete(newPackage, "version")
		if !reflect.DeepEqual(oldPackage, newPackage) {
			return fmt.Errorf("release-only package.json may change only version")
		}
		for _, path := range generatedPaths {
			if err := validateReleaseVersionOutput(root, base, head, path, oldVersion, match[1]); err != nil {
				return err
			}
		}
	} else if len(generatedPaths) != 0 {
		return fmt.Errorf("release-only movement cannot include implementation or other changed files: %s", generatedPaths[0])
	}
	return nil
}

func changelogHeadArgument(head string) string {
	if head == "WORKTREE" {
		return "HEAD"
	}
	return head
}
