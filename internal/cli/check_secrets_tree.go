package cli

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

var secretTreeContentPatterns = []secretPattern{
	{name: "password assignment", re: regexp.MustCompile(`(?i)password\s*=\s*["'][^"']+`)},
	{name: "secret assignment", re: regexp.MustCompile(`(?i)secret\s*=\s*["'][^"']+`)},
	{name: "api_key assignment", re: regexp.MustCompile(`(?i)api_key\s*=\s*["'][^"']+`)},
	{name: "apikey assignment", re: regexp.MustCompile(`(?i)apikey\s*=\s*["'][^"']+`)},
	{name: "token assignment", re: regexp.MustCompile(`(?i)token\s*=\s*["'][^"']+`)},
	{name: "private_key assignment", re: regexp.MustCompile(`(?i)private_key\s*=\s*["'][^"']+`)},
	{name: "AWS_ACCESS_KEY_ID", re: regexp.MustCompile(`AWS_ACCESS_KEY_ID\s*=\s*["']?[A-Z0-9]{20}`)},
	{name: "AWS_SECRET_ACCESS_KEY", re: regexp.MustCompile(`AWS_SECRET_ACCESS_KEY\s*=\s*["']?[A-Za-z0-9/+=]{40}`)},
	{name: "GITHUB_TOKEN", re: regexp.MustCompile(`GITHUB_TOKEN\s*=\s*["']?gh[ps]_[A-Za-z0-9]{36}`)},
}

var secretTreeContentExcludeGlobs = []string{
	"*.example*",
	"*.md",
	"*.rst",
	"*test*",
	"*mock*",
	"*fixture*",
}

func (r Runner) runCheckTreeScan(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, false)
	if err != nil {
		return err
	}
	target := options.path
	if !options.hasPath {
		target = "."
	}
	result := scanSecretsTree(resolveCheckOperatorPath(runtimeRoot, target))
	return writeCheckOperatorResult(out, errOut, "secrets", result, options.jsonOutput)
}

func scanSecretsTree(root string) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Findings = append(result.Findings, fmt.Sprintf("unreadable path %s: %s", filepath.ToSlash(path), walkErr.Error()))
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if treeScanSkipDirs[name] {
				return fs.SkipDir
			}
			return nil
		}
		rel := path
		if trimmed, err := filepath.Rel(root, path); err == nil {
			rel = trimmed
		}
		display := filepath.ToSlash(rel)
		if isSecretTreeEnvFile(name) {
			result.Findings = append(result.Findings, ".env file: "+display)
		}
		if isSecretTreeKeyFile(name) {
			result.Findings = append(result.Findings, "private key file: "+display)
		}
		if secretTreeExcludeContent(name) {
			return nil
		}
		content, err := readRegularFile(path, projectFileReadLimit)
		if err != nil {
			result.Findings = append(result.Findings, fmt.Sprintf("unreadable file %s: %s", display, err.Error()))
			return nil
		}
		if strings.ContainsRune(string(content[:min(len(content), 512)]), 0) {
			return nil
		}
		body := string(content)
		for _, pattern := range secretTreeContentPatterns {
			if pattern.re.MatchString(body) {
				result.Findings = append(result.Findings, fmt.Sprintf("%s in %s", pattern.name, display))
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
	if len(result.Findings) > 0 {
		result.Passed = false
		result.Blocked = true
		result.Errors = append(result.Errors, fmt.Sprintf("Found %d potential issue(s)", len(result.Findings)))
	}
	return result
}

func isSecretTreeEnvFile(name string) bool {
	if name == ".env" || name == ".env.local" {
		return true
	}
	return strings.HasPrefix(name, ".env.") && strings.HasSuffix(name, ".local")
}

func isSecretTreeKeyFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".pem" || ext == ".key" {
		return true
	}
	return strings.HasPrefix(name, "id_rsa")
}

func secretTreeExcludeContent(name string) bool {
	for _, pattern := range secretTreeContentExcludeGlobs {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
