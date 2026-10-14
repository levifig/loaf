package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
)

const implementedIssuesMarker = "<!-- loaf:implemented-issues "

// This declaration is structural input, never an assertion that the URLs are
// real Issues or that they exhaust the implementation. The main agent and Ship
// must verify those facts through the selected tracker connection.
type implementedIssues struct {
	Issues      []string `json:"issues"`
	ReleaseOnly bool     `json:"release_only,omitempty"`
}

var changelogRepositoryRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var changelogCommitIDRE = regexp.MustCompile(`^[a-f0-9]{40}$`)
var changelogIssueRE = regexp.MustCompile(`^https://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/issues/([1-9][0-9]*)$`)
var changelogIssueLinkRE = regexp.MustCompile(`\(\[#([1-9][0-9]*)\]\((https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/issues/[1-9][0-9]*)\)\)`)
var changelogCommentRE = regexp.MustCompile(`(?s)<!--.*?(?:-->|$)`)
var changelogBulletRE = regexp.MustCompile(`^[-*+]\s+(.+)$`)
var changelogReleaseHeadingRE = regexp.MustCompile(`^## \[([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)\](?: - [0-9]{4}-[0-9]{2}-[0-9]{2})?$`)

func parseImplementedIssues(body, repository string) (implementedIssues, error) {
	var declaration implementedIssues
	if strings.Count(body, "<!-- loaf:implemented-issues") != 1 {
		return declaration, fmt.Errorf("PR body must contain exactly one %simplemented Issues JSON declaration followed by -->", implementedIssuesMarker)
	}
	start := strings.Index(body, implementedIssuesMarker)
	if start < 0 {
		return declaration, fmt.Errorf("malformed implemented Issues declaration; use %s{\"issues\":[\"https://github.com/%s/issues/123\"]} -->", implementedIssuesMarker, repository)
	}
	rest := body[start+len(implementedIssuesMarker):]
	end := strings.Index(rest, "-->")
	if end < 0 {
		return declaration, fmt.Errorf("unterminated implemented Issues declaration")
	}
	input := strings.TrimSpace(rest[:end])
	// Reject duplicate object keys rather than accepting JSON's last-key-wins.
	decoder := json.NewDecoder(strings.NewReader(input))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return declaration, fmt.Errorf("implemented Issues declaration must be a JSON object")
	}
	seen := map[string]bool{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return declaration, fmt.Errorf("malformed implemented Issues JSON: %w", err)
		}
		key, ok := keyToken.(string)
		if !ok || seen[key] {
			return declaration, fmt.Errorf("duplicate or invalid implemented Issues field %q", key)
		}
		seen[key] = true
		if key != "issues" && key != "release_only" {
			return declaration, fmt.Errorf("unknown implemented Issues field %q; field names are exact", key)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return declaration, fmt.Errorf("malformed implemented Issues JSON: %w", err)
		}
		if key == "release_only" && string(value) != "true" && string(value) != "false" {
			return declaration, fmt.Errorf("release_only must be a JSON boolean")
		}
	}
	strict := json.NewDecoder(strings.NewReader(input))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&declaration); err != nil {
		return declaration, fmt.Errorf("malformed implemented Issues JSON: %w", err)
	}
	if err := strict.Decode(new(any)); err != io.EOF {
		return declaration, fmt.Errorf("implemented Issues declaration has trailing JSON data")
	}
	if !seen["issues"] || declaration.Issues == nil {
		return declaration, fmt.Errorf("implemented Issues declaration requires an issues array")
	}
	return declaration, validateImplementedIssues(declaration, repository)
}

func validateImplementedIssues(declaration implementedIssues, repository string) error {
	if !changelogRepositoryRE.MatchString(repository) {
		return fmt.Errorf("--repository must be an explicit GitHub owner/repo")
	}
	if declaration.ReleaseOnly {
		if len(declaration.Issues) != 0 {
			return fmt.Errorf("release_only requires an empty issues array; implemented work is not a release-only movement")
		}
		return nil
	}
	if len(declaration.Issues) == 0 {
		return fmt.Errorf("implemented Issues declaration is empty; declare every implemented Issue URL")
	}
	seen := map[string]bool{}
	for _, issue := range declaration.Issues {
		match := changelogIssueRE.FindStringSubmatch(issue)
		if match == nil || match[1] != repository {
			return fmt.Errorf("%q must be a canonical Issue URL in https://github.com/%s/issues/ (PR URLs are not Issue references)", issue, repository)
		}
		if seen[issue] {
			return fmt.Errorf("duplicate implemented Issue %s", issue)
		}
		seen[issue] = true
	}
	return nil
}

type issueChangelogSections struct {
	prefix, unreleased, released string
}

