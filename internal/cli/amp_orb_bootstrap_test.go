package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const ampOrbBootstrapTestVersion = "0.6.0-rc.1"

type ampOrbBootstrapFixture struct {
	project       string
	orbHome       string
	userBin       string
	toolsDir      string
	archive       string
	archiveDigest string
	curlLog       string
	tarLog        string
	loafLog       string
	readyMarker   string
	env           []string
}

func TestAmpOrbConsumerBootstrapSetupResumeAndIdempotence(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	lifecycleBefore := map[string]string{}
	for _, name := range []string{"setup", "resume", "loaf-orb-bootstrap.sh"} {
		lifecycleBefore[name] = ampOrbBootstrapReadFile(t, filepath.Join(fixture.project, ".agents", name))
	}

	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("setup error = %v\n%s", err, output)
	}
	releaseDir := filepath.Join(fixture.orbHome, "releases", ampOrbBootstrapTestVersion)
	loafBinary := filepath.Join(releaseDir, "bin", "loaf")
	for _, path := range []string{
		loafBinary,
		filepath.Join(releaseDir, loafReleaseManifestFile),
		filepath.Join(releaseDir, loafArchiveDigestFile),
	} {
		if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published release file %s: info=%v err=%v", path, info, err)
		}
	}
	receipt := ampOrbBootstrapReadFile(t, filepath.Join(releaseDir, loafArchiveDigestFile))
	if receipt != fixture.archiveDigest+"\n" {
		t.Fatalf("archive receipt = %q, want exact verified digest", receipt)
	}
	resolvedLink, err := filepath.EvalSymlinks(filepath.Join(fixture.userBin, "loaf"))
	if err != nil {
		t.Fatalf("resolve user-bin loaf: %v", err)
	}
	resolvedBinary, err := filepath.EvalSymlinks(loafBinary)
	if err != nil {
		t.Fatalf("resolve pinned loaf: %v", err)
	}
	if resolvedLink != resolvedBinary {
		t.Fatalf("user-bin loaf = %s, want %s", resolvedLink, resolvedBinary)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 1)
	assertAmpOrbBootstrapLineCount(t, fixture.tarLog, 1)
	assertAmpOrbBootstrapLineCount(t, fixture.loafLog, 3)
	if curlCall := ampOrbBootstrapReadFile(t, fixture.curlLog); !strings.Contains(curlCall, "--connect-timeout 10") || !strings.Contains(curlCall, "--max-time 120") {
		t.Fatalf("curl call lacks bounded timeouts: %s", curlCall)
	}
	if tarCall := ampOrbBootstrapReadFile(t, fixture.tarLog); !strings.Contains(tarCall, "-xpzf") {
		t.Fatalf("tar call does not preserve archive modes: %s", tarCall)
	}

	agentsBefore := ampOrbBootstrapReadFile(t, filepath.Join(fixture.project, "AGENTS.md"))
	if !strings.Contains(agentsBefore, "consumer-owned introduction") || !strings.Contains(agentsBefore, "consumer-owned epilogue") {
		t.Fatalf("setup did not preserve consumer instructions:\n%s", agentsBefore)
	}
	if count := strings.Count(agentsBefore, "<!-- loaf:managed:start -->"); count != 1 {
		t.Fatalf("managed fence count after setup = %d, want 1\n%s", count, agentsBefore)
	}

	if output, err := fixture.run(t, "resume"); err != nil {
		t.Fatalf("ready resume error = %v\n%s", err, output)
	}
	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("idempotent setup error = %v\n%s", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 1)
	assertAmpOrbBootstrapLineCount(t, fixture.tarLog, 1)
	assertAmpOrbBootstrapLineCount(t, fixture.loafLog, 7)
	agentsAfter := ampOrbBootstrapReadFile(t, filepath.Join(fixture.project, "AGENTS.md"))
	if agentsAfter != agentsBefore {
		t.Fatalf("ready resume/setup changed consumer instructions:\nbefore:\n%s\nafter:\n%s", agentsBefore, agentsAfter)
	}
	for name, want := range lifecycleBefore {
		if got := ampOrbBootstrapReadFile(t, filepath.Join(fixture.project, ".agents", name)); got != want {
			t.Fatalf("lifecycle script %s changed during setup/resume", name)
		}
	}
}

