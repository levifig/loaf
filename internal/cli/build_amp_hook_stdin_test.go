package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type nativeAmpHookStdinProbeReport struct {
	Unhandled []nativeAmpHookStdinUnhandled `json:"unhandled"`
	Cases     map[string]HookStdinCase      `json:"cases"`
}

type nativeAmpHookStdinUnhandled struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HookStdinCase struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

func TestNativeAmpRunHookHandlesClosedStdinWithoutCrashing(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node not found: %v", err)
	}

	root := realpath(t, t.TempDir())
	writeNativeAmpHookStdinProbe(t, root)

	cmd := exec.Command(node, "--experimental-strip-types", "probe.ts")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "LOAF_DB=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("closed-stdin probe crashed or failed: %v\n%s", err, output)
	}

	var report nativeAmpHookStdinProbeReport
	if unmarshalErr := json.Unmarshal(lastJSONObject(output), &report); unmarshalErr != nil {
		t.Fatalf("parse probe report: %v\n%s", unmarshalErr, output)
	}
	if len(report.Unhandled) != 0 {
		t.Fatalf("unhandled stream errors = %#v, want none", report.Unhandled)
	}

	consume := report.Cases["command-consume"]
	if consume.ExitCode != 0 || consume.Stdout != "payload-ok" || consume.Error != "" {
		t.Fatalf("command-consume = %#v, want exit 0 with echoed payload", consume)
	}

	earlyCommand := report.Cases["command-early-exit"]
	if earlyCommand.ExitCode != 0 || earlyCommand.Error != "" {
		t.Fatalf("command-early-exit = %#v, want child exit 0 without delivery error", earlyCommand)
	}

	earlyScript := report.Cases["script-early-exit"]
	if earlyScript.ExitCode != 0 || earlyScript.Error != "" {
		t.Fatalf("script-early-exit = %#v, want child exit 0 without delivery error", earlyScript)
	}

	reject := report.Cases["command-reject"]
	if reject.ExitCode != 2 || !strings.Contains(reject.Stderr, "blocked") {
		t.Fatalf("command-reject = %#v, want exit 2 with stderr blocked", reject)
	}

	rejectClosed := report.Cases["command-reject-closed-stdin"]
	if rejectClosed.ExitCode != 2 || !strings.Contains(rejectClosed.Stderr, "blocked") {
		t.Fatalf("command-reject-closed-stdin = %#v, want preserved rejection", rejectClosed)
	}

	rejectAfterClose := report.Cases["command-reject-after-close"]
	if rejectAfterClose.ExitCode != 2 || !strings.Contains(rejectAfterClose.Stderr, "delayed-block") {
		t.Fatalf("command-reject-after-close = %#v, want delayed rejection after stdin close", rejectAfterClose)
	}

	advisorySpawn := report.Cases["command-spawn-advisory"]
	if advisorySpawn.ExitCode != 1 || advisorySpawn.Error == "" {
		t.Fatalf("command-spawn-advisory = %#v, want advisory exit 1 with error", advisorySpawn)
	}

	closedSpawn := report.Cases["command-spawn-fail-closed"]
	if closedSpawn.ExitCode != 2 || closedSpawn.Error == "" {
		t.Fatalf("command-spawn-fail-closed = %#v, want fail-closed exit 2 with error", closedSpawn)
	}

	timeout := report.Cases["command-timeout"]
	if timeout.ExitCode != 1 {
		t.Fatalf("command-timeout = %#v, want signal-killed exit 1", timeout)
	}

	scriptConsume := report.Cases["script-consume"]
	if scriptConsume.ExitCode != 0 || scriptConsume.Stdout != "from-script" || scriptConsume.Error != "" {
		t.Fatalf("script-consume = %#v, want exit 0 stdout from-script", scriptConsume)
	}
}

func TestNativeAmpAndOpenCodeGeneratedPluginsEmbedClosedStdinHandling(t *testing.T) {
	root := testRepositoryRoot(t)
	for _, rel := range []string{
		filepath.Join("dist", "amp", ".amp", "plugins", "loaf.ts"),
		filepath.Join("dist", "opencode", "plugins", "hooks.ts"),
	} {
		body := readBuildFileString(t, filepath.Join(root, rel))
		for _, want := range []string{
			"isClosedPipeError",
			"observeHookChild",
			"EPIPE",
			"ERR_STREAM_DESTROYED",
			"child.stdin.on('error'",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing %q", rel, want)
			}
		}
	}
}

