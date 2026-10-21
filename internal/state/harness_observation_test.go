package state

import (
	"strings"
	"testing"
)

func clearAdditionalHarnessMarkers(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CURSOR_AGENT", "CURSOR_VERSION", "AGENT", "AMP_VERSION", "OPENCODE", "OPENCODE_VERSION", "LOAF_HOOK_HARNESS", "LOAF_HOOK_TYPE"} {
		t.Setenv(key, "")
	}
}

func TestObservedHarnessNativeMarkers(t *testing.T) {
	keys := []string{"CODEX_THREAD_ID", "CODEX_VERSION", "CLAUDECODE", "CLAUDE_CODE_VERSION", "CURSOR_AGENT", "CURSOR_VERSION", "AGENT", "AMP_VERSION", "OPENCODE", "OPENCODE_VERSION", "LOAF_HOOK_HARNESS", "LOAF_HOOK_TYPE"}
	for _, test := range []struct {
		name             string
		env              map[string]string
		harness, version string
	}{
		{"cursor shell", map[string]string{"CURSOR_AGENT": "1"}, "cursor", "unknown"},
		{"cursor version", map[string]string{"CURSOR_AGENT": "1", "CURSOR_VERSION": "2026.10.01-e373342"}, "cursor", "2026.10.01-e373342"},
		{"cursor version is not identity", map[string]string{"CURSOR_VERSION": "2026.10.01-e373342"}, "unknown", "unknown"},
		{"cursor config is not identity", map[string]string{"CURSOR_AGENT": "true"}, "unknown", "unknown"},
		{"cursor invalid version", map[string]string{"CURSOR_AGENT": "1", "CURSOR_VERSION": "secret / payload"}, "cursor", "unknown"},
		{"cursor nested in codex", map[string]string{"CURSOR_AGENT": "1", "CODEX_THREAD_ID": "parent"}, "unknown", "unknown"},
		{"cursor nested in claude", map[string]string{"CURSOR_AGENT": "1", "CLAUDECODE": "1"}, "unknown", "unknown"},
		{"cursor and amp", map[string]string{"CURSOR_AGENT": "1", "AGENT": "amp"}, "unknown", "unknown"},
		{"cursor and opencode", map[string]string{"CURSOR_AGENT": "1", "OPENCODE": "1"}, "unknown", "unknown"},
		{"amp adapter with inherited cursor", map[string]string{"LOAF_HOOK_TYPE": "pre-tool", "LOAF_HOOK_HARNESS": "amp", "CURSOR_AGENT": "1"}, "amp", "unknown"},
		{"amp shell", map[string]string{"AGENT": "amp"}, "amp", "unknown"},
		{"opencode shell", map[string]string{"OPENCODE": "1", "AGENT": "1"}, "opencode", "unknown"},
		{"amp version", map[string]string{"AGENT": "amp", "AMP_VERSION": "0.0.123-test"}, "amp", "0.0.123-test"},
		{"opencode version", map[string]string{"OPENCODE": "1", "OPENCODE_VERSION": "1.19.0"}, "opencode", "1.19.0"},
		{"version is not identity", map[string]string{"AMP_VERSION": "1.0", "OPENCODE_VERSION": "1.0"}, "unknown", "unknown"},
		{"generic agent is not identity", map[string]string{"AGENT": "1"}, "unknown", "unknown"},
		{"configuration is not identity", map[string]string{"OPENCODE": "false"}, "unknown", "unknown"},
		{"invalid version", map[string]string{"OPENCODE": "1", "OPENCODE_VERSION": "secret / payload"}, "opencode", "unknown"},
		{"amp nested in codex", map[string]string{"AGENT": "amp", "CODEX_THREAD_ID": "parent"}, "unknown", "unknown"},
		{"opencode nested in claude", map[string]string{"OPENCODE": "1", "CLAUDECODE": "1"}, "unknown", "unknown"},
		{"two native markers", map[string]string{"OPENCODE": "1", "AGENT": "amp"}, "unknown", "unknown"},
		{"amp adapter overrides inherited marker", map[string]string{"LOAF_HOOK_TYPE": "pre-tool", "LOAF_HOOK_HARNESS": "amp", "CODEX_THREAD_ID": "parent", "CODEX_VERSION": "9.9"}, "amp", "unknown"},
		{"opencode adapter overrides inherited marker", map[string]string{"LOAF_HOOK_TYPE": "post-tool", "LOAF_HOOK_HARNESS": "opencode", "AGENT": "amp", "AMP_VERSION": "9.9"}, "opencode", "unknown"},
		{"unscoped adapter marker ignored", map[string]string{"LOAF_HOOK_HARNESS": "amp"}, "unknown", "unknown"},
		{"unknown adapter ignored", map[string]string{"LOAF_HOOK_TYPE": "pre-tool", "LOAF_HOOK_HARNESS": "other"}, "unknown", "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, key := range keys {
				t.Setenv(key, "")
			}
			for key, value := range test.env {
				t.Setenv(key, value)
			}
			h, v := ObservedHarnessFromEnvironment()
			if h != test.harness || v != test.version {
				t.Fatalf("got %s/%s, want %s/%s", h, v, test.harness, test.version)
			}
		})
	}
}

func TestObservedHarnessSessionFromEnvironment(t *testing.T) {
	keys := []string{"CODEX_THREAD_ID", "CODEX_VERSION", "CLAUDECODE", "CLAUDE_CODE_VERSION", "CLAUDE_CODE_SESSION_ID"}
	for _, test := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"claude code session", map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_SESSION_ID": "fcb66a6e-7aa5-491f-b7e4-6d86b9e0c596"}, "fcb66a6e-7aa5-491f-b7e4-6d86b9e0c596"},
		{"value stays opaque", map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_SESSION_ID": " Session/42:α "}, "Session/42:α"},
		{"claude code without session", map[string]string{"CLAUDECODE": "1"}, ""},
		{"session variable is not identity", map[string]string{"CLAUDE_CODE_SESSION_ID": "orphan"}, ""},
		{"claude nested in codex", map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_SESSION_ID": "outer", "CODEX_THREAD_ID": "inner"}, ""},
		{"codex documents no session variable", map[string]string{"CODEX_THREAD_ID": "thread-1", "CLAUDE_CODE_SESSION_ID": "inherited"}, ""},
		{"amp adapter ignores inherited claude session", map[string]string{"LOAF_HOOK_TYPE": "pre-tool", "LOAF_HOOK_HARNESS": "amp", "CLAUDECODE": "1", "CLAUDE_CODE_SESSION_ID": "inherited"}, ""},
		{"oversized value dropped", map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_SESSION_ID": strings.Repeat("s", journalOriginStringMaxLength+1)}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			clearAdditionalHarnessMarkers(t)
			for _, key := range keys {
				t.Setenv(key, "")
			}
			for key, value := range test.env {
				t.Setenv(key, value)
			}
			if got := ObservedHarnessSessionFromEnvironment(); got != test.want {
				t.Fatalf("session = %q, want %q", got, test.want)
			}
		})
	}
}
