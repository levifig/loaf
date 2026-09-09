package cli

import (
	"path"
	"regexp"
	"strings"
)

var (
	infraKubectlDeleteNSRE      = regexp.MustCompile(`kubectl\s+delete\s+(namespace|ns)\b`)
	infraKubectlDeleteAllRE     = regexp.MustCompile(`kubectl\s+delete\s+.*--all\b`)
	infraKubectlDeleteRE        = regexp.MustCompile(`kubectl\s+delete\b`)
	infraTerraformDestroyRE     = regexp.MustCompile(`terraform\s+destroy\b`)
	infraTerraformAutoApproveRE = regexp.MustCompile(`terraform\s+destroy\b.*-auto-approve`)
	infraDockerPruneAllRE       = regexp.MustCompile(`docker\s+system\s+prune\s+(-a|--all)`)
	infraDockerForceRMRE        = regexp.MustCompile(`docker\s+(rm|rmi)\s+.*(-f|--force)`)
	infraHelmUninstallRE        = regexp.MustCompile(`helm\s+uninstall\b`)
	infraRMRFOperandRE          = regexp.MustCompile(`rm\s+(-rf|-fr)\s+(?:'([^']*)'|"([^"]*)"|([^\s'";|&()<>]+))`)
)

func infraCommandRemovesSystemRoot(command string) bool {
	for _, match := range infraRMRFOperandRE.FindAllStringSubmatch(command, -1) {
		operand := firstNonEmpty(match[2], match[3], match[4])
		// Compare complete POSIX operands, accepting equivalent root spellings
		// such as /etc/. without classifying /tmp/build as the root /tmp.
		switch path.Clean(operand) {
		case "/", "/usr", "/etc", "/var", "/home", "/tmp":
			return true
		}
	}
	return false
}

func runNativeValidateInfraSafety(context checkHookContext) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	command := safetyInspectableCommand(checkContextCommand(context))
	if strings.TrimSpace(command) == "" {
		return result
	}
	switch {
	case infraKubectlDeleteNSRE.MatchString(command):
		return blockedCheckResult("kubectl delete namespace detected. This will delete the entire namespace and all resources within it. If intentional, ask the user to confirm before proceeding.")
	case infraKubectlDeleteAllRE.MatchString(command):
		return blockedCheckResult("kubectl delete --all detected. This will delete all resources of that type. If intentional, ask the user to confirm before proceeding.")
	case infraTerraformAutoApproveRE.MatchString(command):
		return blockedCheckResult("terraform destroy -auto-approve detected. Auto-approved destroy is dangerous. Remove -auto-approve flag. If intentional, ask the user to confirm before proceeding.")
	case infraDockerPruneAllRE.MatchString(command):
		return blockedCheckResult("docker system prune -a detected. This removes all unused images, not just dangling ones. If intentional, ask the user to confirm before proceeding.")
	case infraCommandRemovesSystemRoot(command):
		return blockedCheckResult("Dangerous rm -rf on system path detected. This could damage the system.")
	}
	if infraKubectlDeleteRE.MatchString(command) {
		result.Warnings = append(result.Warnings, "kubectl delete detected. Verify the target resource.")
	}
	if infraTerraformDestroyRE.MatchString(command) && !infraTerraformAutoApproveRE.MatchString(command) {
		result.Warnings = append(result.Warnings, "terraform destroy detected. This will destroy infrastructure. Terraform will prompt for confirmation.")
	}
	if infraDockerForceRMRE.MatchString(command) {
		result.Warnings = append(result.Warnings, "Forced docker removal detected. Verify targets.")
	}
	if infraHelmUninstallRE.MatchString(command) {
		result.Warnings = append(result.Warnings, "helm uninstall detected. This will remove the release and its resources.")
	}
	return result
}