func TestNativeAmpAndOpenCodeGeneratedHookEntriesHandleClosedStdin(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node not found: %v", err)
	}

	repo := testRepositoryRoot(t)
	root := realpath(t, t.TempDir())
	writeNativeGeneratedHookStdinProbe(t, root)

	cmd := exec.Command(node, "--experimental-strip-types", "generated-probe.ts")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "LOAF_DB=", "LOAF_REPO_ROOT="+repo)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated hook entry probe crashed or failed: %v\n%s", err, output)
	}

	var report nativeAmpHookStdinProbeReport
	if unmarshalErr := json.Unmarshal(lastJSONObject(output), &report); unmarshalErr != nil {
		t.Fatalf("parse generated probe report: %v\n%s", unmarshalErr, output)
	}
	if len(report.Unhandled) != 0 {
		t.Fatalf("unhandled stream errors = %#v, want none", report.Unhandled)
	}
	ampAllow := report.Cases["amp-early-exit"]
	if ampAllow.ExitCode != 0 || ampAllow.Error != "" {
		t.Fatalf("amp-early-exit = %#v, want allow without error", ampAllow)
	}
	openCodeAllow := report.Cases["opencode-early-exit"]
	if openCodeAllow.ExitCode != 0 || openCodeAllow.Error != "" {
		t.Fatalf("opencode-early-exit = %#v, want allow without error", openCodeAllow)
	}
}

func lastJSONObject(output []byte) []byte {
	text := strings.TrimSpace(string(output))
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			return []byte(line)
		}
	}
	return output
}

func writeNativeAmpHookStdinProbe(t *testing.T, root string) {
	t.Helper()
	mkdirAll(t, filepath.Join(root, "hooks"))
	writeFile(t, filepath.Join(root, "hooks", "early-exit.sh"), "#!/bin/sh\nexit 0\n")
	writeFile(t, filepath.Join(root, "hooks", "consume.sh"), "#!/bin/sh\ncat\n")
	writeFile(t, filepath.Join(root, "probe.ts"), nativeAmpHookStdinProbeSource())
}

func writeNativeGeneratedHookStdinProbe(t *testing.T, root string) {
	t.Helper()
	mkdirAll(t, filepath.Join(root, "bin"))
	writeFile(t, filepath.Join(root, "bin", "loaf"), "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(root, "bin", "loaf"), 0o755); err != nil {
		t.Fatalf("chmod loaf stub: %v", err)
	}
	writeFile(t, filepath.Join(root, "generated-probe.ts"), nativeGeneratedHookStdinProbeSource())
}

func nativeAmpHookStdinProbeSource() string {
	return `import { execFile } from 'child_process';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

` + nativeAmpCoreFunctions() + `

const unhandled: { type: string; code?: string; message: string }[] = [];
process.on('uncaughtException', (error: any) => {
  unhandled.push({ type: 'uncaughtException', code: error?.code, message: error?.message || String(error) });
});
process.on('unhandledRejection', (error: any) => {
  unhandled.push({ type: 'unhandledRejection', code: error?.code, message: error?.message || String(error) });
});

const largePayload = 'x'.repeat(1024 * 1024);
const missingPath = join(__dirname, 'missing-bin');

async function main() {
  const cases: Record<string, HookResult> = {};
  cases['command-consume'] = await runHook('pre-tool', 'Bash', 'consume', 'cat', undefined, 'payload-ok', 5000, false, __dirname);
  cases['command-early-exit'] = await runHook('pre-tool', 'Bash', 'early', 'exit 0', undefined, largePayload, 5000, true, __dirname);
  cases['script-early-exit'] = await runHook('pre-tool', 'Bash', 'early-script', undefined, 'early-exit.sh', largePayload, 5000, true, __dirname);
  cases['command-reject'] = await runHook('pre-tool', 'Bash', 'reject', 'printf blocked >&2; exit 2', undefined, 'payload', 5000, true, __dirname);
  cases['command-reject-closed-stdin'] = await runHook('pre-tool', 'Bash', 'reject-closed', 'printf blocked >&2; exit 2', undefined, largePayload, 5000, true, __dirname);
  cases['command-reject-after-close'] = await runHook('pre-tool', 'Bash', 'reject-after-close', 'exec 0<&-; printf delayed-block >&2; exit 2', undefined, largePayload, 5000, true, __dirname);

  const previousPath = process.env.PATH;
  process.env.PATH = missingPath;
  cases['command-spawn-advisory'] = await runHook('pre-tool', 'Bash', 'spawn-advisory', 'true', undefined, 'payload', 5000, false, __dirname);
  cases['command-spawn-fail-closed'] = await runHook('pre-tool', 'Bash', 'spawn-closed', 'true', undefined, 'payload', 5000, true, __dirname);
  process.env.PATH = previousPath;

  cases['command-timeout'] = await runHook('pre-tool', 'Bash', 'timeout', 'sleep 10', undefined, 'payload', 200, false, __dirname);
  cases['script-consume'] = await runHook('pre-tool', 'Bash', 'script-consume', undefined, 'consume.sh', 'from-script', 5000, false, __dirname);

  await new Promise((resolve) => setTimeout(resolve, 100));
  process.stdout.write(JSON.stringify({ unhandled, cases }));
  process.exit(unhandled.length === 0 ? 0 : 1);
}

main().catch((error) => {
  unhandled.push({ type: 'main', message: error?.message || String(error) });
  process.stdout.write(JSON.stringify({ unhandled, cases: {} }));
  process.exit(1);
});
`
}

