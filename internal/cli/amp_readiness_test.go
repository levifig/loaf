package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const ampReadinessTestVersion = "9.8.7-test.1"

type ampReadinessFixture struct {
	t       *testing.T
	project string
	home    string
	release string
	binary  string
	pin     string
	archive string
}

func newAmpReadinessFixture(t *testing.T) *ampReadinessFixture {
	t.Helper()
	previousSentinel := ampOrbIdentityRequestTokenPath
	previousCommand := ampOrbProjectIDToken
	ampOrbIdentityRequestTokenPath = filepath.Join(t.TempDir(), "absent-workload-identity-request-token")
	t.Cleanup(func() {
		ampOrbIdentityRequestTokenPath = previousSentinel
		ampOrbProjectIDToken = previousCommand
	})
	projectRoot := filepath.Join(t.TempDir(), "project")
	home := filepath.Join(t.TempDir(), "home")
	releaseRoot := filepath.Join(t.TempDir(), "release")
	for _, path := range []string{projectRoot, home, filepath.Join(releaseRoot, "bin"), filepath.Join(releaseRoot, "dist", "amp", "skills", "foundations"), filepath.Join(releaseRoot, "dist", "amp", ".amp", "plugins")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv(projectEnvironmentEnv, "1")

	binary := filepath.Join(releaseRoot, "bin", "loaf")
	ampReadinessWriteFile(t, filepath.Join(releaseRoot, "package.json"), []byte(`{"name":"loaf","version":"`+ampReadinessTestVersion+`"}`+"\n"), 0o644)
	ampReadinessWriteFile(t, binary, []byte("fixture loaf binary\n"), 0o755)
	ampReadinessWriteFile(t, filepath.Join(releaseRoot, "dist", "amp", "skills", "foundations", "SKILL.md"), []byte("---\nname: foundations\ndescription: Test.\n---\n"), 0o644)
	pluginPath := filepath.Join(releaseRoot, "dist", "amp", ".amp", "plugins", "loaf.ts")
	ampReadinessWriteFile(t, pluginPath, []byte("export default function loaf() {}\n"), 0o644)
	pluginDigest, err := sha256RegularFile(pluginPath)
	if err != nil {
		t.Fatal(err)
	}
	mode := uint32(0o644)
	desired := targetAdapterManifest{
		Version: 1, Target: "amp", PackageVersion: ampReadinessTestVersion,
		CapabilityContractVersion: TargetCapabilityEvidenceContractVersion,
		Adapters:                  []string{"amp-plugin-v1"},
		Artifacts: []targetAdapterArtifact{
			{ID: "managed-instructions", Kind: "instruction", Destination: "project-instructions", SHA256: strings.Repeat("a", 64)},
			{ID: "plugin:.amp/plugins/loaf.ts", Kind: "plugin", SourcePath: ".amp/plugins/loaf.ts", Destination: "plugins/loaf.ts", SHA256: pluginDigest, Mode: &mode},
		},
	}
	if err := writeTargetAdapterManifest(filepath.Join(releaseRoot, "dist", "amp", targetBuildManifestFile), desired); err != nil {
		t.Fatal(err)
	}

	ampDir := filepath.Join(projectRoot, ".amp")
	if err := installAmpTarget(targetInstallOptions{
		Target: "amp", DistDir: filepath.Join(releaseRoot, "dist", "amp"), ConfigDir: ampDir,
		Version: ampReadinessTestVersion, HomeDir: projectRoot, ProjectRoot: projectRoot,
	}); err != nil {
		t.Fatalf("install fixture Amp target: %v", err)
	}

	archive := strings.Repeat("b", 64)
	ampReadinessWriteFile(t, filepath.Join(releaseRoot, loafArchiveDigestFile), []byte(archive+"\n"), 0o644)
	manifest := ampReadinessReleaseManifest(t, releaseRoot)
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	ampReadinessWriteFile(t, filepath.Join(releaseRoot, loafReleaseManifestFile), append(body, '\n'), 0o644)

	ampReadinessGit(t, projectRoot, "init")
	ampReadinessGit(t, projectRoot, "config", "user.name", "Readiness Test")
	ampReadinessGit(t, projectRoot, "config", "user.email", "readiness@example.invalid")
	ampReadinessGit(t, projectRoot, "remote", "add", "origin", "https://github.com/acme/ready.git")
	ampReadinessWriteFile(t, filepath.Join(projectRoot, ".agents", "loaf.conf"), []byte("{\"conf_id\":\"conf_test\",\"project_id\":\"proj_test\"}\n"), 0o644)
	pin := filepath.Join(projectRoot, "amp.pin")
	pinBody := fmt.Sprintf("schema=1\nversion=%s\narchive.linux-x64.sha256=%s\narchive.linux-arm64.sha256=%s\nproject.id=proj_test\ngit.remote=acme/ready\ntracker.provider=github\ntracker.scope=acme/ready#Loaf\n", ampReadinessTestVersion, archive, archive)
	ampReadinessWriteFile(t, pin, []byte(pinBody), 0o644)
	ampReadinessGit(t, projectRoot, "add", "amp.pin", ".agents/loaf.conf")
	ampReadinessGit(t, projectRoot, "commit", "-m", "pin Amp release")

	pathBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(pathBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(binary, filepath.Join(pathBin, "loaf")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &ampReadinessFixture{t: t, project: projectRoot, home: home, release: releaseRoot, binary: binary, pin: pin, archive: archive}
}

func TestAmpReadinessSuccessIsReadOnly(t *testing.T) {
	fixture := newAmpReadinessFixture(t)
	if _, err := resolveAmpReadinessPinPath(fixture.project, fixture.pin); err != nil {
		t.Fatalf("fixture pin resolution: %v", err)
	}
	beforeProject := ampReadinessTreeSnapshot(t, fixture.project)
	beforeRelease := ampReadinessTreeSnapshot(t, fixture.release)
	record, err := fixture.run(fixture.archive)
	if err != nil {
		t.Fatalf("readiness error = %v, findings = %v", err, record.Findings)
	}
	if !record.Ready || record.ProjectID != "proj_test" || record.CanonicalRemote != "acme/ready" || record.AmpProjectScope != "acme/ready" || record.AmpWorkspaceScope != "" || record.AmpProjectID != "" || record.AmpWorkspaceID != "" || record.TrackerScope != "acme/ready#Loaf" || record.PackageVersion != ampReadinessTestVersion {
		t.Fatalf("record = %#v", record)
	}
	if len(record.Skills) != 1 || len(record.Plugins) != 1 || record.BinarySHA256 == "" || record.ReleaseManifestSHA256 == "" {
		t.Fatalf("proof fields incomplete: %#v", record)
	}
	if after := ampReadinessTreeSnapshot(t, fixture.project); after != beforeProject {
		t.Fatal("readiness mutated the project tree")
	}
	if after := ampReadinessTreeSnapshot(t, fixture.release); after != beforeRelease {
		t.Fatal("readiness mutated the release tree")
	}
}

func TestAmpReadinessDiscoversOrbProjectAndWorkspaceIdentity(t *testing.T) {
	fixture := newAmpReadinessFixture(t)
	ampOrbIdentityRequestTokenPath = filepath.Join(t.TempDir(), "workload-identity-request-token")
	ampReadinessWriteFile(t, ampOrbIdentityRequestTokenPath, []byte("request\n"), 0o600)
	called := 0
	ampOrbProjectIDToken = func(args ...string) ([]byte, error) {
		called++
		if got, want := strings.Join(args, " "), "orb id-token --audience urn:loaf:readiness --subject-scope project"; got != want {
			t.Fatalf("Amp identity command args = %q, want %q", got, want)
		}
		return ampReadinessJWT(t, map[string]any{
			"project_id": "amp-project-uuid-123", "workspace_id": "workspace_123",
			"email": "private@example.invalid", "thread_id": "thread-secret", "user_id": "user-secret",
		}), nil
	}
	record, err := fixture.run(fixture.archive)
	if err != nil || !record.Ready {
		t.Fatalf("readiness error = %v, findings = %v", err, record.Findings)
	}
	if called != 1 || record.AmpProjectID != "amp-project-uuid-123" || record.AmpWorkspaceID != "workspace_123" {
		t.Fatalf("Orb identity record = %#v, calls = %d", record, called)
	}
	body, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, secret := range []string{"private@example.invalid", "thread-secret", "user-secret"} {
		if bytes.Contains(body, []byte(secret)) {
			t.Fatalf("readiness record leaked ignored claim %q", secret)
		}
	}
	if strings.Contains(strings.Join(record.Limitations, "\n"), "outside an Orb") {
		t.Fatalf("Orb record retained outside-Orb limitation: %v", record.Limitations)
	}
}

func TestAmpReadinessOrbIdentityFailuresAreClosedAndSecretSafe(t *testing.T) {
	tests := []struct {
		name  string
		token func(*testing.T) ([]byte, error)
		want  string
	}{
		{name: "acquisition failure", token: func(*testing.T) ([]byte, error) { return nil, errors.New("bearer secret-token@example.invalid") }, want: "could not be acquired"},
		{name: "missing project", token: func(t *testing.T) ([]byte, error) {
			return ampReadinessJWT(t, map[string]any{"workspace_id": "workspace_123"}), nil
		}, want: "invalid identity claims"},
		{name: "invalid project", token: func(t *testing.T) ([]byte, error) {
			return ampReadinessJWT(t, map[string]any{"project_id": "private@example.invalid"}), nil
		}, want: "invalid identity claims"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAmpReadinessFixture(t)
			ampOrbIdentityRequestTokenPath = filepath.Join(t.TempDir(), "workload-identity-request-token")
			ampReadinessWriteFile(t, ampOrbIdentityRequestTokenPath, []byte("request\n"), 0o600)
			ampOrbProjectIDToken = func(_ ...string) ([]byte, error) { return tc.token(t) }
			record, err := fixture.run(fixture.archive)
			var exit ExitError
			if !errors.As(err, &exit) || exit.Code != 2 || record.Ready || !strings.Contains(strings.Join(record.Findings, "\n"), tc.want) {
				t.Fatalf("err = %v, record = %#v", err, record)
			}
			body, _ := json.Marshal(record)
			if bytes.Contains(body, []byte("secret-token")) || bytes.Contains(body, []byte("example.invalid")) {
				t.Fatalf("readiness record leaked token error: %s", body)
			}
		})
	}
}

func TestAmpReadinessOrbWorkspaceClaimIsOptional(t *testing.T) {
	fixture := newAmpReadinessFixture(t)
	ampOrbIdentityRequestTokenPath = filepath.Join(t.TempDir(), "workload-identity-request-token")
	ampReadinessWriteFile(t, ampOrbIdentityRequestTokenPath, []byte("request\n"), 0o600)
	ampOrbProjectIDToken = func(_ ...string) ([]byte, error) {
		return ampReadinessJWT(t, map[string]any{"project_id": "amp-project-uuid-123"}), nil
	}
	record, err := fixture.run(fixture.archive)
	if err != nil || !record.Ready || record.AmpProjectID != "amp-project-uuid-123" || record.AmpWorkspaceID != "" {
		t.Fatalf("err = %v, record = %#v", err, record)
	}
	if !strings.Contains(strings.Join(record.Limitations, "\n"), "optional workspace_id") {
		t.Fatalf("limitations = %v", record.Limitations)
	}
}

func TestAmpReadinessKeyRefusals(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ampReadinessFixture)
		archive func(*ampReadinessFixture) string
		want    string
	}{
		{name: "archive checksum mismatch", archive: func(*ampReadinessFixture) string { return strings.Repeat("c", 64) }, want: "archive digest"},
		{name: "manifest version mismatch", mutate: func(f *ampReadinessFixture) {
			path := filepath.Join(f.release, loafReleaseManifestFile)
			body := strings.Replace(string(ampReadinessReadFile(t, path)), ampReadinessTestVersion, "9.8.8", 1)
			ampReadinessWriteFile(t, path, []byte(body), 0o644)
		}, want: "manifest package version"},
		{name: "missing distribution", mutate: func(f *ampReadinessFixture) {
			if err := os.RemoveAll(filepath.Join(f.release, "dist", "amp")); err != nil {
				t.Fatal(err)
			}
		}, want: "release files"},
		{name: "foreign plugin bytes", mutate: func(f *ampReadinessFixture) {
			ampReadinessWriteFile(t, filepath.Join(f.project, ".amp", "plugins", "loaf.ts"), []byte("foreign\n"), 0o644)
		}, want: "plugin is missing, foreign, or stale"},
		{name: "duplicate global source", mutate: func(f *ampReadinessFixture) {
			ampReadinessWriteFile(t, filepath.Join(f.home, ".agents", "skills", "foundations", "SKILL.md"), []byte("duplicate\n"), 0o644)
		}, want: "duplicate global Loaf skill"},
		{name: "remote mismatch", mutate: func(f *ampReadinessFixture) {
			ampReadinessGit(t, f.project, "remote", "set-url", "origin", "https://github.com/acme/other.git")
		}, want: "remote does not match"},
		{name: "PATH mismatch", mutate: func(f *ampReadinessFixture) {
			foreignDir := filepath.Join(f.home, "foreign-bin")
			ampReadinessWriteFile(t, filepath.Join(foreignDir, "loaf"), []byte("foreign binary\n"), 0o755)
			t.Setenv("PATH", foreignDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		}, want: "PATH loaf does not resolve"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAmpReadinessFixture(t)
			if tc.mutate != nil {
				tc.mutate(fixture)
			}
			archive := fixture.archive
			if tc.archive != nil {
				archive = tc.archive(fixture)
			}
			record, err := fixture.run(archive)
			var exit ExitError
			if !errors.As(err, &exit) || exit.Code != 2 || record.Ready {
				t.Fatalf("err = %v, record = %#v", err, record)
			}
			if !strings.Contains(strings.Join(record.Findings, "\n"), tc.want) {
				t.Fatalf("findings = %v, want %q", record.Findings, tc.want)
			}
		})
	}
}

