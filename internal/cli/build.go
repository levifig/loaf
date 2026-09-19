package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var defaultBuildTargets = []string{"claude-code", "opencode", "cursor", "codex", "amp"}

type buildOptions struct {
	target string
	help   bool
}

func (r Runner) runBuild(args []string, out io.Writer, runtimeRoot string) error {
	options, err := parseBuildArgs(args)
	if err != nil {
		return err
	}
	if options.help {
		writeBuildHelp(out)
		return nil
	}
	loafRoot, err := resolveSourceCheckoutRoot(r.WorkingDir, runtimeRoot)
	if err != nil {
		return err
	}
	return runBuildContent(loafRoot, options, out)
}

func parseBuildArgs(args []string) (buildOptions, error) {
	var options buildOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			options.help = true
		case arg == "-t" || arg == "--target":
			if i+1 >= len(args) {
				return buildOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			options.target = args[i]
		case strings.HasPrefix(arg, "--target="):
			options.target = strings.TrimPrefix(arg, "--target=")
			if options.target == "" {
				return buildOptions{}, fmt.Errorf("--target requires a value")
			}
		default:
			return buildOptions{}, fmt.Errorf("unknown build option %q", arg)
		}
	}
	return options, nil
}

func writeBuildHelp(out io.Writer) {
	fmt.Fprintln(out, strings.Join([]string{
		"Usage: loaf build [options]",
		"",
		"Build skill distributions for agent harnesses.",
		"",
		"Options:",
		"  -t, --target <name>  Build a specific target only",
		"  -h, --help           Show help",
	}, "\n"))
}

func runBuildContent(loafRoot string, options buildOptions, out io.Writer) error {
	if options.target == "" {
		return runNativeBuildAllTargets(loafRoot, out)
	}
	targets, err := nativeBuildTargetNames(loafRoot)
	if err != nil {
		return err
	}
	if !containsBuildTarget(targets, options.target) {
		return fmt.Errorf("error: Unknown target %s\nValid targets: %s", options.target, strings.Join(targets, ", "))
	}
	return runNativeBuildTarget(loafRoot, out, options.target)
}

