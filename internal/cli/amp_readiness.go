package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/levifig/loaf/internal/project"
)

const (
	ampReadinessContractVersion = 1
	loafReleaseManifestFile     = "loaf-release-manifest.json"
	loafArchiveDigestFile       = ".loaf-archive.sha256"
	ampReadinessTokenAudience   = "urn:loaf:readiness"
)

var ampReadinessSafeValue = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/+#-]{0,255}$`)

var (
	ampOrbIdentityRequestTokenPath = "/run/amp/workload-identity-request-token"
	ampOrbProjectIDToken           = func(args ...string) ([]byte, error) {
		return exec.Command("amp", args...).Output()
	}
)

type ampReadinessOptions struct {
	target        string
	pinPath       string
	archiveSHA256 string
	json          bool
	help          bool
}

type ampReadinessPin struct {
	Version         string
	ArchiveSHA256   string
	ProjectID       string
	GitRemote       string
	TrackerProvider string
	TrackerScope    string
}

type ampReadinessSource struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type ampReadinessRecord struct {
	ContractVersion       int                  `json:"contract_version"`
	Ready                 bool                 `json:"ready"`
	Target                string               `json:"target"`
	ProjectID             string               `json:"project_id,omitempty"`
	ContinuityProjectID   string               `json:"continuity_project_id,omitempty"`
	CanonicalRemote       string               `json:"canonical_remote,omitempty"`
	TrackerProvider       string               `json:"tracker_provider,omitempty"`
	TrackerScope          string               `json:"tracker_scope,omitempty"`
	AmpProjectScope       string               `json:"amp_project_scope,omitempty"`
	AmpWorkspaceScope     string               `json:"amp_workspace_scope,omitempty"`
	AmpProjectID          string               `json:"amp_project_id,omitempty"`
	AmpWorkspaceID        string               `json:"amp_workspace_id,omitempty"`
	BinaryPath            string               `json:"binary_path,omitempty"`
	BinarySHA256          string               `json:"binary_sha256,omitempty"`
	ArchiveSHA256         string               `json:"archive_sha256,omitempty"`
	ArchiveSidecarPath    string               `json:"archive_sidecar_path,omitempty"`
	ReleaseManifestPath   string               `json:"release_manifest_path,omitempty"`
	ReleaseManifestSHA256 string               `json:"release_manifest_sha256,omitempty"`
	PackageVersion        string               `json:"package_version,omitempty"`
	Skills                []ampReadinessSource `json:"skills"`
	Plugins               []ampReadinessSource `json:"plugins"`
	Findings              []string             `json:"findings"`
	Limitations           []string             `json:"limitations"`
}

type loafReleaseManifest struct {
	SchemaVersion  int                         `json:"schema_version"`
	PackageVersion string                      `json:"package_version"`
	Target         string                      `json:"target"`
	Files          []loafReleaseManifestRecord `json:"files"`
}

type loafReleaseManifestRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}

func (r Runner) runHarnessCommand(args []string, out io.Writer, runtimeRoot string) error {
	if len(args) > 0 && args[0] == "readiness" {
		return r.runAmpReadiness(args[1:], out)
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		writeHarnessCommandHelp(out)
		return nil
	}
	return r.runHarness(args, out, runtimeRoot)
}

func writeHarnessCommandHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  loaf harness reconcile --target <target> [--json]")
	fmt.Fprintln(out, "  loaf harness readiness --target amp --pin <path> --archive-sha256 <hex> [--json]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Reconcile Loaf-owned harness content or prove a project-local Amp environment is release-pinned and ready.")
}

func writeAmpReadinessHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: loaf harness readiness --target amp --pin <path> --archive-sha256 <hex> [--json]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Read-only verification of the running Loaf release and effective project-local Amp sources.")
}

