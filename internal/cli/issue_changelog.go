package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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

// Linear short URLs are stable references; title slugs and tracking suffixes
// are deliberately not accepted as alternative spellings of the same Issue.
var changelogLinearIssueRE = regexp.MustCompile(`^https://linear\.app/([a-z0-9]+(?:-[a-z0-9]+)*)/issue/([A-Z][A-Z0-9]*-[1-9][0-9]*)$`)
var changelogLinearIssueLinkRE = regexp.MustCompile(`\[([A-Z][A-Z0-9]*-[1-9][0-9]*)\]\((https://linear\.app/[a-z0-9]+(?:-[a-z0-9]+)*/issue/[A-Z][A-Z0-9]*-[1-9][0-9]*)\)`)
var changelogCommentRE = regexp.MustCompile(`(?s)<!--.*?(?:-->|$)`)
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
		if match != nil {
			if match[1] != repository {
				return fmt.Errorf("%q must be a canonical GitHub Issue URL in https://github.com/%s/issues/", issue, repository)
			}
		} else if !changelogLinearIssueRE.MatchString(issue) {
			return fmt.Errorf("%q is unsupported: use a canonical GitHub Issue URL in https://github.com/%s/issues/ or a short Linear Issue URL https://linear.app/workspace/issue/TEAM-N (PR URLs, title slugs, queries and fragments are not supported)", issue, repository)
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

// The same Markdown entries serve authoring reminders and coverage checks.
func issueChangelogEntries(section string) []string {
	var entries []string
	for _, entry := range parsedChangelogEntries(section) {
		entries = append(entries, entry.prose)
	}
	return entries
}

func normalizedChangelogProse(entry string) string {
	entry = changelogCommentRE.ReplaceAllString(entry, "")
	entry = changelogIssueLinkRE.ReplaceAllString(entry, "")
	entry = changelogLinearIssueLinkRE.ReplaceAllString(entry, "")
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
	// Section and AST offsets must address exactly the same source bytes.
	base = strings.ReplaceAll(base, "\r\n", "\n")
	head = strings.ReplaceAll(head, "\r\n", "\n")
	before, err := splitIssueChangelog(base)
	if err != nil {
		return fmt.Errorf("review base: %w", err)
	}
	after, err := splitIssueChangelog(head)
	if err != nil {
		return err
	}
	old := map[string]bool{}
	// Compare entries from the same parser, including nested and released
	// notes, so an unchanged link cannot credit another new unlinked entry.
	for _, entry := range parsedChangelogEntries(base) {
		if entry.start >= len(before.prefix) {
			old[normalizedChangelogProse(entry.prose)] = true
		}
	}
	covered := map[string]bool{}
	end := len(after.prefix) + len(after.unreleased)
	for _, entry := range parsedChangelogEntries(head) {
		if entry.start < len(after.prefix) || entry.end > end {
			continue
		}
		normalized := normalizedChangelogProse(entry.prose)
		if normalized == "" || strings.HasPrefix(normalized, "nounreleasedchanges") || old[normalized] {
			continue
		}
		for issue := range entry.issues {
			covered[issue] = true
		}
	}
	var missing []string
	for _, issue := range declaration.Issues {
		if !covered[issue] {
			missing = append(missing, issue)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("CHANGELOG.md [Unreleased] lacks a new or substantively updated entry versus the review base for: %s; use ([#N](https://github.com/owner/repo/issues/N)) or [TEAM-N](https://linear.app/workspace/issue/TEAM-N) in the entry prose", strings.Join(missing, ", "))
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

type parsedChangelogEntry struct {
	start, end int
	prose      string
	issues     map[string]bool
}

// Parse original source once. Each actual list paragraph owns its links and
// source span; HTML blocks, images and reference definitions keep their context.
func parsedChangelogEntries(document string) []parsedChangelogEntry {
	source := []byte(document)
	root := goldmark.DefaultParser().Parse(text.NewReader(source))
	var entries []parsedChangelogEntry
	var visit func(ast.Node, bool)
	visit = func(node ast.Node, inList bool) {
		switch node.Kind() {
		case ast.KindImage, ast.KindBlockquote, ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindHTMLBlock, ast.KindHeading:
			return
		case ast.KindList:
			if node.(*ast.List).IsOrdered() {
				return
			}
		case ast.KindListItem:
			inList = true
		case ast.KindParagraph, ast.KindTextBlock:
			lines := node.Lines()
			if inList && lines.Len() != 0 {
				entries = append(entries, parsedChangelogEntry{
					start:  lines.At(0).Start,
					end:    lines.At(lines.Len() - 1).Stop,
					prose:  string(lines.Value(source)),
					issues: linkedChangelogIssues(node, source),
				})
			}
			return
		}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			visit(child, inList)
		}
	}
	visit(root, false)
	return entries
}

func linkedChangelogIssues(paragraph ast.Node, source []byte) map[string]bool {
	issues := map[string]bool{}
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		if node.Kind() == ast.KindImage {
			return
		}
		if link, ok := node.(*ast.Link); ok {
			label, plain := link.FirstChild().(*ast.Text)
			if !plain || label.NextSibling() != nil || link.Reference != nil {
				return
			}
			key, target := string(label.Value(source)), string(link.Destination)
			raw := "[" + key + "](" + target + ")"
			offset := label.Segment.Start - 1
			if match := changelogIssueRE.FindStringSubmatch(target); match != nil && key == "#"+match[2] {
				raw = "(" + raw + ")"
				offset--
			} else if match := changelogLinearIssueRE.FindStringSubmatch(target); match == nil || key != match[2] {
				return
			}
			// Broader CommonMark forms (titles, reference links, escaped
			// labels or destinations) are not canonical coverage references.
			if offset >= 0 && strings.HasPrefix(string(source[offset:]), raw) {
				issues[target] = true
			}
			return
		}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			visit(child)
		}
	}
	visit(paragraph)
	return issues
}