func runNativeBuildTarget(root string, out io.Writer, targetName string) error {
	start := time.Now()
	fmt.Fprintf(out, "\n%s\n\n", ansiBold("loaf build"))

	sharedStart := time.Now()
	fmt.Fprintf(out, "  %s shared skills intermediate...", ansiCyan("building"))
	if err := buildNativeSharedSkillsIntermediate(root); err != nil {
		fmt.Fprintf(out, "\r  %s shared skills intermediate\n", ansiRed("✗"))
		return err
	}
	fmt.Fprintf(out, "\r  %s shared skills intermediate %s\n", ansiGreen("✓"), ansiGray("("+elapsedSeconds(sharedStart)+")"))

	targetStart := time.Now()
	fmt.Fprintf(out, "  %s %s...", ansiCyan("building"), targetName)
	warnings, err := buildNativeTargetOnly(root, targetName)
	if err != nil {
		fmt.Fprintf(out, "\r  %s %s\n", ansiRed("✗"), targetName)
		return err
	}
	fmt.Fprintf(out, "\r  %s %s %s\n", ansiGreen("✓"), targetName, ansiGray("("+elapsedSeconds(targetStart)+")"))
	for _, warning := range warnings {
		fmt.Fprintf(out, "    %s %s\n", ansiYellow("warn"), warning)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %s\n", ansiGreen("Build complete"), ansiGray("("+elapsedSeconds(start)+")"))
	return nil
}

func runNativeBuildAllTargets(root string, out io.Writer) error {
	start := time.Now()
	fmt.Fprintf(out, "\n%s\n\n", ansiBold("loaf build"))

	sharedStart := time.Now()
	fmt.Fprintf(out, "  %s shared skills intermediate...", ansiCyan("building"))
	if err := buildNativeSharedSkillsIntermediate(root); err != nil {
		fmt.Fprintf(out, "\r  %s shared skills intermediate\n", ansiRed("✗"))
		return err
	}
	fmt.Fprintf(out, "\r  %s shared skills intermediate %s\n", ansiGreen("✓"), ansiGray("("+elapsedSeconds(sharedStart)+")"))

	failed := false
	for _, targetName := range defaultBuildTargets {
		targetStart := time.Now()
		fmt.Fprintf(out, "  %s %s...", ansiCyan("building"), targetName)
		warnings, err := buildNativeTargetOnly(root, targetName)
		if err != nil {
			fmt.Fprintf(out, "\r  %s %s\n", ansiRed("✗"), targetName)
			fmt.Fprintf(out, "    %s\n", ansiRed(err.Error()))
			failed = true
			continue
		}
		fmt.Fprintf(out, "\r  %s %s %s\n", ansiGreen("✓"), targetName, ansiGray("("+elapsedSeconds(targetStart)+")"))
		for _, warning := range warnings {
			fmt.Fprintf(out, "    %s %s\n", ansiYellow("warn"), warning)
		}
	}
	fmt.Fprintln(out)
	if failed {
		return fmt.Errorf("%s %s", ansiRed("Build failed"), ansiGray("("+elapsedSeconds(start)+")"))
	}
	fmt.Fprintf(out, "%s %s\n", ansiGreen("Build complete"), ansiGray("("+elapsedSeconds(start)+")"))
	return nil
}

func buildNativeTargetOnly(root string, targetName string) ([]string, error) {
	var err error
	switch targetName {
	case "claude-code":
		err = buildNativeClaudeCodeTarget(root)
	case "opencode":
		err = buildNativeOpenCodeTarget(root)
	case "cursor":
		err = buildNativeCursorTarget(root)
	case "codex":
		err = buildNativeCodexTarget(root)
	case "amp":
		err = buildNativeAmpTarget(root)
	default:
		return nil, fmt.Errorf("native build target %s is not implemented", targetName)
	}
	if err != nil {
		return nil, err
	}
	if err := writeNativeBuildTargetManifest(root, targetName); err != nil {
		return nil, err
	}
	return validateNativeBuildArtifacts(root, targetName)
}

func nativeBuildTargetNames(loafRoot string) ([]string, error) {
	path := filepath.Join(loafRoot, "config", "targets.yaml")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return append([]string{}, defaultBuildTargets...), nil
		}
		return nil, err
	}
	defer file.Close()

	var targets []string
	inTargets := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			inTargets = trimmed == "targets:"
			continue
		}
		if !inTargets {
			continue
		}
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			name := strings.TrimSuffix(trimmed, ":")
			if name != "" {
				targets = append(targets, name)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return append([]string{}, defaultBuildTargets...), nil
	}
	return targets, nil
}

func containsBuildTarget(targets []string, target string) bool {
	for _, candidate := range targets {
		if candidate == target {
			return true
		}
	}
	return false
}

func validateNativeBuildArtifacts(root string, targetName string) ([]string, error) {
	outputDir := nativeBuildTargetOutputDir(root, targetName)
	var jsFiles []string
	var tsFiles []string
	if err := filepath.WalkDir(outputDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".mjs", ".cjs":
			jsFiles = append(jsFiles, path)
		case ".ts", ".mts", ".cts":
			tsFiles = append(tsFiles, path)
		}
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(tsFiles) != 0 {
		return nil, fmt.Errorf("generated TypeScript adapters are no longer supported: %s", strings.Join(nativeBuildRelativePaths(root, tsFiles), ", "))
	}
	for _, path := range jsFiles {
		if err := runNativeBuildArtifactCheck("node", []string{"--check", path}); err != nil {
			return nil, fmt.Errorf("JavaScript validation failed for %s: %w", nativeBuildRelativePath(root, path), err)
		}
	}
	if err := validateNativeBuildUnresolvedPlaceholders(root, targetName); err != nil {
		return nil, err
	}
	return nil, nil
}

// nativeBuildInstallTimePlaceholders are {{TOKEN}} forms that generated non-skill
// artifacts may still carry at build time because install expands them into
// PATH `loaf` command prefixes. Any other {{TOKEN}} in those artifacts is a
// stray placeholder and fails the build.
var nativeBuildInstallTimePlaceholders = map[string]bool{
	"{{LOAF_BASIC_RULES}}": true,
}

