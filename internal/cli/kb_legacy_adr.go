package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

type kbValidateOptions struct {
	legacyADR  bool
	jsonOutput bool
	files      []string
}

func parseKbValidateArgs(args []string) (kbValidateOptions, error) {
	var options kbValidateOptions
	positionalOnly := false
	for _, arg := range args {
		if positionalOnly {
			options.files = append(options.files, arg)
			continue
		}
		switch arg {
		case "--legacy-adr":
			options.legacyADR = true
		case "--json":
			options.jsonOutput = true
		case "--":
			positionalOnly = true
		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown option %q", arg)
			}
			options.files = append(options.files, arg)
		}
	}
	if options.legacyADR && len(options.files) == 0 {
		return options, fmt.Errorf("--legacy-adr requires at least one file")
	}
	if !options.legacyADR && len(options.files) > 0 {
		return options, fmt.Errorf("explicit files require --legacy-adr")
	}
	return options, nil
}

var (
	legacyADRIDPattern       = regexp.MustCompile(`^ADR-\d{3}$`)
	legacyADRFilenamePattern = regexp.MustCompile(`^(ADR-\d{3})-[a-z0-9][a-z0-9-]*\.md$`)
	legacyADRDatePattern     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	legacyADRFieldPattern    = regexp.MustCompile(`^([a-z_]+):\s*(.*?)\s*(?:#.*)?$`)
	legacyADRItemPattern     = regexp.MustCompile(`^\s+-\s+(.*?)\s*(?:#.*)?$`)
)

func validateLegacyADRFiles(root string, paths []string) []kbValidationResult {
	results := make([]kbValidationResult, 0, len(paths))
	for _, path := range paths {
		result := kbValidationResult{File: filepath.ToSlash(path), Errors: []kbValidationIssue{}, Warnings: []kbValidationIssue{}}
		absPath := path
		if !filepath.IsAbs(path) {
			absPath = filepath.Join(root, path)
		}
		content, err := readRegularFile(absPath, projectFileReadLimit)
		if err != nil {
			result.Errors = append(result.Errors, kbValidationIssue{Field: "read", Message: err.Error()})
		} else if !utf8.Valid(content) {
			result.Errors = append(result.Errors, kbValidationIssue{Field: "read", Message: "file is not valid UTF-8"})
		} else {
			result.Errors = validateLegacyADR(path, strings.ReplaceAll(string(content), "\r\n", "\n"))
		}
		results = append(results, result)
	}
	return results
}

// This deliberately preserves the old validator's limited frontmatter subset
// and date-shape checks. It is not a YAML parser or a general ADR schema.
func legacyADRFrontmatter(content string) (map[string]string, []kbValidationIssue) {
	values := make(map[string]string)
	errors := []kbValidationIssue{}
	if !strings.HasPrefix(content, "---\n") {
		return values, append(errors, kbValidationIssue{Field: "frontmatter", Message: "missing YAML frontmatter"})
	}
	end := strings.Index(content[4:], "\n---\n")
	if end < 0 {
		return values, append(errors, kbValidationIssue{Field: "frontmatter", Message: "unterminated YAML frontmatter"})
	}
	sequenceKey := ""
	for _, line := range strings.Split(content[4:4+end], "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if match := legacyADRFieldPattern.FindStringSubmatch(line); match != nil {
			key, value := match[1], strings.Trim(strings.TrimSpace(match[2]), `"'`)
			values[key] = value
			sequenceKey = ""
			if value == "" {
				sequenceKey = key
			}
			continue
		}
		if match := legacyADRItemPattern.FindStringSubmatch(line); match != nil && sequenceKey != "" {
			value := strings.Trim(strings.TrimSpace(match[1]), `"'`)
			if value != "" {
				if values[sequenceKey] != "" {
					values[sequenceKey] += ", "
				}
				values[sequenceKey] += value
			}
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			errors = append(errors, kbValidationIssue{Field: "frontmatter", Message: "unsupported frontmatter line: " + line})
		}
	}
	return values, errors
}

func validateLegacyADR(path, content string) []kbValidationIssue {
	values, errors := legacyADRFrontmatter(content)
	add := func(field, message string) {
		errors = append(errors, kbValidationIssue{Field: field, Message: message})
	}
	filename := legacyADRFilenamePattern.FindStringSubmatch(filepath.Base(path))
	if filename == nil {
		add("filename", "filename must match ADR-NNN-lowercase-slug.md")
	}
	id, title, status := values["id"], values["title"], values["status"]
	if !legacyADRIDPattern.MatchString(id) {
		add("id", "frontmatter id must match ADR-NNN")
	} else if filename != nil && filename[1] != id {
		add("id", "frontmatter id must match filename id")
	}
	if title == "" {
		add("title", "frontmatter title is required")
	}
	if id != "" && title != "" && !strings.Contains(content, "# "+id+": "+title) {
		add("heading", "H1 must match frontmatter id and title")
	}
	switch status {
	case "Proposed", "Accepted", "Rejected", "Deprecated", "Superseded":
	default:
		add("status", "status must be one of: Accepted, Deprecated, Proposed, Rejected, Superseded")
	}
	for _, field := range []string{"date", "revised", "accepted_date", "rejected_date", "deprecated_date"} {
		if value := values[field]; value != "" && !legacyADRDatePattern.MatchString(value) {
			add(field, field+" must use YYYY-MM-DD")
		}
	}
	if _, ok := values["date"]; !ok {
		add("date", "frontmatter date is required")
	}
	hasSection := func(heading string) bool {
		return regexp.MustCompile(`(?m)^##\s+` + regexp.QuoteMeta(heading) + `\s*$`).MatchString(content)
	}
	for _, section := range []string{"Context", "Decision", "Consequences", "Alternatives Considered"} {
		if !hasSection(section) {
			add("sections", "missing section: ## "+section)
		}
	}
	if status == "Rejected" || status == "Deprecated" {
		field := strings.ToLower(status) + "_date"
		if values[field] == "" {
			add(field, status+" status requires "+field)
		}
	}
	if status == "Superseded" && !strings.Contains(values["superseded_by"], "ADR-") {
		add("superseded_by", "Superseded status requires superseded_by ADR reference(s)")
	}
	if (status == "Rejected" || status == "Deprecated" || status == "Superseded") && !hasSection(status) {
		add("sections", status+" status requires ## "+status)
	}
	if values["revised"] != "" && !hasSection("Revisions") {
		add("sections", "revised frontmatter requires ## Revisions")
	}
	return errors
}
