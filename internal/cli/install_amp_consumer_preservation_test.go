package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerInstallAmpPreservesConsumerBootstrapAndInstructions(t *testing.T) {
	distribution, _ := setupInstallCommandFixture(t)
	writeInstallFile(t, filepath.Join(distribution, "dist", "amp", "skills", "foundations", "SKILL.md"), "# Foundations\n")
	plugin := "export default function loaf() {}\n"
	writeInstallFile(t, filepath.Join(distribution, "dist", "amp", ".amp", "plugins", "loaf.js"), plugin)
	writeTestTargetAdapterManifest(t, filepath.Join(distribution, "dist", "amp"), "amp", []map[string]string{{
		"id": ampHookPluginArtifactID, "kind": "plugin", "source_path": ".amp/plugins/loaf.js", "destination": "plugins/loaf.js", "sha256": sha256Hex(plugin),
	}})

	consumer := t.TempDir()
	if err := os.Mkdir(filepath.Join(consumer, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir(.git) error = %v", err)
	}
	setup := "#!/bin/sh\nexec .agents/loaf-orb-bootstrap.sh setup\n"
	resume := "#!/bin/sh\nexec .agents/loaf-orb-bootstrap.sh resume\n"
	pin := "schema=1\nversion=9.8.7-test.1\n"
	instructions := "# Consumer instructions\n\nKeep this introduction.\n\nKeep this epilogue.\n"
	writeInstallFile(t, filepath.Join(consumer, ".agents", "setup"), setup)
	writeInstallFile(t, filepath.Join(consumer, ".agents", "resume"), resume)
	writeInstallFile(t, filepath.Join(consumer, ".agents", "loaf-orb.pin"), pin)
	writeInstallFile(t, filepath.Join(consumer, "AGENTS.md"), instructions)

	t.Setenv(projectEnvironmentEnv, "1")
	for run := 1; run <= 2; run++ {
		var stdout bytes.Buffer
		err := (Runner{
			Stdout:     &stdout,
			WorkingDir: consumer,
			Executable: distributionFixtureExecutable(distribution),
		}).Run([]string{"install", "--to", "amp", "--yes"})
		if err != nil {
			t.Fatalf("install --to amp run %d error = %v\n%s", run, err, stdout.String())
		}
	}

	assertInstallFile(t, filepath.Join(consumer, ".agents", "setup"), setup)
	assertInstallFile(t, filepath.Join(consumer, ".agents", "resume"), resume)
	assertInstallFile(t, filepath.Join(consumer, ".agents", "loaf-orb.pin"), pin)
	assertInstallFile(t, filepath.Join(consumer, ".agents", "skills", "foundations", "SKILL.md"), "# Foundations\n")
	assertInstallFile(t, filepath.Join(consumer, ".amp", "plugins", "loaf.js"), plugin)

	agents := string(readFileBytes(t, filepath.Join(consumer, "AGENTS.md")))
	if !strings.Contains(agents, "Keep this introduction.") || !strings.Contains(agents, "Keep this epilogue.") {
		t.Fatalf("AGENTS.md = %q, want both consumer-owned regions preserved", agents)
	}
	if count := strings.Count(agents, "<!-- loaf:managed:start -->"); count != 1 {
		t.Fatalf("AGENTS.md managed fence count = %d, want 1 after repeated install\n%s", count, agents)
	}
}
