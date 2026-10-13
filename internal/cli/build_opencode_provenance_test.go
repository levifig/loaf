package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeOpenCodePluginFailClosedSemantics(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node not found: %v", err)
	}
	root := testRepositoryRoot(t)
	hooks, err := readNativeBuildHooks(filepath.Join(root, "config", "hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	pluginPath := filepath.Join(t.TempDir(), "hooks.js")
	writeFile(t, pluginPath, renderNativeOpenCodePlugin(hooks, "test"))
	if err := copyNativeBuildDir(filepath.Join(root, "content", "hooks"), filepath.Join(filepath.Dir(pluginPath), "hooks"), nil, false); err != nil {
		t.Fatal(err)
	}
	probes := []nativeBuildHook{
		{id: "missing-dispatch", matcher: "MissingDispatch", section: "pre-tool", failClosed: true},
		{id: "spawn-error", matcher: "SpawnError", command: "true", section: "pre-tool", failClosed: true},
		{id: "signal", matcher: "Signal", command: "kill -TERM $$", section: "pre-tool", failClosed: true},
		{id: "timeout", matcher: "Timeout", command: "exec sleep 5", timeout: 50, section: "pre-tool", failClosed: true},
		// Shell builtins keep this timeout probe bounded without orphan children.
		{id: "timeout-zero-exit", matcher: "TimeoutZeroExit", command: "trap 'exit 0' TERM; SECONDS=0; while (( SECONDS < 3 )); do :; done", timeout: 50, section: "pre-tool", failClosed: true},
		{id: "unexpected", matcher: "Unexpected", command: "exit 3", section: "pre-tool", failClosed: true},
		{id: "advisory", matcher: "Advisory", command: "exit 3", section: "pre-tool"},
		{id: "advisory-reject", matcher: "AdvisoryReject", command: "printf 'policy rejection' >&2; exit 2", section: "pre-tool"},
	}
	probePath := filepath.Join(t.TempDir(), "hooks.js")
	writeFile(t, probePath, renderNativeOpenCodePlugin(probes, "test"))
	cmd := exec.Command(node, "--test", "internal/cli/opencode_provenance_plugin.test.mjs")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "LOAF_OPENCODE_PLUGIN_PATH="+pluginPath, "LOAF_OPENCODE_PROBE_PATH="+probePath, "LOAF_OPENCODE_NATIVE_BINARY="+buildCLIBinaryForTest(t))
	output, err := cmd.CombinedOutput()
	t.Logf("generated OpenCode public-plugin journey:\n%s", output)
	if err != nil {
		t.Fatalf("generated OpenCode plugin probe failed: %v", err)
	}
}

func TestNativeOpenCodeFailClosedPreToolHooksHaveExecutableDispatch(t *testing.T) {
	hooks, err := readNativeBuildHooks(filepath.Join(testRepositoryRoot(t), "config", "hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	plugin := renderNativeOpenCodePlugin(hooks, "test")
	path := filepath.Join(t.TempDir(), "hooks.js")
	writeFile(t, path, plugin)
	entries, err := readNativeBuildPluginPreToolHooks(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range entries {
		for _, entry := range group {
			if entry.FailClosed && entry.Command == "" && entry.Script == "" {
				t.Errorf("fail-closed OpenCode pre-tool hook %q has no executable dispatch", entry.ID)
			}
		}
	}
	if !strings.Contains(plugin, `"command": "loaf check --hook ephemeral-provenance"`) {
		t.Error("OpenCode provenance hook has no native check command")
	}
}