type nativeBuildUnresolvedPlaceholderFinding struct {
	path  string
	line  int
	token string
}

// nativeBuildUnresolvedPlaceholderFindingCap bounds how many findings we retain
// per artifact and in the aggregate diagnostic. An artifact of millions of
// newline-separated {{TOKEN}} values would otherwise grow the slice and the
// error string with the whole file; the scanner still counts every hit so the
// message can say "showing N of M" instead of silently truncating.
const nativeBuildUnresolvedPlaceholderFindingCap = 32

// validateNativeBuildUnresolvedPlaceholders rejects unresolved {{TOKEN}} forms in
// generated non-skill artifacts. Skills/, commands/, and agents/ still carry
// retired content tokens until TASK-003, so those trees are skipped. Everything
// else under the target output — including .js plugins, shell/Python hooks, and
// instruction Markdown — is scanned. The root adapter manifest is excluded by
// exact path (not basename) so a nested copy named .loaf-target-manifest.json
// cannot hide unresolved tokens.
func validateNativeBuildUnresolvedPlaceholders(root string, targetName string) error {
	outputDir := nativeBuildTargetOutputDir(root, targetName)
	manifestPath := filepath.Join(outputDir, targetBuildManifestFile)
	var paths []string
	err := filepath.WalkDir(outputDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			// Skills and OpenCode commands still carry retired content tokens until
			// TASK-003. Agents are authored profiles, not the generated Codex/config
			// artifacts this guard targets. bin/ is walked: text launchers and
			// package.json there are scanned; opaque native binaries are skipped
			// later by magic-byte classification, not by directory name.
			if name == "node_modules" || name == "skills" || name == "commands" || name == "agents" {
				return filepath.SkipDir
			}
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var findings []nativeBuildUnresolvedPlaceholderFinding
	totalFound := 0
	for _, path := range paths {
		if path == manifestPath {
			continue
		}
		relative := nativeBuildRelativePath(root, path)
		opaque, fileFindings, fileTotal, err := scanNativeBuildArtifactPlaceholders(path, relative)
		if err != nil {
			return err
		}
		if opaque {
			continue
		}
		totalFound += fileTotal
		room := nativeBuildUnresolvedPlaceholderFindingCap - len(findings)
		if room <= 0 {
			continue
		}
		if len(fileFindings) > room {
			findings = append(findings, fileFindings[:room]...)
		} else {
			findings = append(findings, fileFindings...)
		}
	}
	if totalFound == 0 {
		return nil
	}
	var out strings.Builder
	if totalFound > len(findings) {
		out.WriteString(fmt.Sprintf(
			"unresolved placeholder lint failed: showing %d of %d findings (%d suppressed):",
			len(findings), totalFound, totalFound-len(findings),
		))
	} else {
		out.WriteString("unresolved placeholder lint failed:")
	}
	for _, finding := range findings {
		out.WriteString("\n")
		out.WriteString(finding.path)
		out.WriteString(":")
		out.WriteString(fmt.Sprintf("%d", finding.line))
		out.WriteString(": unresolved harness token ")
		out.WriteString(finding.token)
	}
	return errors.New(out.String())
}

// nativeBuildOpacityProbeBytes is how many leading bytes we read to decide
// whether an artifact is an opaque native binary. Mach-O, ELF, and PE magic all
// sit in the first four bytes; the probe is larger only so a future format with
// a slightly deeper signature still classifies without a full read.
const nativeBuildOpacityProbeBytes = 512

// isOpaqueNativeBuildArtifact reports whether prefix is a known native binary
// format. Opacity is decided by magic, not by "contains a NUL somewhere": a
// text artifact that embeds \x00{{TOKEN}} must still be scanned (NUL alone is
// not a skip signal), and a Mach-O/ELF/PE with no NUL must still be skipped.
// Checking only a bounded prefix for NUL would still let the embedded-NUL
// bypass through, which is why magic wins over the NUL heuristic.
func isOpaqueNativeBuildArtifact(prefix []byte) bool {
	if len(prefix) >= 4 {
		// ELF
		if prefix[0] == 0x7f && prefix[1] == 'E' && prefix[2] == 'L' && prefix[3] == 'F' {
			return true
		}
		// Mach-O 32/64-bit, both endians, plus fat/universal.
		switch uint32(prefix[0])<<24 | uint32(prefix[1])<<16 | uint32(prefix[2])<<8 | uint32(prefix[3]) {
		case 0xfeedface, 0xcefaedfe, 0xfeedfacf, 0xcffaedfe, 0xcafebabe, 0xbebafeca:
			return true
		}
	}
	// PE / DOS stub
	if len(prefix) >= 2 && prefix[0] == 'M' && prefix[1] == 'Z' {
		return true
	}
	return false
}

// scanNativeBuildArtifactPlaceholders classifies opacity from a bounded prefix,
// then streams the remainder line-by-line looking for unresolved tokens. Known
// native binaries (Mach-O/ELF/PE) are skipped after the probe. Unknown magic is
// never skipped: it is scanned in chunks so an unsupported binary or a huge
// hook cannot force a whole-file ReadAll into memory, while a text artifact
// that merely lacks a known header still cannot hide a token past the probe.
// Findings retained are capped at nativeBuildUnresolvedPlaceholderFindingCap;
// total counts every hit so callers can report suppression honestly.
func scanNativeBuildArtifactPlaceholders(path string, relative string) (opaque bool, findings []nativeBuildUnresolvedPlaceholderFinding, total int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return false, nil, 0, err
	}
	defer file.Close()

	prefix := make([]byte, nativeBuildOpacityProbeBytes)
	n, err := file.Read(prefix)
	if err != nil && err != io.EOF {
		return false, nil, 0, err
	}
	prefix = prefix[:n]
	if isOpaqueNativeBuildArtifact(prefix) {
		return true, nil, 0, nil
	}

	reader := io.MultiReader(bytes.NewReader(prefix), file)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), projectFileReadLimit)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		for _, token := range nativeBuildUnresolvedTokensInLine(scanner.Text()) {
			if nativeBuildInstallTimePlaceholders[token] {
				continue
			}
			total++
			if len(findings) < nativeBuildUnresolvedPlaceholderFindingCap {
				findings = append(findings, nativeBuildUnresolvedPlaceholderFinding{
					path:  relative,
					line:  lineNumber,
					token: token,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return false, nil, 0, fmt.Errorf("%s: line exceeds %d bytes while scanning for unresolved placeholders (unrecognized binary or non-line-oriented artifact); refusing unbounded read", relative, projectFileReadLimit)
		}
		return false, nil, 0, err
	}
	return false, findings, total, nil
}

// nativeBuildUnresolvedTokensInLine returns every {{...}} span on the line.
// A leading $ does not exempt a token: GitHub Actions ${{ github.* }} forms live
// only under skipped skill trees, and "${{ARBITRARY}}" must not bypass the guard
// in generated artifacts.
func nativeBuildUnresolvedTokensInLine(line string) []string {
	var tokens []string
	remaining := line
	for {
		start := strings.Index(remaining, "{{")
		if start < 0 {
			return tokens
		}
		end := strings.Index(remaining[start+2:], "}}")
		if end < 0 {
			return tokens
		}
		token := remaining[start : start+2+end+2]
		tokens = append(tokens, token)
		remaining = remaining[start+len(token):]
	}
}

func nativeBuildTargetOutputDir(root string, targetName string) string {
	if targetName == "claude-code" {
		return filepath.Join(root, "plugins", "loaf")
	}
	return filepath.Join(root, "dist", targetName)
}

func runNativeBuildArtifactCheck(name string, args []string) error {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(strings.TrimSpace(stderr.String()) + "\n" + strings.TrimSpace(stdout.String()))
		if detail == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, detail)
	}
	return nil
}

func nativeBuildRelativePaths(root string, paths []string) []string {
	relative := make([]string, 0, len(paths))
	for _, path := range paths {
		relative = append(relative, nativeBuildRelativePath(root, path))
	}
	return relative
}

func nativeBuildRelativePath(root string, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}
