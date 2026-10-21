package state

import (
	"os"
	"strings"
)

// ObservedHarnessFromEnvironment uses running-harness markers only. Conflicting
// inherited markers cannot establish which harness is active; do not guess.
func ObservedHarnessFromEnvironment() (string, string) {
	// Runtime adapters identify their own hook subprocess, independently of
	// inherited parent-harness markers. This is provenance, not authentication.
	switch os.Getenv("LOAF_HOOK_TYPE") {
	case "pre-tool", "post-tool", "session", "harness":
		switch os.Getenv("LOAF_HOOK_HARNESS") {
		case "amp":
			return "amp", knownHarnessVersion(os.Getenv("AMP_VERSION"))
		case "opencode":
			return "opencode", knownHarnessVersion(os.Getenv("OPENCODE_VERSION"))
		}
	}
	harness, version := "unknown", "unknown"
	for _, marker := range []struct {
		active           bool
		name, versionKey string
	}{
		{strings.TrimSpace(os.Getenv("CODEX_THREAD_ID")) != "", "codex", "CODEX_VERSION"},
		{strings.TrimSpace(os.Getenv("CLAUDECODE")) != "", "claude-code", "CLAUDE_CODE_VERSION"},
		{os.Getenv("AGENT") == "amp", "amp", "AMP_VERSION"},
		{os.Getenv("OPENCODE") == "1", "opencode", "OPENCODE_VERSION"},
		{os.Getenv("CURSOR_AGENT") == "1", "cursor", "CURSOR_VERSION"},
	} {
		if !marker.active {
			continue
		}
		if harness != "unknown" {
			return "unknown", "unknown"
		}
		harness, version = marker.name, knownHarnessVersion(os.Getenv(marker.versionKey))
	}
	return harness, version
}

// harnessSessionVariables names the variable each harness documents for the
// conversation identifier it exports to its tool subprocesses. A harness with
// no documented variable has no entry and yields no session.
var harnessSessionVariables = map[string]string{
	"claude-code": "CLAUDE_CODE_SESSION_ID",
}

// ObservedHarnessSessionFromEnvironment returns the opaque session identifier
// exported by the observed harness. An unknown or ambiguous harness yields
// nothing, so a nested harness never borrows its parent's identifier. A value
// the journal origin cannot store is dropped rather than failing the write it
// would annotate.
func ObservedHarnessSessionFromEnvironment() string {
	harness, _ := ObservedHarnessFromEnvironment()
	key, ok := harnessSessionVariables[harness]
	if !ok {
		return ""
	}
	value := strings.TrimSpace(os.Getenv(key))
	if len(value) > journalOriginStringMaxLength {
		return ""
	}
	return value
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
