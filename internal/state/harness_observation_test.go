package state

import "testing"

func clearAdditionalHarnessMarkers(t *testing.T) {
	t.Helper()
	for _, key := range []string{"AGENT", "AMP_VERSION", "OPENCODE", "OPENCODE_VERSION", "LOAF_HOOK_HARNESS", "LOAF_HOOK_TYPE"} {
		t.Setenv(key, "")
	}
}

func TestObservedHarnessNativeMarkers(t *testing.T) {
	keys := []string{"CODEX_THREAD_ID", "CODEX_VERSION", "CLAUDECODE", "CLAUDE_CODE_VERSION", "AGENT", "AMP_VERSION", "OPENCODE", "OPENCODE_VERSION", "LOAF_HOOK_HARNESS", "LOAF_HOOK_TYPE"}
	for _, test := range []struct {
		name             string
		env              map[string]string
		harness, version string
	}{
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
