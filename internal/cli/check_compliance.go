package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

var complianceBoxRE = regexp.MustCompile(`^-\s+\[([ xX])\]\s+(.+)$`)

func (r Runner) runCheckCompliance(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, true)
	if err != nil {
		return err
	}
	content, err := readCheckOperatorFile(resolveCheckOperatorPath(runtimeRoot, options.path))
	if err != nil {
		return err
	}
	return writeCheckOperatorResult(out, errOut, "compliance", validateComplianceChecklist(content), options.jsonOutput)
}

func validateComplianceChecklist(content string) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	var incomplete []string
	total := 0
	for _, line := range strings.Split(content, "\n") {
		match := complianceBoxRE.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		total++
		if strings.ToLower(match[1]) != "x" {
			incomplete = append(incomplete, match[2])
		}
	}
	if total == 0 {
		return result
	}
	if len(incomplete) == 0 {
		return result
	}
	result.Passed = false
	result.Blocked = true
	result.Errors = append(result.Errors, fmt.Sprintf("Compliance: INCOMPLETE (%d items remaining)", len(incomplete)))
	for _, item := range incomplete {
		result.Findings = append(result.Findings, item)
	}
	return result
}
