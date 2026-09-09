package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	pythonDefRE         = regexp.MustCompile(`^def ([A-Za-z_][A-Za-z0-9_]*)`)
	testNamingFixtureRE = regexp.MustCompile(`(_perfect|_degraded|_chaos|_minimal|_empty|_invalid)$`)
	testNamingFileOK    = regexp.MustCompile(`^(test_.*\.py|.*_test\.py|conftest\.py)$`)
)

func (r Runner) runCheckTestNaming(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, false)
	if err != nil {
		return err
	}
	target := options.path
	if !options.hasPath {
		target = "."
	}
	resolved := resolveCheckOperatorPath(runtimeRoot, target)
	return writeCheckOperatorResult(out, errOut, "test-naming", checkPythonTestNaming(resolved), options.jsonOutput)
}

func checkPythonTestNaming(root string) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("unreadable path %s: %w", filepath.ToSlash(path), walkErr)
		}
		if entry.IsDir() {
			if treeScanSkipDirs[entry.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		rel := path
		if trimmed, err := filepath.Rel(root, path); err == nil {
			rel = trimmed
		}
		slash := filepath.ToSlash(rel)
		rulePath := testNamingRulePath(root, rel)
		name := entry.Name()
		if pathHasDir(rulePath, "tests") && strings.HasSuffix(name, ".py") && !testNamingFileOK.MatchString(name) {
			result.Errors = append(result.Errors, slash+" should be test_*.py or *_test.py")
		}
		if testNamingFileOK.MatchString(name) && name != "conftest.py" {
			result.Errors = append(result.Errors, pythonTestFunctionIssues(path, slash)...)
		}
		if pathHasDir(rulePath, "fixtures") {
			switch strings.ToLower(filepath.Ext(name)) {
			case ".json", ".yaml", ".yml":
				stem := strings.TrimSuffix(name, filepath.Ext(name))
				if !testNamingFixtureRE.MatchString(stem) {
					result.Errors = append(result.Errors, slash+" fixture should use a scenario suffix: _perfect, _degraded, _chaos, _minimal, _empty, _invalid")
				}
			}
		}
		return nil
	})
	if err != nil {
		result.Passed = false
		result.Blocked = true
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	if len(result.Errors) > 0 {
		result.Passed = false
		result.Blocked = true
	}
	return result
}

func testNamingRulePath(root, rel string) string {
	prefix := testNamingRelevantAncestorSuffix(filepath.Clean(root))
	if prefix == "" {
		return rel
	}
	return filepath.Join(prefix, rel)
}

// testNamingRelevantAncestorSuffix returns the path suffix from the nearest
// tests or fixtures ancestor through the normalized scan target. Cwd is not a
// project boundary and is never consulted.
func testNamingRelevantAncestorSuffix(root string) string {
	var parts []string
	current := root
	for {
		base := filepath.Base(current)
		if base == "." || base == string(filepath.Separator) || base == "" {
			break
		}
		parts = append([]string{base}, parts...)
		if base == "tests" || base == "fixtures" {
			return filepath.Join(parts...)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func pythonTestFunctionIssues(path, display string) []string {
	file, err := readRegularFile(path, projectFileReadLimit)
	if err != nil {
		return []string{"unreadable file " + display + ": " + err.Error()}
	}
	var issues []string
	scanner := bufio.NewScanner(bytes.NewReader(file))
	scanner.Buffer(make([]byte, 0, 64*1024), projectFileReadLimit)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		match := pythonDefRE.FindStringSubmatch(scanner.Text())
		if match == nil {
			continue
		}
		name := match[1]
		if strings.HasPrefix(name, "test_") || strings.HasPrefix(name, "_") || strings.HasPrefix(name, "setup") || strings.HasPrefix(name, "teardown") {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s:%d function '%s' doesn't start with 'test_'", display, lineNum, name))
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, "unreadable file "+display+": "+err.Error())
	}
	return issues
}

func pathHasDir(rel, name string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == name {
			return true
		}
	}
	return false
}
