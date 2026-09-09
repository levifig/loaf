package cli

import (
	"path/filepath"
	"strings"
)

func defaultKBDirectories() []string {
	return []string{"docs/knowledge", "docs/architecture", "docs/decisions"}
}

func isDefaultKBDirectory(path string) bool {
	for _, dir := range defaultKBDirectories() {
		if filepath.Clean(path) == filepath.FromSlash(dir) {
			return true
		}
	}
	return false
}

func unreadableKBFile(gitRoot, path string, readErr error) knowledgeFile {
	rel, err := filepath.Rel(gitRoot, path)
	if err != nil {
		rel = path
	}
	return knowledgeFile{Path: path, RelativePath: filepath.ToSlash(rel), ReadError: readErr.Error()}
}

// Location establishes the document's role even when a retained ADR carries
// historical knowledge metadata. Architecture never inherits review machinery.
func isArchitectureDocument(relativePath string) bool {
	path := filepath.ToSlash(filepath.Clean(relativePath))
	return path == "docs/ARCHITECTURE.md" || strings.HasPrefix(path, "docs/architecture/") || strings.HasPrefix(path, "docs/decisions/")
}

// Check only basic structure, not YAML semantics, required headings, decision
// status, or architectural correctness. Those belong to the owning convention.
func architectureDocumentError(body []byte) string {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if lines[0] == "---" {
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
		}
		if end < 0 {
			return "Unterminated frontmatter"
		}
		text = strings.Join(lines[end+1:], "\n")
	}
	if strings.TrimSpace(text) == "" {
		return "Document body must not be empty"
	}
	return ""
}
