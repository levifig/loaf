package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	k8sRunAsNonRootRE     = regexp.MustCompile(`(?m)^\s*runAsNonRoot:\s*true\s*(?:#.*)?$`)
	k8sAllowPrivEscRE     = regexp.MustCompile(`(?m)^\s*allowPrivilegeEscalation:\s*false\s*(?:#.*)?$`)
	k8sImageRE            = regexp.MustCompile(`(?m)^\s*image:\s*([^\s#]+)`)
	k8sStringDataBlockRE  = regexp.MustCompile(`(?ms)^stringData:\s*\n((?:^[ \t]+[A-Za-z0-9_.-]+:\s*.*\n?)+)`)
	k8sStringDataKeyRE    = regexp.MustCompile(`(?m)^\s+([A-Za-z0-9_.-]+):`)
	k8sDocumentSplitRE    = regexp.MustCompile(`(?m)^---\s*$`)
	k8sSensitiveKeyTokens = []string{"PASS" + "WORD", "KEY", "TOKEN"}
)

func writeCheckK8sManifestHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check k8s-manifest <file> [--json]", "Check a Kubernetes manifest with the same regex field probes as the former helper. This is not a YAML parse and does not claim schema parity.", "--json       Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func (r Runner) runCheckK8sManifest(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, true)
	if err != nil {
		return err
	}
	content, err := readCheckOperatorFile(resolveCheckOperatorPath(runtimeRoot, options.path))
	if err != nil {
		return fmt.Errorf("unreadable manifest %s: %w", options.path, err)
	}
	return writeCheckOperatorResult(out, errOut, "k8s-manifest", validateK8sManifests(content), options.jsonOutput)
}

func validateK8sManifests(content string) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	for _, doc := range k8sManifestDocuments(content) {
		result.Findings = append(result.Findings, validateK8sManifest(doc)...)
	}
	if len(result.Findings) > 0 {
		result.Passed = false
		result.Blocked = true
		result.Errors = append(result.Errors, fmt.Sprintf("Found %d issue(s)", len(result.Findings)))
	}
	return result
}

func k8sManifestDocuments(text string) []string {
	var docs []string
	for _, doc := range k8sDocumentSplitRE.Split(text, -1) {
		doc = strings.TrimSpace(doc)
		if doc != "" {
			docs = append(docs, doc)
		}
	}
	return docs
}

func validateK8sManifest(doc string) []string {
	kind := k8sFieldValue(doc, "kind", "Unknown")
	name := k8sFieldValue(doc, "name", "unnamed")
	path := kind + "/" + name
	switch kind {
	case "Deployment", "StatefulSet":
		return validateK8sPodSpec(doc, path+".spec.template.spec")
	case "CronJob":
		return validateK8sPodSpec(doc, path+".spec.jobTemplate.spec.template.spec")
	case "Pod":
		return validateK8sPodSpec(doc, path+".spec")
	case "Sec" + "ret":
		var errors []string
		for _, key := range k8sOpaqueStringDataKeys(doc) {
			upper := strings.ToUpper(key)
			for _, token := range k8sSensitiveKeyTokens {
				if strings.Contains(upper, token) {
					errors = append(errors, path+": Sensitive data in stringData (use data with base64)")
					break
				}
			}
		}
		return errors
	default:
		return nil
	}
}

func validateK8sPodSpec(text, path string) []string {
	var errors []string
	if !k8sRunAsNonRootRE.MatchString(text) {
		errors = append(errors, path+": Missing runAsNonRoot: true")
	}
	if !strings.Contains(text, "containers:") {
		return errors
	}
	containerPath := path + ".containers"
	for _, required := range []string{"resources:", "requests:", "limits:", "livenessProbe:", "readinessProbe:"} {
		if !strings.Contains(text, required) {
			errors = append(errors, containerPath+": Missing "+strings.TrimSuffix(required, ":"))
		}
	}
	for _, image := range k8sImageRE.FindAllStringSubmatch(text, -1) {
		if strings.Contains(image[1], ":latest") || !strings.Contains(image[1], ":") {
			errors = append(errors, containerPath+": Using :latest or no tag for image")
		}
	}
	if !k8sAllowPrivEscRE.MatchString(text) {
		errors = append(errors, containerPath+": Missing allowPrivilegeEscalation: false")
	}
	return errors
}

func k8sFieldValue(text, field, defaultValue string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(field) + `:\s*([^\s#]+)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return defaultValue
	}
	return strings.Trim(strings.TrimSpace(match[1]), `"'`)
}

func k8sOpaqueStringDataKeys(text string) []string {
	match := k8sStringDataBlockRE.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	var keys []string
	for _, key := range k8sStringDataKeyRE.FindAllStringSubmatch(match[1], -1) {
		keys = append(keys, key[1])
	}
	return keys
}