func nativeGeneratedHookStdinProbeSource() string {
	return `import { pathToFileURL } from 'url';
import { join } from 'path';

const repoRoot = process.env.LOAF_REPO_ROOT;
const previousPath = process.env.PATH;
process.env.PATH = join(process.cwd(), 'bin') + (previousPath ? ':' + previousPath : '');

const unhandled: { type: string; code?: string; message: string }[] = [];
process.on('uncaughtException', (error: any) => {
  unhandled.push({ type: 'uncaughtException', code: error?.code, message: error?.message || String(error) });
});
process.on('unhandledRejection', (error: any) => {
  unhandled.push({ type: 'unhandledRejection', code: error?.code, message: error?.message || String(error) });
});

const largeCommand = 'gh pr create --title probe --body ' + 'x'.repeat(1024 * 1024);

async function main() {
  const { default: initializeAmp } = await import(pathToFileURL(join(repoRoot, 'dist/amp/.amp/plugins/loaf.ts')).href);
  const { default: initializeOpenCode } = await import(pathToFileURL(join(repoRoot, 'dist/opencode/plugins/hooks.ts')).href);

  const ampHandlers = new Map();
  initializeAmp({
    registerTool() {},
    on(event: string, handler: unknown) { ampHandlers.set(event, handler); },
    helpers: {
      filePathFromURI: (uri: URL | string) => new URL(uri).pathname,
      shellCommandFromToolCall: (event: { input: { command: string; cwd?: string } }) => ({ command: event.input.command, dir: event.input.cwd }),
    },
    system: { workspaceRoot: pathToFileURL(process.cwd()) },
  });

  const ampResult = await ampHandlers.get('tool.call')({
    toolUseID: 'probe',
    tool: 'shell_command',
    input: { command: largeCommand, cwd: process.cwd() },
    thread: { id: 'T-parent' },
  });

  const openCode = await initializeOpenCode({
    client: { session: { get: async () => ({ data: {} }) } },
  });
  await openCode['tool.execute.before'](
    { tool: 'bash', sessionID: 'S-root', callID: 'c1' },
    { args: { command: largeCommand } },
  );

  await new Promise((resolve) => setTimeout(resolve, 100));
  process.stdout.write(JSON.stringify({
    unhandled,
    cases: {
      'amp-early-exit': { exitCode: ampResult?.action === 'allow' ? 0 : 2, stdout: '', stderr: ampResult?.message || '', error: ampResult?.action === 'allow' ? '' : ampResult?.message },
      'opencode-early-exit': { exitCode: 0, stdout: '', stderr: '' },
    },
  }));
  process.exit(unhandled.length === 0 ? 0 : 1);
}

main().catch((error) => {
  unhandled.push({ type: 'main', message: error?.message || String(error) });
  process.stdout.write(JSON.stringify({ unhandled, cases: {} }));
  process.exit(1);
});
`
}
