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

const ampOrbBootstrapTestVersion = "1.2.3-test.1"

type ampOrbBootstrapFixture struct {
	project       string
	orbHome       string
	userBin       string
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
	assertAmpOrbBootstrapLineCount(t, fixture.loafLog, 2)

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
	assertAmpOrbBootstrapLineCount(t, fixture.loafLog, 4)
	agentsAfter := ampOrbBootstrapReadFile(t, filepath.Join(fixture.project, "AGENTS.md"))
	if agentsAfter != agentsBefore {
		t.Fatalf("ready resume/setup changed consumer instructions:\nbefore:\n%s\nafter:\n%s", agentsBefore, agentsAfter)
	}
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
printf '%s\n' download >> "${AMP_ORB_CURL_LOG:?}"
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
printf '%s\n' extract >> "${AMP_ORB_TAR_LOG:?}"
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
		"LOAF_ORB_ISOLATED_TESTING=1",
		"LOAF_ORB_TEST_RELEASE_BASE_URL=https://fixture.invalid/releases",
		"TMPDIR="+filepath.Join(root, "tmp"),
		"PATH="+toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"),
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
		project: project, orbHome: orbHome, userBin: userBin, archive: archive, archiveDigest: digest,
		curlLog: curlLog, tarLog: tarLog, loafLog: loafLog, readyMarker: readyMarker, env: env,
	}
}

func (fixture ampOrbBootstrapFixture) run(t *testing.T, entry string) (string, error) {
	t.Helper()
	cmd := exec.Command("sh", filepath.Join(fixture.project, ".agents", entry))
	cmd.Dir = fixture.project
	cmd.Env = fixture.env
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

func ampOrbBootstrapGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
