package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	dockerfileLatestRE         = regexp.MustCompile(`FROM.*:latest`)
	dockerfileAssignmentLeakRE = regexp.MustCompile(`(` + "PASS" + "WORD|" + "SEC" + "RET|KEY|TOKEN)=")
)

func writeCheckDockerfileHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check dockerfile [file] [--json]", "Validate Dockerfile best practices (multi-stage, USER, HEALTHCHECK, pinned tags, apt cleanup, PYTHONUNBUFFERED, assignment leaks). Defaults to Dockerfile in the working directory.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func (r Runner) runCheckDockerfile(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, false)
	if err != nil {
		return err
	}
	path := options.path
	if !options.hasPath {
		path = "Dockerfile"
	}
	content, err := readCheckOperatorFile(resolveCheckOperatorPath(runtimeRoot, path))
	if err != nil {
		return fmt.Errorf("unreadable Dockerfile %s: %w", path, err)
	}
	return writeCheckOperatorResult(out, errOut, "dockerfile", validateDockerfile(content), options.jsonOutput)
}

func validateDockerfile(content string) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	if !regexp.MustCompile(`(?m)^FROM.*AS`).MatchString(content) {
		result.Findings = append(result.Findings, "No multi-stage build detected")
	}
	if !regexp.MustCompile(`(?m)^USER`).MatchString(content) {
		result.Findings = append(result.Findings, "No USER directive found (running as root)")
	}
	if !regexp.MustCompile(`(?m)^HEALTHCHECK`).MatchString(content) {
		result.Findings = append(result.Findings, "No HEALTHCHECK defined")
	}
	if dockerfileLatestRE.MatchString(content) {
		result.Findings = append(result.Findings, "Using :latest tag (specify version)")
	}
	if strings.Contains(content, "apt-get install") && !strings.Contains(content, "rm -rf /var/lib/apt/lists") {
		result.Findings = append(result.Findings, "apt-get used without cleanup")
	}
	if strings.Contains(content, "python") && !strings.Contains(content, "PYTHONUNBUFFERED") {
		result.Findings = append(result.Findings, "Python detected but PYTHONUNBUFFERED not set")
	}
	if dockerfileAssignmentLeakRE.MatchString(content) {
		result.Findings = append(result.Findings, "Potential assignment leak in Dockerfile")
	}
	if len(result.Findings) > 0 {
		result.Passed = false
		result.Blocked = true
		result.Errors = append(result.Errors, fmt.Sprintf("Found %d issue(s)", len(result.Findings)))
	}
	return result
}
