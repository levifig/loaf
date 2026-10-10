package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// mergeStrategies are the pull request landing methods a project can select
// with git.merge_strategy in .agents/loaf.json. The names match the
// `gh pr merge --squash|--merge|--rebase` flags. When the field is unset, the
// git-workflow skill defines the fallback; Loaf does not write a default.
var mergeStrategies = []string{"squash", "merge", "rebase"}

// configuredMergeStrategy returns the project's valid git.merge_strategy, or ""
// when it is unset, invalid, or unreadable. Callers phrase guidance neutrally
// for "" rather than assuming squash; `loaf config check` reports invalid values.
func configuredMergeStrategy(root string) string {
	body, err := os.ReadFile(filepath.Join(root, ".agents", "loaf.json"))
	if err != nil {
		return ""
	}
	var config struct {
		Git struct {
			MergeStrategy any `json:"merge_strategy"`
		} `json:"git"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		return ""
	}
	strategy, ok := config.Git.MergeStrategy.(string)
	if !ok || !slices.Contains(mergeStrategies, strategy) {
		return ""
	}
	return strategy
}

// validateGitConfig checks the optional git section of .agents/loaf.json.
func validateGitConfig(config map[string]any, errors *[]string) {
	raw, exists := config["git"]
	if !exists {
		return
	}
	section, ok := raw.(map[string]any)
	if !ok {
		*errors = append(*errors, "git must be an object")
		return
	}
	value, exists := section["merge_strategy"]
	if !exists {
		return
	}
	if strategy, ok := value.(string); !ok || !slices.Contains(mergeStrategies, strategy) {
		*errors = append(*errors, fmt.Sprintf("git.merge_strategy must be one of %s", strings.Join(mergeStrategies, ", ")))
	}
}