func TestAmpOrbConsumerBootstrapAgentCheckReadyIsReadOnly(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("setup error = %v\n%s", err, output)
	}
	projectBefore := ampReadinessTreeSnapshot(t, fixture.project)
	orbBefore := ampReadinessTreeSnapshot(t, fixture.orbHome)
	curlBefore := ampOrbBootstrapLineCount(t, fixture.curlLog)
	loafBefore := ampOrbBootstrapLineCount(t, fixture.loafLog)

	if output, err := fixture.run(t, "agent-check"); err != nil {
		t.Fatalf("agent-check error = %v\n%s", err, output)
	}
	if got := ampReadinessTreeSnapshot(t, fixture.project); got != projectBefore {
		t.Fatal("agent-check mutated the consumer project")
	}
	if got := ampReadinessTreeSnapshot(t, fixture.orbHome); got != orbBefore {
		t.Fatal("agent-check mutated the Orb release prefix")
	}
	if got := ampOrbBootstrapLineCount(t, fixture.curlLog); got != curlBefore {
		t.Fatalf("agent-check curl count = %d, want %d", got, curlBefore)
	}
	if got := ampOrbBootstrapLineCount(t, fixture.loafLog); got != loafBefore+1 {
		t.Fatalf("agent-check loaf call count = %d, want %d", got, loafBefore+1)
	}
}