func splitIssueChangelog(text string) (issueChangelogSections, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.SplitAfter(text, "\n")
	start, end, offset, count := -1, len(text), 0, 0
	var fence byte
	fenceLength := 0
	inComment := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		marker, length, tail := issueChangelogFence(line)
		if fence != 0 {
			if marker == fence && length >= fenceLength && strings.TrimSpace(tail) == "" {
				fence = 0
			}
		} else if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
		} else if strings.HasPrefix(trimmed, "<!--") {
			inComment = !strings.Contains(line, "-->")
		} else if marker != 0 && (marker != '`' || !strings.Contains(tail, "`")) {
			fence, fenceLength = marker, length
		} else {
			if trimmed == "## [Unreleased]" {
				count++
				start = offset + len(line)
			} else if start >= 0 && end == len(text) && strings.HasPrefix(line, "## ") {
				end = offset
			}
		}
		offset += len(line)
	}
	if count != 1 || start > end {
		return issueChangelogSections{}, fmt.Errorf("CHANGELOG.md must contain exactly one ## [Unreleased] section")
	}
	return issueChangelogSections{text[:start], text[start:end], text[end:]}, nil
}

// Only actual list prose counts: headings, comments, fences, quotes, and inline
// code examples cannot provide Issue coverage. Entries may wrap over lines.
func issueChangelogEntries(section string) []string {
	section = changelogCommentRE.ReplaceAllString(section, "")
	var entries []string
	current := ""
	var fence byte
	fenceLength := 0
	flush := func() {
		if current != "" {
			entries = append(entries, current)
			current = ""
		}
	}
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		marker, length, tail := issueChangelogFence(line)
		if fence != 0 {
			if marker == fence && length >= fenceLength && strings.TrimSpace(tail) == "" {
				fence = 0
			}
			continue
		}
		if marker != 0 && (marker != '`' || !strings.Contains(tail, "`")) {
			flush()
			fence, fenceLength = marker, length
			continue
		}
		if trimmed == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ">") {
			flush()
			continue
		}
		if match := changelogBulletRE.FindStringSubmatch(line); match != nil {
			flush()
			current = match[1]
		} else if current != "" && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			current += " " + trimmed
		} else {
			flush()
		}
	}
	flush()
	return entries
}

func normalizedChangelogProse(entry string) string {
	entry = changelogIssueLinkRE.ReplaceAllString(entry, "")
	// Formatting, wrapping, punctuation and merely attaching an Issue reference
	// do not turn an unchanged note into implemented-work coverage.
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, entry)
}

func validateIssueChangelogCoverage(base, head string, declaration implementedIssues) error {
	before, err := splitIssueChangelog(base)
	if err != nil {
		return fmt.Errorf("review base: %w", err)
	}
	after, err := splitIssueChangelog(head)
	if err != nil {
		return err
	}
	old := map[string]bool{}
	// Copying a released note back into Unreleased is not new coverage either.
	for _, entry := range issueChangelogEntries(before.unreleased + "\n" + before.released) {
		old[normalizedChangelogProse(entry)] = true
	}
	covered := map[string]bool{}
	for _, entry := range issueChangelogEntries(after.unreleased) {
		normalized := normalizedChangelogProse(entry)
		if normalized == "" || strings.HasPrefix(normalized, "nounreleasedchanges") || old[normalized] {
			continue
		}
		for _, match := range changelogIssueLinkRE.FindAllStringSubmatch(withoutChangelogInlineCode(entry), -1) {
			url := changelogIssueRE.FindStringSubmatch(match[2])
			if url != nil && match[1] == url[2] {
				covered[match[2]] = true
			}
		}
	}
	var missing []string
	for _, issue := range declaration.Issues {
		if !covered[issue] {
			missing = append(missing, issue)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("CHANGELOG.md [Unreleased] lacks a new or substantively updated entry versus the review base for: %s; use ([#N](https://github.com/owner/repo/issues/N)) in the entry prose", strings.Join(missing, ", "))
	}
	return nil
}

func issueChangelogFence(line string) (byte, int, string) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, 0, ""
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == trimmed[0] {
		length++
	}
	if length < 3 {
		return 0, 0, ""
	}
	return trimmed[0], length, trimmed[length:]
}

func withoutChangelogInlineCode(text string) string {
	var visible strings.Builder
	for i := 0; i < len(text); {
		if text[i] != '`' {
			visible.WriteByte(text[i])
			i++
			continue
		}
		start := i
		for i < len(text) && text[i] == '`' {
			i++
		}
		run := i - start
		end := i
		matched := false
		for end < len(text) {
			if text[end] != '`' {
				end++
				continue
			}
			closeStart := end
			for end < len(text) && text[end] == '`' {
				end++
			}
			if end-closeStart == run {
				i = end
				matched = true
				break
			}
		}
		if !matched {
			visible.WriteString(text[start:i])
		}
	}
	return visible.String()
}
