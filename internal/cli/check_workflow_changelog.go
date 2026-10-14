package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The first concrete policy is GitHub-specific. A repository opts into it in
// its own tracked PR template; merely hosting code on GitHub does not elect
// GitHub as canonical tracker. Explicit CLI/CI invocations never skip validation.

func repositoryUsesIssueChangelog(root string, bases ...string) bool {
	for _, ref := range append([]string{"HEAD"}, bases...) {
		if ref == "" {
			continue
		}
		commit, err := resolveChangelogCommit(root, ref)
		if err != nil {
			continue
		}
		for _, path := range []string{".github/PULL_REQUEST_TEMPLATE.md", ".github/pull_request_template.md"} {
			body, err := readChangelogCandidateFile(root, commit, path)
			if err == nil && strings.Contains(body, implementedIssuesMarker) {
				return true
			}
		}
	}
	return false
}

func advisoryWorkflowPrePR(command, root string) checkResult {
	result := checkResult{Passed: true}
	if !strings.Contains(command, "gh pr create") {
		return result
	}
	body, err := readRegularFile(filepath.Join(root, "CHANGELOG.md"), projectFileReadLimit)
	if err != nil {
		message := "Could not read CHANGELOG.md"
		if os.IsNotExist(err) {
			message = "CHANGELOG.md not found"
		}
		result.Warnings = append(result.Warnings, message)
	} else if section, err := splitIssueChangelog(string(body)); err != nil {
		result.Warnings = append(result.Warnings, err.Error())
	} else if len(issueChangelogEntries(section.unreleased)) == 0 {
		result.Warnings = append(result.Warnings, "CHANGELOG.md [Unreleased] section is empty")
	}
	validatePRCreateCommand(command, &result)
	result.Warnings = append(result.Warnings, result.Errors...)
	result.Errors = nil
	result.Passed = true
	result.Blocked = false
	return result
}

// Recognize literal gh invocations without evaluating shell expansions. Shell
// setup, dynamic flags and stdin bodies need a separate, explicit local check;
// the hook must never attest different context from the command being created.
func workflowChangelogRequest(command, root string) (issueChangelogRequest, bool, error) {
	request := issueChangelogRequest{head: "HEAD"}
	segments := splitSafetySegments(command, true)
	for _, segment := range segments {
		tokens, ok := hookCommandTokens(segment)
		if !ok {
			if strings.Contains(segment, "gh pr create") {
				return request, true, fmt.Errorf("cannot parse PR creation; use a literal gh pr create command")
			}
			continue
		}
		if len(tokens) < 3 {
			continue
		}
		if filepath.Base(tokens[0]) != "gh" || tokens[1] != "pr" || tokens[2] != "create" {
			for i := 1; i+2 < len(tokens); i++ {
				if filepath.Base(tokens[i]) == "gh" && tokens[i+1] == "pr" && tokens[i+2] == "create" {
					return request, true, fmt.Errorf("wrapped PR creation has ambiguous context; invoke literal gh pr create separately")
				}
			}
			continue
		}
		if len(segments) != 1 {
			return request, true, fmt.Errorf("run gh pr create separately from shell setup so review context is unambiguous")
		}
		body, bodyFile := "", ""
		fill := false
		seen := map[string]bool{}
		for i := 3; i < len(tokens); i++ {
			key, value, hasValue := strings.Cut(tokens[i], "=")
			switch key {
			case "--repo", "-R":
				key = "repo"
			case "--base", "-B":
				key = "base"
			case "--body", "-b":
				key = "body"
			case "--body-file", "-F":
				key = "body-file"
			case "--fill", "--fill-first", "--fill-verbose":
				fill = true
				continue
			case "--title", "-t", "--reviewer", "-r", "--assignee", "-a", "--label", "-l", "--milestone", "-m", "--project", "-p":
				if !hasValue {
					i++
				}
				continue
			case "--draft", "-d", "--no-maintainer-edit":
				continue
			default:
				return request, true, fmt.Errorf("unsupported or dynamic PR creation flag %q; use literal --repo, --base and --body/--body-file", key)
			}
			if seen[key] {
				return request, true, fmt.Errorf("ambiguous repeated PR creation flag %s", key)
			}
			seen[key] = true
			if !hasValue {
				i++
				if i >= len(tokens) {
					return request, true, fmt.Errorf("missing PR creation flag value for %s", key)
				}
				value = tokens[i]
			}
			if strings.ContainsAny(value, "$`") || strings.HasPrefix(value, "-") {
				return request, true, fmt.Errorf("PR creation %s must be a nonempty literal value", key)
			}
			switch key {
			case "repo":
				request.repository = value
			case "base":
				request.base = value
			case "body":
				body = value
			case "body-file":
				bodyFile = value
			}
		}
		if request.repository == "" || request.base == "" {
			return request, true, fmt.Errorf("gh pr create requires explicit --repo owner/repo and --base for issue-changelog coverage")
		}
		if seen["body"] && seen["body-file"] {
			return request, true, fmt.Errorf("choose one PR body source")
		}
		if !seen["body"] && !seen["body-file"] && fill {
			return request, true, fmt.Errorf("--fill cannot establish implemented Issues; supply an explicit --body or --body-file declaration")
		}
		if bodyFile != "" {
			if bodyFile == "-" {
				return request, true, fmt.Errorf("stdin PR bodies cannot be inspected by the hook; use a regular --body-file")
			}
			var err error
			body, err = readCheckOperatorFile(resolveCheckOperatorPath(root, bodyFile))
			if err != nil {
				return request, true, fmt.Errorf("cannot read PR body file: %w", err)
			}
		}
		var err error
		request.declaration, err = parseImplementedIssues(body, request.repository)
		return request, true, err
	}
	return request, false, nil
}