func parseAmpReadinessArgs(args []string) (ampReadinessOptions, error) {
	var options ampReadinessOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target", "--pin", "--archive-sha256":
			if i+1 >= len(args) {
				return options, fmt.Errorf("%s requires a value", args[i])
			}
			flag := args[i]
			i++
			switch flag {
			case "--target":
				options.target = args[i]
			case "--pin":
				options.pinPath = args[i]
			case "--archive-sha256":
				options.archiveSHA256 = args[i]
			}
		case "--json":
			options.json = true
		case "--help", "-h":
			options.help = true
		default:
			return options, fmt.Errorf("unknown harness readiness option %q", args[i])
		}
	}
	if options.help {
		return options, nil
	}
	if options.target == "" || options.pinPath == "" || options.archiveSHA256 == "" {
		return options, fmt.Errorf("--target, --pin, and --archive-sha256 are required")
	}
	if options.target != "amp" {
		return options, fmt.Errorf("harness readiness supports only target amp")
	}
	if !isLowerSHA256(options.archiveSHA256) {
		return options, fmt.Errorf("--archive-sha256 must be 64 lowercase hexadecimal characters")
	}
	return options, nil
}

func (r Runner) runAmpReadiness(args []string, out io.Writer) error {
	options, err := parseAmpReadinessArgs(args)
	if err != nil {
		return err
	}
	if options.help {
		writeAmpReadinessHelp(out)
		return nil
	}
	record := ampReadinessRecord{
		ContractVersion: ampReadinessContractVersion,
		Target:          options.target,
		ArchiveSHA256:   options.archiveSHA256,
		Skills:          []ampReadinessSource{},
		Plugins:         []ampReadinessSource{},
		Findings:        []string{},
		Limitations: []string{
			"Amp exposes no deterministic runtime API for enumerating loaded skill and plugin sources; readiness proves the documented project-local and global filesystem search locations.",
		},
	}
	if os.Getenv(projectEnvironmentEnv) != "1" {
		record.Findings = append(record.Findings, "LOAF_PROJECT_ENV=1 is required")
	}

	rootStart := r.WorkingDir
	if filepath.IsAbs(options.pinPath) {
		rootStart = filepath.Dir(options.pinPath)
	} else if rootStart != "" {
		rootStart = filepath.Join(rootStart, filepath.Dir(options.pinPath))
	}
	root, rootErr := project.ResolveRepositoryRoot(rootStart)
	if rootErr != nil {
		record.Findings = append(record.Findings, "project root could not be resolved")
		return finishAmpReadiness(out, options.json, record)
	}
	projectRoot := root.Path()
	pinPath, pinErr := resolveAmpReadinessPinPath(projectRoot, options.pinPath)
	if pinErr != nil {
		record.Findings = append(record.Findings, "pin must be a committed regular file inside the project: "+pinErr.Error())
	} else {
		pin, parseErr := readAmpReadinessPin(pinPath)
		if parseErr != nil {
			record.Findings = append(record.Findings, "pin is malformed or contains unsafe data")
		} else {
			record.ProjectID = pin.ProjectID
			record.CanonicalRemote = pin.GitRemote
			record.AmpProjectScope = pin.GitRemote
			record.TrackerProvider = pin.TrackerProvider
			record.TrackerScope = pin.TrackerScope
			verifyAmpOrbIdentity(&record)
			if pin.ArchiveSHA256 != options.archiveSHA256 {
				record.Findings = append(record.Findings, "supplied archive digest does not match the platform pin")
			}
			verifyAmpReadinessProjectIdentity(root, pin, &record)
			verifyAmpReadinessDistribution(r, pin, options.archiveSHA256, &record)
			verifyAmpReadinessInstall(projectRoot, pin.Version, &record)
		}
	}
	return finishAmpReadiness(out, options.json, record)
}

