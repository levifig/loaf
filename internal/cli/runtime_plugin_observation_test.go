package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeRuntimePluginAttribution(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	hooks := []nativeBuildHook{{id: "check-secrets", matcher: "Bash", command: "loaf check --hook check-secrets", timeout: 1000, failClosed: true, section: "pre-tool"}}
	for name, body := range map[string]string{"amp": renderNativeAmpPlugin(hooks, "test"), "opencode": renderNativeOpenCodePlugin(hooks, "test")} {
		writeFile(t, filepath.Join(dir, name+".mjs"), body)
	}
	cmd := exec.Command(node, "--test", "internal/cli/runtime_plugin_observation.test.mjs")
	cmd.Dir = testRepositoryRoot(t)
	cmd.Env = append(os.Environ(), "LOAF_TEST_PLUGIN_DIR="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("adapter provenance: %v\n%s", err, out)
	}
}