func TestAmpOrbConsumerBootstrapAgentCheckRejectsInheritedEnvironmentWithoutMutation(t *testing.T) {
	tests := []struct {
		name      string
		overrides []string
	}{
		{name: "missing project environment", overrides: []string{"LOAF_PROJECT_ENV=0"}},
		{name: "wrong PATH", overrides: []string{"PATH_WITHOUT_USER_BIN=1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAmpOrbBootstrapFixture(t, false)
			if output, err := fixture.run(t, "setup"); err != nil {
				t.Fatalf("setup error = %v\n%s", err, output)
			}
			projectBefore := ampReadinessTreeSnapshot(t, fixture.project)
			orbBefore := ampReadinessTreeSnapshot(t, fixture.orbHome)
			curlBefore := ampOrbBootstrapLineCount(t, fixture.curlLog)
			loafBefore := ampOrbBootstrapLineCount(t, fixture.loafLog)
			env := fixture.env
			if test.overrides[0] == "PATH_WITHOUT_USER_BIN=1" {
				env = ampOrbBootstrapEnvWith(env, "PATH="+fixture.toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			} else {
				env = ampOrbBootstrapEnvWith(env, test.overrides...)
			}
			output, err := fixture.runWithEnv(t, "agent-check", env)
			if err == nil || !strings.Contains(output, "Amp project environment") {
				t.Fatalf("agent-check error = %v, output = %q", err, output)
			}
			if got := ampReadinessTreeSnapshot(t, fixture.project); got != projectBefore {
				t.Fatal("failed agent-check mutated the consumer project")
			}
			if got := ampReadinessTreeSnapshot(t, fixture.orbHome); got != orbBefore {
				t.Fatal("failed agent-check mutated the Orb release prefix")
			}
			if got := ampOrbBootstrapLineCount(t, fixture.curlLog); got != curlBefore {
				t.Fatalf("failed agent-check curl count = %d, want %d", got, curlBefore)
			}
			if got := ampOrbBootstrapLineCount(t, fixture.loafLog); got != loafBefore {
				t.Fatalf("failed agent-check loaf call count = %d, want %d", got, loafBefore)
			}
		})
	}
}

func TestAmpOrbConsumerBootstrapAgentCheckRejectsMissingRuntime(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	projectBefore := ampReadinessTreeSnapshot(t, fixture.project)

	output, err := fixture.run(t, "agent-check")
	if err == nil || !strings.Contains(output, "pinned Loaf release") {
		t.Fatalf("agent-check missing-runtime error = %v, output = %q", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 0)
	if got := ampReadinessTreeSnapshot(t, fixture.project); got != projectBefore {
		t.Fatal("missing-runtime agent-check mutated the consumer project")
	}
	if _, err := os.Lstat(fixture.orbHome); !os.IsNotExist(err) {
		t.Fatalf("missing-runtime agent-check created Orb state: %v", err)
	}
}

func TestAmpOrbConsumerBootstrapAgentCheckHidesReadinessDetails(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("setup error = %v\n%s", err, output)
	}
	env := ampOrbBootstrapEnvWith(fixture.env, "AMP_ORB_FORCE_READINESS_FAILURE=1")
	output, err := fixture.runWithEnv(t, "agent-check", env)
	if err == nil || !strings.Contains(output, "pinned Loaf readiness check failed") {
		t.Fatalf("agent-check readiness error = %v, output = %q", err, output)
	}
	if strings.Contains(output, "private-machine-path") {
		t.Fatalf("agent-check exposed captured readiness details: %q", output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 1)
}

func TestAmpOrbConsumerBootstrapHealthyInternalsReportWrongEnvironmentWithoutDownload(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("setup error = %v\n%s", err, output)
	}
	wrongEnv := ampOrbBootstrapEnvWith(fixture.env, "LOAF_PROJECT_ENV=0")
	for _, entry := range []string{"setup", "resume"} {
		curlBefore := ampOrbBootstrapLineCount(t, fixture.curlLog)
		output, err := fixture.runWithEnv(t, entry, wrongEnv)
		if err == nil || !strings.Contains(output, "Amp project environment") {
			t.Fatalf("%s wrong-environment error = %v, output = %q", entry, err, output)
		}
		if got := ampOrbBootstrapLineCount(t, fixture.curlLog); got != curlBefore {
			t.Fatalf("%s downloaded with healthy internals: got %d curl calls, want %d", entry, got, curlBefore)
		}
	}
}

func TestAmpOrbConsumerBootstrapRepairThenChecksInheritedEnvironment(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	wrongEnv := ampOrbBootstrapEnvWith(fixture.env, "LOAF_PROJECT_ENV=0")

	output, err := fixture.runWithEnv(t, "setup", wrongEnv)
	if err == nil || !strings.Contains(output, "Amp project environment") {
		t.Fatalf("repair setup error = %v, output = %q", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 1)
	if _, err := os.Stat(filepath.Join(fixture.orbHome, "releases", ampOrbBootstrapTestVersion, "bin", "loaf")); err != nil {
		t.Fatalf("repair did not publish pinned runtime: %v", err)
	}
	if output, err := fixture.run(t, "agent-check"); err != nil {
		t.Fatalf("agent-check after repair error = %v\n%s", err, output)
	}
}

func TestAmpOrbConsumerBootstrapRecoversOrphanLock(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	lockDir := filepath.Join(fixture.orbHome, ".bootstrap.lock")
	ampOrbBootstrapWriteFile(t, filepath.Join(lockDir, "owner.pid"), []byte("99999999\n"), 0o644)

	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("setup with orphan lock error = %v\n%s", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 1)
	if _, err := os.Lstat(lockDir); !os.IsNotExist(err) {
		t.Fatalf("orphan lock was not cleaned after setup: %v", err)
	}
}

func TestAmpOrbConsumerBootstrapRefusesLiveLock(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	lockDir := filepath.Join(fixture.orbHome, ".bootstrap.lock")
	owner := fmt.Sprintf("%d\n", os.Getpid())
	ampOrbBootstrapWriteFile(t, filepath.Join(lockDir, "owner.pid"), []byte(owner), 0o644)

	output, err := fixture.run(t, "setup")
	if err == nil || !strings.Contains(output, "another setup or resume process") {
		t.Fatalf("setup with live lock error = %v, output = %q", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 0)
	if got := ampOrbBootstrapReadFile(t, filepath.Join(lockDir, "owner.pid")); got != owner {
		t.Fatalf("live lock owner changed: got %q want %q", got, owner)
	}
}

func TestAmpOrbConsumerBootstrapReplacesOwnedDanglingUserBinLink(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	dangling := filepath.Join(fixture.orbHome, "releases", "0.9.0", "bin", "loaf")
	if err := os.Symlink(dangling, filepath.Join(fixture.userBin, "loaf")); err != nil {
		t.Fatal(err)
	}

	if output, err := fixture.run(t, "setup"); err != nil {
		t.Fatalf("setup with owned dangling link error = %v\n%s", err, output)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(fixture.userBin, "loaf"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(fixture.orbHome, "releases", ampOrbBootstrapTestVersion, "bin", "loaf")
	want, err = filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != want {
		t.Fatalf("activated loaf = %s, want %s", resolved, want)
	}
}

func TestAmpOrbConsumerBootstrapRejectsUnsafePinBeforeDownload(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "semver leading zero", old: "version=" + ampOrbBootstrapTestVersion, new: "version=01.2.3"},
		{name: "sensitive project identity", old: "project.id=project-test", new: "project.id=project-token"},
		{name: "traversing remote", old: "git.remote=acme/consumer", new: "git.remote=acme/../consumer"},
		{name: "URL tracker scope", old: "tracker.scope=acme/consumer#Loaf", new: "tracker.scope=https://example.invalid/scope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAmpOrbBootstrapFixture(t, false)
			pinPath := filepath.Join(fixture.project, ".agents", "loaf-orb.pin")
			body := ampOrbBootstrapReadFile(t, pinPath)
			body = strings.Replace(body, test.old, test.new, 1)
			ampOrbBootstrapWriteFile(t, pinPath, []byte(body), 0o644)
			ampOrbBootstrapGit(t, fixture.project, "add", ".agents/loaf-orb.pin")
			ampOrbBootstrapGit(t, fixture.project, "commit", "-m", "Mutate bootstrap pin")

			if output, err := fixture.run(t, "setup"); err == nil {
				t.Fatalf("unsafe pin succeeded; output=%q", output)
			}
			assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 0)
			assertAmpOrbBootstrapLineCount(t, fixture.tarLog, 0)
		})
	}
}

func TestAmpOrbConsumerBootstrapRejectsUncommittedPinBeforeDownload(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, false)
	pinPath := filepath.Join(fixture.project, ".agents", "loaf-orb.pin")
	body := ampOrbBootstrapReadFile(t, pinPath)
	body = strings.Replace(body, "tracker.scope=acme/consumer#Loaf", "tracker.scope=acme/consumer#Other", 1)
	ampOrbBootstrapWriteFile(t, pinPath, []byte(body), 0o644)

	output, err := fixture.run(t, "setup")
	if err == nil || !strings.Contains(output, "must match committed bytes") {
		t.Fatalf("setup with uncommitted pin error = %v, output = %q", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 0)
	assertAmpOrbBootstrapLineCount(t, fixture.tarLog, 0)
}

func TestAmpOrbConsumerBootstrapChecksumMismatchFailsBeforeExtraction(t *testing.T) {
	fixture := newAmpOrbBootstrapFixture(t, true)

	output, err := fixture.run(t, "setup")
	if err == nil || !strings.Contains(output, "archive checksum does not match the project pin") {
		t.Fatalf("setup error = %v, output = %q; want checksum refusal", err, output)
	}
	assertAmpOrbBootstrapLineCount(t, fixture.curlLog, 1)
	assertAmpOrbBootstrapLineCount(t, fixture.tarLog, 0)
	assertAmpOrbBootstrapLineCount(t, fixture.loafLog, 0)
	if _, err := os.Lstat(filepath.Join(fixture.orbHome, "releases", ampOrbBootstrapTestVersion)); !os.IsNotExist(err) {
		t.Fatalf("checksum mismatch published a release: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(fixture.userBin, "loaf")); !os.IsNotExist(err) {
		t.Fatalf("checksum mismatch activated a user-bin loaf: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(fixture.orbHome, ".bootstrap.lock")); !os.IsNotExist(err) {
		t.Fatalf("checksum mismatch left bootstrap lock: %v", err)
	}
	wantAgents := "# Consumer instructions\n\nconsumer-owned introduction\n\nconsumer-owned epilogue\n"
	if got := ampOrbBootstrapReadFile(t, filepath.Join(fixture.project, "AGENTS.md")); got != wantAgents {
		t.Fatalf("checksum mismatch changed AGENTS.md:\n%s", got)
	}
}

func TestAmpOrbConsumerBootstrapGuideUsesOfficialDogfoodAndLiteralEnvironment(t *testing.T) {
	guide := ampOrbBootstrapReadFile(t, filepath.Join(ampOrbBootstrapRepoRoot(t), "docs", "cloud", "amp-orb-consumer-bootstrap.md"))
	for _, want := range []string{
		"version=0.6.0-rc.1",
		"official `v0.6.0-rc.1` GitHub release",
		"LOAF_PROJECT_ENV=1",
		"PATH=/home/user/.local/bin:<existing-path>",
		"leave the project `PATH` setting unset",
		"Omit `LOAF_BIN_DIR` for this default",
		".agents/loaf-orb-bootstrap.sh agent-check",
		"git status --porcelain",
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("guide is missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"PATH=/home/user/.local/bin:/usr/local/bin:/usr/bin:/bin",
		"PATH=$HOME",
		"PATH=${HOME",
		"PATH=${LOAF_BIN_DIR",
	} {
		if strings.Contains(guide, unwanted) {
			t.Errorf("guide relies on shell expansion in Amp PATH setting: %q", unwanted)
		}
	}
}

func newAmpOrbBootstrapFixture(t *testing.T, mismatch bool) ampOrbBootstrapFixture {
	t.Helper()
	for _, tool := range []string{"git", "sh", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required for Amp Orb bootstrap tests: %v", tool, err)
		}
	}
	realTar, err := exec.LookPath("tar")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project := filepath.Join(root, "consumer")
	agentsDir := filepath.Join(project, ".agents")
	toolsDir := filepath.Join(root, "tools")
	orbHome := filepath.Join(root, "orb")
	userBin := filepath.Join(root, "user-bin")
	home := filepath.Join(root, "home")
	for _, path := range []string{agentsDir, toolsDir, userBin, home} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	repoRoot := ampOrbBootstrapRepoRoot(t)
	templateDir := filepath.Join(repoRoot, "content", "templates", "amp-orb", ".agents")
	for _, name := range []string{"setup", "resume", "loaf-orb-bootstrap.sh"} {
		body, err := os.ReadFile(filepath.Join(templateDir, name))
		if err != nil {
			t.Fatalf("read %s template: %v", name, err)
		}
		ampOrbBootstrapWriteFile(t, filepath.Join(agentsDir, name), body, 0o755)
	}

	loafStub := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "${AMP_ORB_LOAF_LOG:?}"
case "${1:-}" in
  install)
    [ "${LOAF_PROJECT_ENV:-}" = 1 ] || exit 81
    [ "$*" = "install --to amp --yes" ] || exit 82
    if ! grep -q '<!-- loaf:managed:start -->' "${AMP_ORB_PROJECT:?}/AGENTS.md"; then
      printf '\n%s\n%s\n%s\n' '<!-- loaf:managed:start -->' 'managed fixture' '<!-- loaf:managed:end -->' >> "${AMP_ORB_PROJECT:?}/AGENTS.md"
    fi
    printf '%s\n' ready > "${AMP_ORB_READY:?}"
    ;;
  harness)
    if [ "${AMP_ORB_FORCE_READINESS_FAILURE:-}" = 1 ]; then
      printf '%s\n' 'private-machine-path' >&2
      exit 9
    fi
    [ -f "${AMP_ORB_READY:?}" ] || exit 2
    [ "$*" = "harness readiness --target amp --pin .agents/loaf-orb.pin --archive-sha256 ${AMP_ORB_ARCHIVE_SHA256:?}" ] || exit 83
    ;;
  *) exit 84 ;;
esac
`
	archive := filepath.Join(root, "loaf_"+ampOrbBootstrapTestVersion+"_linux-x64.tar.gz")
	ampOrbBootstrapWriteArchive(t, archive, loafStub)
	archiveBody, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(archiveBody))
	pinnedDigest := digest
	if mismatch {
		pinnedDigest = strings.Repeat("0", 64)
	}
	pin := fmt.Sprintf("schema=1\nversion=%s\narchive.linux-x64.sha256=%s\narchive.linux-arm64.sha256=%s\nproject.id=project-test\ngit.remote=acme/consumer\ntracker.provider=github\ntracker.scope=acme/consumer#Loaf\n", ampOrbBootstrapTestVersion, pinnedDigest, digest)
	ampOrbBootstrapWriteFile(t, filepath.Join(agentsDir, "loaf-orb.pin"), []byte(pin), 0o644)
	ampOrbBootstrapWriteFile(t, filepath.Join(project, "AGENTS.md"), []byte("# Consumer instructions\n\nconsumer-owned introduction\n\nconsumer-owned epilogue\n"), 0o644)

	curlLog := filepath.Join(root, "curl.log")
	tarLog := filepath.Join(root, "tar.log")
	loafLog := filepath.Join(root, "loaf.log")
	readyMarker := filepath.Join(root, "ready")
	ampOrbBootstrapWriteFile(t, filepath.Join(toolsDir, "uname"), []byte("#!/bin/sh\ncase \"${1:-}\" in -s) echo Linux ;; -m) echo x86_64 ;; *) exit 1 ;; esac\n"), 0o755)
	ampOrbBootstrapWriteFile(t, filepath.Join(toolsDir, "curl"), []byte(`#!/bin/sh
set -eu
printf '%s\n' "$*" >> "${AMP_ORB_CURL_LOG:?}"
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = --output ]; then
    shift
    output=$1
  fi
  shift