func TestAmpReadinessRejectsUnsafeOrUncommittedPin(t *testing.T) {
	fixture := newAmpReadinessFixture(t)
	body := strings.Replace(string(ampReadinessReadFile(t, fixture.pin)), "git.remote=acme/ready", "git.remote=https://token@example.com/acme/ready", 1)
	ampReadinessWriteFile(t, fixture.pin, []byte(body), 0o644)
	record, err := fixture.run(fixture.archive)
	var exit ExitError
	if !errors.As(err, &exit) || exit.Code != 2 || !strings.Contains(strings.Join(record.Findings, "\n"), "committed regular file") {
		t.Fatalf("err = %v, findings = %v", err, record.Findings)
	}
}

func (f *ampReadinessFixture) run(archive string) (ampReadinessRecord, error) {
	f.t.Helper()
	var out bytes.Buffer
	runner := Runner{
		Stdout: &out, Stderr: &bytes.Buffer{}, WorkingDir: f.project,
		Executable: func() (string, error) { return f.binary, nil },
	}
	err := runner.Run([]string{"harness", "readiness", "--target", "amp", "--pin", f.pin, "--archive-sha256", archive, "--json"})
	var record ampReadinessRecord
	if decodeErr := json.Unmarshal(out.Bytes(), &record); decodeErr != nil {
		f.t.Fatalf("decode output %q: %v", out.String(), decodeErr)
	}
	return record, err
}