func verifyAmpOrbIdentity(record *ampReadinessRecord) {
	info, err := os.Lstat(ampOrbIdentityRequestTokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			record.Limitations = append(record.Limitations, "Amp Orb workload identity is not observable outside an Orb lifecycle environment; amp_project_id and amp_workspace_id are omitted.")
			return
		}
		record.Findings = append(record.Findings, "Amp Orb workload identity sentinel could not be inspected")
		return
	}
	if !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		record.Findings = append(record.Findings, "Amp Orb workload identity sentinel is not a regular file")
		return
	}
	token, err := ampOrbProjectIDToken("orb", "id-token", "--audience", ampReadinessTokenAudience, "--subject-scope", "project")
	if err != nil {
		record.Findings = append(record.Findings, "Amp Orb project identity token could not be acquired")
		return
	}
	projectID, workspaceID, err := decodeAmpOrbIdentityClaims(token)
	// Do not retain the bearer token beyond claim extraction.
	for i := range token {
		token[i] = 0
	}
	if err != nil {
		record.Findings = append(record.Findings, "Amp Orb project identity token contains invalid identity claims")
		return
	}
	record.AmpProjectID = projectID
	record.AmpWorkspaceID = workspaceID
	if workspaceID == "" {
		record.Limitations = append(record.Limitations, "The Amp Orb project token did not include the optional workspace_id claim; amp_workspace_id is omitted.")
	}
}

func decodeAmpOrbIdentityClaims(raw []byte) (string, string, error) {
	token := strings.TrimSpace(string(raw))
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("invalid JWT envelope")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || validateJSONNoDuplicateKeys(payload) != nil {
		return "", "", fmt.Errorf("invalid JWT payload")
	}
	var claims map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&claims); err != nil || decoder.Decode(&struct{}{}) != io.EOF || claims == nil {
		return "", "", fmt.Errorf("invalid JWT claims")
	}
	projectID, err := ampOrbIdentityStringClaim(claims, "project_id", true)
	if err != nil {
		return "", "", err
	}
	workspaceID, err := ampOrbIdentityStringClaim(claims, "workspace_id", false)
	if err != nil {
		return "", "", err
	}
	return projectID, workspaceID, nil
}

func ampOrbIdentityStringClaim(claims map[string]json.RawMessage, name string, required bool) (string, error) {
	raw, present := claims[name]
	if !present {
		if required {
			return "", fmt.Errorf("required identity claim is missing")
		}
		return "", nil
	}
	var value string
	if json.Unmarshal(raw, &value) != nil || !validAmpOrbIdentity(value) {
		return "", fmt.Errorf("identity claim is invalid")
	}
	return value, nil
}