done
[ -n "$output" ] || exit 91
cp "${AMP_ORB_ARCHIVE:?}" "$output"
`), 0o755)
	ampOrbBootstrapWriteFile(t, filepath.Join(toolsDir, "shasum"), []byte(`#!/bin/sh
set -eu
last=
for argument in "$@"; do last=$argument; done
printf '%s  %s\n' "${AMP_ORB_ARCHIVE_SHA256:?}" "$last"
`), 0o755)
	ampOrbBootstrapWriteFile(t, filepath.Join(toolsDir, "tar"), []byte(`#!/bin/sh
set -eu
printf '%s\n' "$*" >> "${AMP_ORB_TAR_LOG:?}"
exec "${AMP_ORB_REAL_TAR:?}" "$@"
`), 0o755)

	ampOrbBootstrapGit(t, project, "init")
	ampOrbBootstrapGit(t, project, "config", "user.name", "Amp Orb Test")
	ampOrbBootstrapGit(t, project, "config", "user.email", "amp-orb@example.invalid")
	ampOrbBootstrapGit(t, project, "remote", "add", "origin", "https://github.com/acme/consumer.git")
	ampOrbBootstrapGit(t, project, "add", ".agents", "AGENTS.md")
	ampOrbBootstrapGit(t, project, "commit", "-m", "Add consumer bootstrap fixture")

	env := append([]string{}, os.Environ()...)
	env = append(env,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"LOAF_ORB_HOME="+orbHome,
		"LOAF_BIN_DIR="+userBin,
		"LOAF_PROJECT_ENV=1",
		"LOAF_ORB_ISOLATED_TESTING=1",
		"LOAF_ORB_TEST_RELEASE_BASE_URL=https://fixture.invalid/releases",
		"TMPDIR="+filepath.Join(root, "tmp"),
		"PATH="+toolsDir+string(os.PathListSeparator)+userBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"AMP_ORB_PROJECT="+project,
		"AMP_ORB_ARCHIVE="+archive,
		"AMP_ORB_ARCHIVE_SHA256="+digest,
		"AMP_ORB_CURL_LOG="+curlLog,
		"AMP_ORB_TAR_LOG="+tarLog,
		"AMP_ORB_LOAF_LOG="+loafLog,
		"AMP_ORB_READY="+readyMarker,
		"AMP_ORB_REAL_TAR="+realTar,
	)
	if err := os.MkdirAll(filepath.Join(root, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	return ampOrbBootstrapFixture{
		project: project, orbHome: orbHome, userBin: userBin, toolsDir: toolsDir, archive: archive, archiveDigest: digest,
		curlLog: curlLog, tarLog: tarLog, loafLog: loafLog, readyMarker: readyMarker, env: env,
	}
}

func (fixture ampOrbBootstrapFixture) run(t *testing.T, entry string) (string, error) {
	t.Helper()
	return fixture.runWithEnv(t, entry, fixture.env)
}

func (fixture ampOrbBootstrapFixture) runWithEnv(t *testing.T, entry string, env []string) (string, error) {
	t.Helper()
	path := filepath.Join(fixture.project, ".agents", entry)
	args := []string{"-c", `umask 077; exec sh "$1"`, "amp-orb-bootstrap-test", path}
	if entry == "agent-check" {
		path = filepath.Join(fixture.project, ".agents", "loaf-orb-bootstrap.sh")
		args = []string{"-c", `umask 077; exec sh "$1" agent-check`, "amp-orb-bootstrap-test", path}
	}
	cmd := exec.Command("sh", args...)
	cmd.Dir = fixture.project
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func ampOrbBootstrapWriteArchive(t *testing.T, path, loafStub string) {
	t.Helper()
	var body bytes.Buffer
	gz := gzip.NewWriter(&body)
	tw := tar.NewWriter(gz)
	root := "loaf_" + ampOrbBootstrapTestVersion + "_linux-x64"
	for _, file := range []struct {
		name string
		body string
		mode int64
	}{
		{name: root + "/bin/loaf", body: loafStub, mode: 0o755},
		{name: root + "/loaf-release-manifest.json", body: "{}\n", mode: 0o644},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: file.name, Mode: file.mode, Size: int64(len(file.body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	ampOrbBootstrapWriteFile(t, path, body.Bytes(), 0o644)
}

func ampOrbBootstrapRepoRoot(t *testing.T) string {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(workingDir, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func ampOrbBootstrapWriteFile(t *testing.T, path string, body []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, mode); err != nil {
		t.Fatal(err)
	}
}

func ampOrbBootstrapReadFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func assertAmpOrbBootstrapLineCount(t *testing.T, path string, want int) {
	t.Helper()
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) && want == 0 {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(body), "\n"); got != want {
		t.Fatalf("%s line count = %d, want %d; body=%q", path, got, want, body)
	}
}

func ampOrbBootstrapLineCount(t *testing.T, path string) int {
	t.Helper()
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(body), "\n")
}

func ampOrbBootstrapEnvWith(env []string, replacements ...string) []string {
	replacementByKey := map[string]string{}
	for _, replacement := range replacements {
		key, _, _ := strings.Cut(replacement, "=")
		replacementByKey[key] = replacement
	}
	result := make([]string, 0, len(env)+len(replacements))
	for _, value := range env {
		key, _, _ := strings.Cut(value, "=")
		if _, replaced := replacementByKey[key]; !replaced {
			result = append(result, value)
		}
	}
	for _, replacement := range replacements {
		result = append(result, replacement)
	}
	return result
}

func ampOrbBootstrapGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
