package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Advisory hooks do not print the ordinary check "passed" banner: most edits
// need no interruption. Neither payload, Git, nor cache failures block edits.
func (r Runner) runKbStalenessNudge(out io.Writer, runtimeRoot string, options checkOptions) error {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	payload := r.readKbNudgePayload()
	if payload != nil {
		result.Warnings = kbStalenessNudges(runtimeRoot, payload, time.Now())
	}
	if options.jsonOutput {
		return writeCheckJSON(out, options.hook, result, options.advisory)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(out, warning)
	}
	return nil
}

func (r Runner) readKbNudgePayload() map[string]any {
	reader := firstReader(r.Stdin, os.Stdin)
	if input := os.Getenv("TOOL_INPUT"); input != "" {
		reader = strings.NewReader(input)
	} else if file, ok := reader.(*os.File); ok {
		if info, err := file.Stat(); err != nil || info.Mode()&os.ModeCharDevice != 0 {
			return nil
		}
	}
	body, err := io.ReadAll(io.LimitReader(reader, projectFileReadLimit+1))
	if err != nil || len(body) > projectFileReadLimit {
		return nil
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	return payload
}

func kbNudgeFilePath(payload map[string]any) string {
	for _, key := range []string{"tool_input", "input"} {
		if input, ok := payload[key].(map[string]any); ok {
			if path, ok := input["file_path"].(string); ok && path != "" {
				return path
			}
		}
	}
	if tool, ok := payload["tool"].(map[string]any); ok {
		if input, ok := tool["input"].(map[string]any); ok {
			if path, ok := input["file_path"].(string); ok && path != "" {
				return path
			}
		}
	}
	path, _ := payload["file_path"].(string)
	return path
}

func kbStalenessNudges(runtimeRoot string, payload map[string]any, now time.Time) []string {
	warnings := []string{}
	filePath := kbNudgeFilePath(payload)
	if filePath == "" {
		return warnings
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	gitRoot := kbNudgeGitOutput(ctx, runtimeRoot, "rev-parse", "--show-toplevel")
	if gitRoot == "" {
		return warnings
	}
	if filepath.IsAbs(filePath) {
		relative, err := filepath.Rel(gitRoot, filePath)
		if err != nil {
			return warnings
		}
		filePath = relative
	}
	filePath = filepath.ToSlash(filepath.Clean(filePath))
	if filePath == ".." || strings.HasPrefix(filePath, "../") {
		return warnings
	}
	files := loadNativeKnowledgeFiles(gitRoot, loadNativeKbConfig(gitRoot), io.Discard, false)
	files = filterKnowledgeFilesCovering(files, filePath)
	results := stalenessResults(ctx, gitRoot, files)
	if ctx.Err() != nil {
		return warnings
	}
	envelope := normalizeJournalHookEnvelope(journalHookInput{Raw: payload}, runtimeRoot)
	scope := "conversation:" + envelope.Harness + ":" + envelope.SessionID
	if envelope.SessionID == "" {
		branch := kbNudgeGitOutput(ctx, gitRoot, "symbolic-ref", "--quiet", "--short", "HEAD")
		if branch == "" {
			branch = kbNudgeGitOutput(ctx, gitRoot, "rev-parse", "HEAD")
		}
		scope = "daily:" + branch + ":" + now.UTC().Format("2006-01-02")
	}
	cacheRoot := os.Getenv("XDG_CACHE_HOME")
	if !filepath.IsAbs(cacheRoot) {
		cacheRoot, _ = os.UserCacheDir()
	}
	for _, result := range results {
		if result.IsStale && claimKbNudge(cacheRoot, gitRoot, scope, result.File) {
			warnings = append(warnings, fmt.Sprintf("Knowledge file %q covers this code and may be stale. Read and verify its substantive content against current evidence before marking it reviewed with `loaf kb review`.", result.File))
		}
	}
	return warnings
}

func kbNudgeGitOutput(ctx context.Context, root string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	body, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// Empty, hashed markers are disposable notification cache, not journal or
// session state. Exclusive creation deduplicates concurrent hook processes and
// never follows or overwrites an existing marker. Cache failure permits a nudge.
func claimKbNudge(cacheRoot, projectRoot, scope, file string) bool {
	if cacheRoot == "" {
		return true
	}
	directory := filepath.Join(cacheRoot, "loaf", "kb-nudges")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return true
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	key, _ := json.Marshal([]string{projectRoot, scope, file})
	path := filepath.Join(directory, fmt.Sprintf("%x", sha256.Sum256(key)))
	marker, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return !os.IsExist(err)
	}
	_ = marker.Close()
	return true
}
