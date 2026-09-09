package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	powerPhysicsKeywordRE = regexp.MustCompile(`(?i)convection|radiation|thermal|catenary|sag|tension|conductor|heat_balance|skin_effect`)
	powerStandardCiteRE   = regexp.MustCompile(`CIGRE|IEEE|IEC|EN|ANSI`)
	powerSectionRefRE     = regexp.MustCompile(`[Ss]ection|[Cc]hapter|[Cc]lause|§`)
)

func writeCheckStandardRefsHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check standard-refs [path] [--json]", "Require CIGRE/IEEE/IEC/EN/ANSI citations, with section numbers, in Python physics files. Unreadable paths fail.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func (r Runner) runCheckStandardRefs(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, false)
	if err != nil {
		return err
	}
	path := options.path
	if !options.hasPath {
		path = "."
	}
	result, err := validateStandardRefs(resolveCheckOperatorPath(runtimeRoot, path))
	if err != nil {
		return err
	}
	return writeCheckOperatorResult(out, errOut, "standard-refs", result, options.jsonOutput)
}

func validateStandardRefs(target string) (checkResult, error) {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	info, err := os.Stat(target)
	if err != nil {
		return result, fmt.Errorf("unreadable path %s: %w", filepath.ToSlash(target), err)
	}
	var files []string
	if info.IsDir() {
		walkErr := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return fmt.Errorf("unreadable path %s: %w", filepath.ToSlash(path), walkErr)
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".py") {
				files = append(files, path)
			}
			return nil
		})
		if walkErr != nil {
			return result, walkErr
		}
	} else if strings.HasSuffix(info.Name(), ".py") {
		files = append(files, target)
	}
	var issues []string
	for _, path := range files {
		content, err := readCheckOperatorFile(path)
		if err != nil {
			return result, fmt.Errorf("unreadable file %s: %w", filepath.ToSlash(path), err)
		}
		display := filepath.ToSlash(path)
		if powerPhysicsKeywordRE.MatchString(content) && !powerStandardCiteRE.MatchString(content) {
			issues = append(issues, display+": contains physics code but no CIGRE/IEEE/IEC references")
		}
		if powerStandardCiteRE.MatchString(content) && !powerSectionRefRE.MatchString(content) {
			issues = append(issues, display+": references standard but no section/chapter numbers")
		}
	}
	if len(issues) > 0 {
		result.Passed = false
		result.Blocked = true
		result.Errors = append(result.Errors, fmt.Sprintf("Found %d file(s) needing attention", len(issues)))
		result.Findings = append(result.Findings, issues...)
	}
	return result, nil
}