func validAmpOrbIdentity(value string) bool {
	if value == "" || len(value) > 255 || strings.Contains(value, "@") || strings.Contains(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "..") {
		return false
	}
	return regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`).MatchString(value)
}

func finishAmpReadiness(out io.Writer, jsonOutput bool, record ampReadinessRecord) error {
	sort.Slice(record.Skills, func(i, j int) bool { return record.Skills[i].Path < record.Skills[j].Path })
	sort.Slice(record.Plugins, func(i, j int) bool { return record.Plugins[i].Path < record.Plugins[j].Path })
	record.Ready = len(record.Findings) == 0
	if jsonOutput {
		if err := writeJSON(out, record); err != nil {
			return err
		}
	} else if record.Ready {
		fmt.Fprintf(out, "Loaf Amp readiness: ready (%s, %s)\n", record.PackageVersion, record.CanonicalRemote)
		fmt.Fprintln(out, record.Limitations[0])
	} else {
		fmt.Fprintln(out, "Loaf Amp readiness: not ready")
		for _, finding := range record.Findings {
			fmt.Fprintf(out, "  - %s\n", finding)
		}
	}
	if !record.Ready {
		return ExitError{Code: 2}
	}
	return nil
}

func resolveAmpReadinessPinPath(projectRoot, raw string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(projectRoot); err == nil {
		projectRoot = resolved
	}
	path := raw
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if resolved, resolveErr := filepath.EvalSymlinks(abs); resolveErr == nil {
		abs = resolved
	}
	rel, err := filepath.Rel(projectRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == "." {
		return "", fmt.Errorf("pin outside project")
	}
	info, err := os.Lstat(abs)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("pin is not a regular file")
	}
	rel = filepath.ToSlash(rel)
	cmd := exec.Command("git", "cat-file", "-e", "HEAD:"+rel)
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pin is not committed")
	}
	cmd = exec.Command("git", "diff", "--quiet", "HEAD", "--", rel)
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pin differs from committed bytes")
	}
	return abs, nil
}

func readAmpReadinessPin(path string) (ampReadinessPin, error) {
	body, err := readRegularFile(path, 32*1024)
	if err != nil || len(body) == 0 || body[len(body)-1] != '\n' || bytes.Contains(body, []byte{'\r'}) {
		return ampReadinessPin{}, fmt.Errorf("invalid pin file")
	}
	archiveKey, err := ampReadinessPinArchiveKey()
	if err != nil {
		return ampReadinessPin{}, err
	}
	wantOrder := []string{"schema", "version", "archive.linux-x64.sha256", "archive.linux-arm64.sha256", "project.id", "git.remote", "tracker.provider", "tracker.scope"}
	lines := strings.Split(string(body[:len(body)-1]), "\n")
	if len(lines) != len(wantOrder) {
		return ampReadinessPin{}, fmt.Errorf("pin must contain exactly eight lines")
	}
	values := map[string]string{}
	for i, line := range lines {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key != wantOrder[i] || value == "" {
			return ampReadinessPin{}, fmt.Errorf("invalid pin line")
		}
		if _, duplicate := values[key]; duplicate {
			return ampReadinessPin{}, fmt.Errorf("duplicate pin key")
		}
		values[key] = value
	}
	if values["schema"] != "1" || !isLowerSHA256(values["archive.linux-x64.sha256"]) || !isLowerSHA256(values["archive.linux-arm64.sha256"]) {
		return ampReadinessPin{}, fmt.Errorf("unsupported pin schema or digest")
	}
	if _, ok := parseUpgradeSemver(values["version"]); !ok || values["version"] == packageVersionUnknown {
		return ampReadinessPin{}, fmt.Errorf("invalid pin version")
	}
	for _, key := range []string{"project.id", "tracker.provider", "tracker.scope"} {
		if !safeAmpReadinessValue(values[key]) {
			return ampReadinessPin{}, fmt.Errorf("unsafe pin value")
		}
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`).MatchString(values["tracker.provider"]) {
		return ampReadinessPin{}, fmt.Errorf("invalid tracker provider")
	}
	if !validAmpPinnedRemote(values["git.remote"]) {
		return ampReadinessPin{}, fmt.Errorf("unsafe remote")
	}
	return ampReadinessPin{
		Version: values["version"], ArchiveSHA256: values[archiveKey], ProjectID: values["project.id"],
		GitRemote: values["git.remote"], TrackerProvider: values["tracker.provider"], TrackerScope: values["tracker.scope"],
	}, nil
}

func ampReadinessPinArchiveKey() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "archive.linux-x64.sha256", nil
	case "arm64":
		return "archive.linux-arm64.sha256", nil
	default:
		return "", fmt.Errorf("unsupported Amp Orb architecture")
	}
}

