package cli

import (
	"io"
	"regexp"
	"slices"
	"strings"
)

var (
	microChangelogHeadingRE = regexp.MustCompile(`(?m)^## Changelog`)
	microChangelogNextRE    = regexp.MustCompile(`(?m)^## `)
	microChangelogEntryRE   = regexp.MustCompile(`(?m)^- (.+)$`)
	microChangelogDatedRE   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) -`)
	microChangelogUpdatedRE = regexp.MustCompile(`(?m)^\*\*Last Updated\*\*:\s*(\d{4}-\d{2}-\d{2})`)
)

func (r Runner) runCheckChangelog(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, true)
	if err != nil {
		return err
	}
	content, err := readCheckOperatorFile(resolveCheckOperatorPath(runtimeRoot, options.path))
	if err != nil {
		return err
	}
	return writeCheckOperatorResult(out, errOut, "changelog", validateMicroChangelog(content), options.jsonOutput)
}

func validateMicroChangelog(content string) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	section, ok := microChangelogSection(content)
	if !ok {
		result.Passed = false
		result.Blocked = true
		result.Errors = append(result.Errors, "No ## Changelog section found")
		return result
	}
	var dates []string
	for _, match := range microChangelogEntryRE.FindAllStringSubmatch(section, -1) {
		item := match[1]
		dated := microChangelogDatedRE.FindStringSubmatch(item)
		if dated == nil {
			result.Errors = append(result.Errors, "Entries with incorrect date format: - "+item)
			continue
		}
		dates = append(dates, dated[1])
	}
	if len(dates) > 1 {
		sorted := append([]string(nil), dates...)
		slices.Sort(sorted)
		slices.Reverse(sorted)
		if !slices.Equal(dates, sorted) {
			result.Errors = append(result.Errors, "Entries not in reverse chronological order")
		}
	}
	if updated := microChangelogUpdatedRE.FindStringSubmatch(content); updated != nil && len(dates) > 0 && updated[1] != dates[0] {
		result.Errors = append(result.Errors, "Last Updated ("+updated[1]+") doesn't match newest changelog entry ("+dates[0]+")")
	}
	if len(result.Errors) > 0 {
		result.Passed = false
		result.Blocked = true
	}
	return result
}

func microChangelogSection(content string) (string, bool) {
	start := microChangelogHeadingRE.FindStringIndex(content)
	if start == nil {
		return "", false
	}
	body := content[start[0]:]
	if nl := strings.Index(body, "\n"); nl >= 0 {
		body = body[nl+1:]
	} else {
		body = ""
	}
	if next := microChangelogNextRE.FindStringIndex(body); next != nil {
		body = body[:next[0]]
	}
	return body, true
}