func ampReadinessReleaseManifest(t *testing.T, root string) loafReleaseManifest {
	t.Helper()
	runtimeID, err := ampReadinessRuntimeID()
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{"bin/loaf"}
	dist := filepath.Join(root, "dist", "amp")
	if err := filepath.WalkDir(dist, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dist || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	manifest := loafReleaseManifest{SchemaVersion: 1, PackageVersion: ampReadinessTestVersion, Target: runtimeID}
	for _, rel := range paths {
		path := filepath.Join(root, filepath.FromSlash(rel))
		digest, err := sha256RegularFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Files = append(manifest.Files, loafReleaseManifestRecord{Path: rel, SHA256: digest, Mode: uint32(info.Mode().Perm())})
	}
	return manifest
}

func ampReadinessJWT(t *testing.T, claims map[string]any) []byte {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".fixture-signature")
}

func ampReadinessWriteFile(t *testing.T, path string, body []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, mode); err != nil {
		t.Fatal(err)
	}
}

func ampReadinessReadFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func ampReadinessGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func ampReadinessTreeSnapshot(t *testing.T, root string) string {
	t.Helper()
	var rows []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		row := filepath.ToSlash(rel) + fmt.Sprintf(":%o", info.Mode())
		if info.Mode().IsRegular() {
			digest, err := sha256RegularFile(path)
			if err != nil {
				return err
			}
			row += ":" + digest
		} else if info.Mode()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			row += ":" + target
		}
		rows = append(rows, row)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(rows)
	return strings.Join(rows, "\n")
}

func TestAmpReadinessRuntimeIDMatchesSupportedHost(t *testing.T) {
	id, err := ampReadinessRuntimeID()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if err != nil || (id != runtime.GOOS+"-arm64" && id != runtime.GOOS+"-x64") {
			t.Fatalf("runtime ID = %q, %v", id, err)
		}
	}
}