func validAmpPinnedRemote(value string) bool {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if !regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]{0,99})$`).MatchString(part) || strings.Contains(part, "..") {
			return false
		}
	}
	return safeAmpReadinessValue(value)
}

func safeAmpReadinessValue(value string) bool {
	lower := strings.ToLower(value)
	if !ampReadinessSafeValue.MatchString(value) || strings.Contains(value, "\\") || strings.Contains(value, "@") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") || strings.Contains(value, "..") || strings.Contains(value, "://") {
		return false
	}
	for _, sensitive := range []string{"password", "passwd", "token", "secret", "credential", "authorization", "bearer"} {
		if strings.Contains(lower, sensitive) {
			return false
		}
	}
	return true
}

func unsafeAmpReadinessRemote(raw string) bool {
	lower := strings.ToLower(raw)
	for _, sensitive := range []string{"password", "passwd", "token=", "secret=", "authorization", "bearer "} {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") || strings.HasPrefix(lower, "file:") || strings.Contains(raw, "\\") {
		return true
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		return err != nil || parsed.User != nil || parsed.Hostname() == ""
	}
	if at := strings.Index(raw, "@"); at >= 0 && !strings.HasPrefix(raw, "git@") {
		return true
	}
	return false
}

func ampReadinessRuntimeID() (string, error) {
	var osPart string
	switch runtime.GOOS {
	case "darwin", "linux":
		osPart = runtime.GOOS
	default:
		return "", fmt.Errorf("unsupported Amp Orb operating system")
	}
	var archPart string
	switch runtime.GOARCH {
	case "arm64":
		archPart = "arm64"
	case "amd64":
		archPart = "x64"
	default:
		return "", fmt.Errorf("unsupported Amp Orb architecture")
	}
	return osPart + "-" + archPart, nil
}

func verifyAmpReadinessProjectIdentity(root project.Root, pin ampReadinessPin, record *ampReadinessRecord) {
	remote, err := canonicalAmpGitRemote(root)
	if err != nil || remote == "" {
		record.Findings = append(record.Findings, "canonical Git origin remote is unavailable")
	} else if remote != pin.GitRemote {
		record.Findings = append(record.Findings, "canonical Git origin remote does not match the pin")
	}
	if conf, err := project.ReadProjectConf(root); err == nil {
		record.ContinuityProjectID = conf.ProjectID
		if conf.ProjectID != pin.ProjectID {
			record.Findings = append(record.Findings, "continuity project identity does not match the pin")
		}
	}
}

func canonicalAmpGitRemote(root project.Root) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = root.Path()
	body, err := cmd.Output()
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(string(body))
	if unsafeAmpReadinessRemote(raw) {
		return "", fmt.Errorf("origin contains authentication material")
	}
	normalized, err := project.NormalizeRemoteURL(raw)
	if err != nil {
		return "", err
	}
	_, repository, ok := strings.Cut(normalized, "/")
	if !ok || !validAmpPinnedRemote(repository) {
		return "", fmt.Errorf("origin is not an owner/repository remote")
	}
	return repository, nil
}

func verifyAmpReadinessDistribution(r Runner, pin ampReadinessPin, suppliedArchive string, record *ampReadinessRecord) {
	root, err := r.resolveInstalledDistributionRoot()
	if err != nil {
		record.Findings = append(record.Findings, "running installed distribution could not be resolved")
		return
	}
	record.PackageVersion = packageVersion(root)
	if record.PackageVersion != pin.Version {
		record.Findings = append(record.Findings, "running package version does not match the pin")
	}

	sidecarPath := filepath.Join(root, loafArchiveDigestFile)
	record.ArchiveSidecarPath = sidecarPath
	if digest, err := readAmpArchiveDigestSidecar(sidecarPath); err != nil {
		record.Findings = append(record.Findings, "installed archive digest sidecar is missing or malformed")
	} else if digest != suppliedArchive || digest != pin.ArchiveSHA256 {
		record.Findings = append(record.Findings, "installed archive digest sidecar does not match the supplied digest and pin")
	}

	manifestPath := filepath.Join(root, loafReleaseManifestFile)
	record.ReleaseManifestPath = manifestPath
	manifestBody, err := readRegularFile(manifestPath, projectFileReadLimit)
	if err != nil {
		record.Findings = append(record.Findings, "release manifest is missing or unreadable")
		return
	}
	record.ReleaseManifestSHA256 = sha256Bytes(manifestBody)
	manifest, err := decodeLoafReleaseManifest(manifestBody)
	if err != nil {
		record.Findings = append(record.Findings, "release manifest is malformed")
		return
	}
	runtimeID, runtimeErr := ampReadinessRuntimeID()
	if runtimeErr != nil || manifest.Target != runtimeID {
		record.Findings = append(record.Findings, "release manifest target does not match this runtime")
	}
	if manifest.PackageVersion != record.PackageVersion || manifest.PackageVersion != pin.Version {
		record.Findings = append(record.Findings, "release manifest package version does not agree with the package and pin")
	}
	executable, err := r.executablePath()
	if err != nil {
		record.Findings = append(record.Findings, "running executable path is unavailable")
		return
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}
	executable, _ = filepath.Abs(executable)
	record.BinaryPath = filepath.Clean(executable)
	wantExecutable, _ := filepath.EvalSymlinks(filepath.Join(root, "bin", "loaf"))
	if filepath.Clean(executable) != filepath.Clean(wantExecutable) {
		record.Findings = append(record.Findings, "running executable is not the release manifest binary")
	}
	pathExecutable, pathErr := exec.LookPath("loaf")
	if pathErr != nil {
		record.Findings = append(record.Findings, "PATH loaf is unavailable")
	} else {
		if resolved, resolveErr := filepath.EvalSymlinks(pathExecutable); resolveErr == nil {
			pathExecutable = resolved
		}
		pathExecutable, _ = filepath.Abs(pathExecutable)
		if filepath.Clean(pathExecutable) != filepath.Clean(executable) {
			record.Findings = append(record.Findings, "PATH loaf does not resolve to the verified release binary")
		}
	}
	if digest, digestErr := sha256RegularFile(executable); digestErr != nil {
		record.Findings = append(record.Findings, "running executable could not be hashed")
	} else {
		record.BinarySHA256 = digest
	}
	if err := verifyLoafReleaseFiles(root, manifest, record.BinaryPath, record.BinarySHA256); err != nil {
		record.Findings = append(record.Findings, "release files do not match the manifest: "+err.Error())
	}
}

func readAmpArchiveDigestSidecar(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("invalid sidecar")
	}
	body, err := os.ReadFile(path)
	if err != nil || len(body) != 65 || body[64] != '\n' || !isLowerSHA256(string(body[:64])) {
		return "", fmt.Errorf("invalid sidecar")
	}
	return string(body[:64]), nil
}

func decodeLoafReleaseManifest(body []byte) (loafReleaseManifest, error) {
	if err := validateJSONNoDuplicateKeys(body); err != nil {
		return loafReleaseManifest{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var manifest loafReleaseManifest
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return manifest, fmt.Errorf("trailing JSON")
	}
	if manifest.SchemaVersion != 1 || manifest.PackageVersion == "" || manifest.Target == "" || len(manifest.Files) == 0 {
		return manifest, fmt.Errorf("invalid manifest metadata")
	}
	last := ""
	for _, file := range manifest.Files {
		if file.Path <= last || !validReleaseManifestPath(file.Path) || !isLowerSHA256(file.SHA256) || (file.Mode != 0o644 && file.Mode != 0o755) {
			return manifest, fmt.Errorf("invalid manifest file record")
		}
		last = file.Path
	}
	return manifest, nil
}

func validReleaseManifestPath(path string) bool {
	if path == "bin/loaf" {
		return true
	}
	return strings.HasPrefix(path, "dist/amp/") && path != "dist/amp/" && filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) == path && !strings.Contains(path, "..")
}

func verifyLoafReleaseFiles(root string, manifest loafReleaseManifest, executablePath, executableDigest string) error {
	want := make(map[string]loafReleaseManifestRecord, len(manifest.Files))
	for _, file := range manifest.Files {
		want[file.Path] = file
	}
	binary, ok := want["bin/loaf"]
	if !ok || binary.SHA256 != executableDigest {
		return fmt.Errorf("binary digest mismatch")
	}
	if info, err := os.Lstat(executablePath); err != nil || !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 || uint32(info.Mode().Perm()) != binary.Mode {
		return fmt.Errorf("binary mode mismatch")
	}
	distRoot := filepath.Join(root, "dist", "amp")
	info, err := os.Lstat(distRoot)
	if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("Amp distribution missing")
	}
	actual := map[string]bool{"bin/loaf": true}
	err = filepath.WalkDir(distRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == distRoot {
			return nil
		}
		entryInfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if entryInfo.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("Amp distribution contains a symlink")
		}
		if entryInfo.IsDir() {
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("Amp distribution contains a non-regular file")
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		record, present := want[rel]
		if !present {
			return fmt.Errorf("extra Amp distribution file")
		}
		digest, err := sha256RegularFile(path)
		if err != nil || digest != record.SHA256 || uint32(entryInfo.Mode().Perm()) != record.Mode {
			return fmt.Errorf("Amp distribution file digest or mode mismatch")
		}
		actual[rel] = true
		return nil
	})
	if err != nil {
		return err
	}
	for path := range want {
		if !actual[path] {
			return fmt.Errorf("manifest file is missing")
		}
	}
	return nil
}

func verifyAmpReadinessInstall(projectRoot, version string, record *ampReadinessRecord) {
	skillsDir := filepath.Join(projectRoot, ".agents", "skills")
	pluginsDir := filepath.Join(projectRoot, ".amp", "plugins")
	ampDir := filepath.Join(projectRoot, ".amp")
	if marker, err := readStrictVersionMarker(filepath.Join(ampDir, loafInstallMarkerFile)); err != nil || marker != version {
		record.Findings = append(record.Findings, "project-local Amp version marker is missing or stale")
	}
	if err := verifyAmpInstallRecord(projectRoot, version, ampDir, skillsDir); err != nil {
		record.Findings = append(record.Findings, "project-local Amp install record is missing or stale")
	}
	loafRoot, err := findDistributionFromRecord(record)
	if err != nil {
		return
	}
	distDir := filepath.Join(loafRoot, "dist", "amp")
	expectedSkills := verifyAmpSkills(distDir, skillsDir, record)
	verifyAmpPlugins(distDir, ampDir, pluginsDir, version, record)
	verifyAmpGlobalDuplicates(projectRoot, skillsDir, pluginsDir, expectedSkills, record)
}

func findDistributionFromRecord(record *ampReadinessRecord) (string, error) {
	if record.ReleaseManifestPath == "" {
		return "", fmt.Errorf("missing release manifest")
	}
	return filepath.Dir(record.ReleaseManifestPath), nil
}

func readStrictVersionMarker(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("invalid marker")
	}
	body, err := os.ReadFile(path)
	if err != nil || len(body) < 2 || body[len(body)-1] != '\n' || bytes.Contains(body[:len(body)-1], []byte{'\n', '\r'}) {
		return "", fmt.Errorf("invalid marker")
	}
	return string(body[:len(body)-1]), nil
}

func verifyAmpInstallRecord(projectRoot, version, ampDir, skillsDir string) error {
	path := installRecordPath(projectRoot, "amp")
	body, err := readRegularFile(path, projectFileReadLimit)
	if err != nil || validateJSONNoDuplicateKeys(body) != nil {
		return fmt.Errorf("invalid install record")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var record installTargetRecord
	if decoder.Decode(&record) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("invalid install record")
	}
	if record.Version != version || record.Target != "amp" || canonicalAmpReadinessPath(record.ConfigDir) != canonicalAmpReadinessPath(ampDir) || canonicalAmpReadinessPath(record.SkillsDir) != canonicalAmpReadinessPath(skillsDir) {
		return fmt.Errorf("stale install record")
	}
	return nil
}

func canonicalAmpReadinessPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	if resolved, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
		return filepath.Join(resolved, filepath.Base(path))
	}
	return filepath.Clean(path)
}

func verifyAmpSkills(distDir, skillsDir string, record *ampReadinessRecord) []string {
	expected, err := listInstallSkillDirs(filepath.Join(distDir, "skills"))
	if err != nil || len(expected) == 0 {
		record.Findings = append(record.Findings, "release Amp skills distribution is missing")
		return nil
	}
	state, err := readManagedSkillsState(skillsDir)
	if err != nil || state.legacy {
		record.Findings = append(record.Findings, "project-local skill ownership manifest is missing or not digest-owned")
		return expected
	}
	if len(state.digests) != len(expected) {
		record.Findings = append(record.Findings, "project-local skill ownership set does not match the release")
	}
	for _, name := range expected {
		sourceDigest, sourceErr := hashInstallSkillTree(filepath.Join(distDir, "skills", name))
		installedPath := filepath.Join(skillsDir, name)
		installedDigest, installedErr := hashInstallSkillTree(installedPath)
		if installedErr == nil {
			record.Skills = append(record.Skills, ampReadinessSource{Path: installedPath, SHA256: installedDigest})
		}
		ownedDigest, owned := state.digests[name]
		if sourceErr != nil || installedErr != nil || !owned || ownedDigest != sourceDigest || installedDigest != sourceDigest {
			record.Findings = append(record.Findings, "project-local skill source is missing, foreign, or stale: "+name)
			continue
		}
	}
	return expected
}

func verifyAmpPlugins(distDir, ampDir, pluginsDir, version string, record *ampReadinessRecord) {
	desired, err := readTargetAdapterManifest(filepath.Join(distDir, targetBuildManifestFile))
	if err != nil || desired.Target != "amp" || desired.PackageVersion != version {
		record.Findings = append(record.Findings, "release Amp target manifest is missing or stale")
		return
	}
	installed, err := readTargetAdapterManifest(filepath.Join(ampDir, targetInstallManifestFile))
	if err != nil || installed.Target != "amp" || installed.PackageVersion != version {
		record.Findings = append(record.Findings, "project-local Amp ownership manifest is missing or stale")
		return
	}
	expectedNames := map[string]bool{}
	installedByID := map[string]targetAdapterArtifact{}
	for _, artifact := range installed.Artifacts {
		installedByID[artifact.ID] = artifact
	}
	for _, artifact := range desired.Artifacts {
		if artifact.Kind != "plugin" {
			continue
		}
		name := filepath.Base(filepath.FromSlash(artifact.Destination))
		expectedNames[name] = true
		owned, ok := installedByID[artifact.ID]
		path := filepath.Join(ampDir, filepath.FromSlash(artifact.Destination))
		digest, digestErr := sha256RegularFile(path)
		if digestErr == nil {
			record.Plugins = append(record.Plugins, ampReadinessSource{Path: path, SHA256: digest})
		}
		if !ok || owned.SourcePath != artifact.SourcePath || owned.Destination != artifact.Destination || owned.SHA256 != artifact.SHA256 || digestErr != nil || digest != artifact.SHA256 {
			record.Findings = append(record.Findings, "project-local Amp plugin is missing, foreign, or stale: "+name)
			continue
		}
	}
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		record.Findings = append(record.Findings, "project-local Amp plugin directory is missing")
		return
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "loaf") && strings.HasSuffix(entry.Name(), ".ts") && !expectedNames[entry.Name()] {
			path := filepath.Join(pluginsDir, entry.Name())
			if digest, err := sha256RegularFile(path); err == nil {
				record.Plugins = append(record.Plugins, ampReadinessSource{Path: path, SHA256: digest})
			}
			record.Findings = append(record.Findings, "foreign project-local Loaf plugin source is effective: "+entry.Name())
		}
	}
}

func verifyAmpGlobalDuplicates(projectRoot, skillsDir, pluginsDir string, skillNames []string, record *ampReadinessRecord) {
	home := installHome()
	if home == "" {
		record.Findings = append(record.Findings, "HOME is unavailable for duplicate-source verification")
		return
	}
	for _, name := range skillNames {
		for _, globalRoot := range []string{filepath.Join(home, ".config", "agents", "skills"), filepath.Join(home, ".agents", "skills")} {
			candidate := filepath.Join(globalRoot, name)
			if filepath.Clean(candidate) != filepath.Clean(filepath.Join(skillsDir, name)) && pathExistsLstat(candidate) {
				if digest, err := hashInstallSkillTree(candidate); err == nil {
					record.Skills = append(record.Skills, ampReadinessSource{Path: candidate, SHA256: digest})
				}
				record.Findings = append(record.Findings, "duplicate global Loaf skill source is effective: "+name)
			}
		}
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	for _, globalPlugins := range []string{filepath.Join(xdg, "amp", "plugins"), filepath.Join(home, ".amp", "plugins")} {
		if filepath.Clean(globalPlugins) == filepath.Clean(pluginsDir) || strings.HasPrefix(filepath.Clean(globalPlugins), filepath.Clean(projectRoot)+string(filepath.Separator)) {
			continue
		}
		entries, err := os.ReadDir(globalPlugins)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "loaf") && strings.HasSuffix(entry.Name(), ".ts") {
				path := filepath.Join(globalPlugins, entry.Name())
				if digest, err := sha256RegularFile(path); err == nil {
					record.Plugins = append(record.Plugins, ampReadinessSource{Path: path, SHA256: digest})
				}
				record.Findings = append(record.Findings, "duplicate global Loaf Amp plugin source is effective: "+entry.Name())
			}
		}
	}
}

func pathExistsLstat(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func sha256RegularFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("not a regular file")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func isLowerSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
