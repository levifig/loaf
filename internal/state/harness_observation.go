package state

import (
	"os"
	"strings"
)

// ObservedHarnessFromEnvironment uses running-harness markers only. Conflicting
// inherited markers cannot establish which harness is active; do not guess.
func ObservedHarnessFromEnvironment() (string, string) {
	codex := strings.TrimSpace(os.Getenv("CODEX_THREAD_ID")) != ""
	claude := strings.TrimSpace(os.Getenv("CLAUDECODE")) != ""
	if codex && claude {
		return "unknown", "unknown"
	}
	if codex {
		return "codex", knownHarnessVersion(os.Getenv("CODEX_VERSION"))
	}
	if claude {
		return "claude-code", knownHarnessVersion(os.Getenv("CLAUDE_CODE_VERSION"))
	}
	return "unknown", "unknown"
}

func knownHarnessVersion(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 0 && len(value) <= 64 {
		for _, char := range value {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("._+-", char) {
				continue
			}
			return "unknown"
		}
		return value
	}
	return "unknown"
}
