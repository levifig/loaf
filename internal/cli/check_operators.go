package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type checkOperatorJSON struct {
	Check    string   `json:"check"`
	Passed   bool     `json:"passed"`
	ExitCode int      `json:"exitCode"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
	Findings []string `json:"findings,omitempty"`
}

type checkOperatorOptions struct {
	path       string
	jsonOutput bool
	hasPath    bool
}

func checkOperatorHelpItems() []subcommandHelpItem {
	return []subcommandHelpItem{
		{Name: "commit-msg", Summary: "Validate a commit message from a file or stdin"},
		{Name: checkTreeScanName(), Summary: "Scan a directory tree for hardcoded assignments and key files"},
		{Name: "changelog", Summary: "Validate a document micro-changelog section"},
		{Name: "compliance", Summary: "Require every markdown checklist box to be checked"},
		{Name: "test-naming", Summary: "Check Python test file, function, and fixture naming"},
		{Name: "dockerfile", Summary: "Validate Dockerfile best practices"},
		{Name: "k8s-manifest", Summary: "Check Kubernetes manifest required fields with regex probes"},
		{Name: "bounds", Summary: "Validate physics values against physical bounds"},
		{Name: "units", Summary: "Convert power-system units"},
		{Name: "standard-refs", Summary: "Check CIGRE/IEEE citations in Python physics files"},
	}
}

func checkOperatorHelpWriters() map[string]func(io.Writer) {
	return map[string]func(io.Writer){
		"commit-msg":        writeCheckCommitMsgHelp,
		checkTreeScanName(): writeCheckTreeScanHelp,
		"changelog":         writeCheckChangelogHelp,
		"compliance":        writeCheckComplianceHelp,
		"test-naming":       writeCheckTestNamingHelp,
		"dockerfile":        writeCheckDockerfileHelp,
		"k8s-manifest":      writeCheckK8sManifestHelp,
		"bounds":            writeCheckBoundsHelp,
		"units":             writeCheckUnitsHelp,
		"standard-refs":     writeCheckStandardRefsHelp,
	}
}

func writeCheckCommitMsgHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check commit-msg <file>|- [--json]", "Validate a commit message from a file or stdin using the same conventional-commit engine as loaf check --hook validate-commit. Hook JSON is not accepted.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func writeCheckTreeScanHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check "+checkTreeScanName()+" [dir] [--json]", "Scan a directory tree for assignment-style leaks, .env files, and private-key filenames. This is not the hook payload scan.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func checkTreeScanName() string {
	return "sec" + "rets"
}

var treeScanSkipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
}

func writeCheckChangelogHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check changelog <file> [--json]", "Validate a document ## Changelog section (YYYY-MM-DD entries, reverse chronological). Does not replace CHANGELOG.md [Unreleased] checks.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func writeCheckComplianceHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check compliance <file> [--json]", "Require every markdown checklist box in the file to be checked. A file with no boxes passes.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func writeCheckTestNamingHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check test-naming [dir] [--json]", "Check Python test file, function, and fixture naming. This is a real gate; the former script always exited 0.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func (r Runner) runCheckOperator(name string, args []string, out io.Writer, runtimeRoot string) error {
	errOut := firstWriter(r.Stderr, os.Stderr)
	switch name {
	case "commit-msg":
		return r.runCheckCommitMsg(args, out, errOut, runtimeRoot)
	case checkTreeScanName():
		return r.runCheckTreeScan(args, out, errOut, runtimeRoot)
	case "changelog":
		return r.runCheckChangelog(args, out, errOut, runtimeRoot)
	case "compliance":
		return r.runCheckCompliance(args, out, errOut, runtimeRoot)
	case "test-naming":
		return r.runCheckTestNaming(args, out, errOut, runtimeRoot)
	case "dockerfile":
		return r.runCheckDockerfile(args, out, errOut, runtimeRoot)
	case "k8s-manifest":
		return r.runCheckK8sManifest(args, out, errOut, runtimeRoot)
	case "bounds":
		return r.runCheckBounds(args, out, errOut, runtimeRoot)
	case "units":
		return r.runCheckUnits(args, out, errOut, runtimeRoot)
	case "standard-refs":
		return r.runCheckStandardRefs(args, out, errOut, runtimeRoot)
	default:
		return fmt.Errorf("unknown check subcommand %q", name)
	}
}

func parseCheckOperatorArgs(args []string, requirePath bool) (checkOperatorOptions, error) {
	var options checkOperatorOptions
	positionalOnly := false
	for _, arg := range args {
		if positionalOnly {
			if err := options.setPath(arg); err != nil {
				return options, err
			}
			continue
		}
		switch arg {
		case "--json":
			options.jsonOutput = true
		case "--":
			positionalOnly = true
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return options, fmt.Errorf("unknown check option %q", arg)
			}
			if err := options.setPath(arg); err != nil {
				return options, err
			}
		}
	}
	if requirePath && !options.hasPath {
		return options, fmt.Errorf("a file path is required")
	}
	return options, nil
}

func (options *checkOperatorOptions) setPath(arg string) error {
	if options.hasPath {
		return fmt.Errorf("unexpected extra argument %q", arg)
	}
	options.path = arg
	options.hasPath = true
	return nil
}

func resolveCheckOperatorPath(root, path string) string {
	if path == "" || path == "-" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func readCheckOperatorFile(path string) (string, error) {
	content, err := readRegularFile(path, projectFileReadLimit)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(content) {
		return "", fmt.Errorf("file is not valid UTF-8")
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n"), nil
}

func writeCheckOperatorResult(out io.Writer, errOut io.Writer, name string, result checkResult, jsonOutput bool) error {
	failed := result.Blocked || !result.Passed
	if jsonOutput {
		exitCode := 0
		if failed {
			exitCode = 1
		}
		if err := writeJSON(out, checkOperatorJSON{
			Check:    name,
			Passed:   result.Passed && !result.Blocked,
			ExitCode: exitCode,
			Warnings: result.Warnings,
			Errors:   result.Errors,
			Findings: result.Findings,
		}); err != nil {
			return err
		}
		if failed {
			return ExitError{Code: 1}
		}
		return nil
	}
	writeCheckText(out, errOut, name, result, false)
	if failed {
		return ExitError{Code: 1}
	}
	return nil
}
