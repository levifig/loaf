package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// councilValidatorContract holds the requirement literals declared in
// content/skills/orchestration/scripts/validate-council.py.
type councilValidatorContract struct {
	requiredFields   []string
	validStatuses    []string
	requiredSections []string
}

// councilDocument is the parsed shape of a council markdown file: its
// frontmatter fields under the council block, the number of composition
// entries, and the exact section headings in the body.
type councilDocument struct {
	fields      map[string]string
	composition int
	headings    map[string]bool
}

func TestCouncilTemplateSatisfiesValidatorContract(t *testing.T) {
	root := councilContractRepoRoot(t)
	contract := readCouncilValidatorContract(t, root)

	templatePath := filepath.Join(root, "content", "skills", "council", "templates", "council.md")
	doc := parseCouncilDocument(t, fencedCouncilBlock(t, readCouncilContractFile(t, templatePath)))

	for _, field := range contract.requiredFields {
		if _, ok := doc.fields[field]; !ok {
			t.Errorf("council template frontmatter is missing field %q required by validate-council.py", field)
		}
	}

	status := doc.fields["status"]
	if !slices.Contains(contract.validStatuses, status) {
		t.Errorf("council template status %q is not accepted by validate-council.py (valid: %v)", status, contract.validStatuses)
	}

	if doc.composition < 5 {
		t.Errorf("council template composition has %d entries; validate-council.py requires at least 5", doc.composition)
	}
	if doc.composition%2 == 0 {
		t.Errorf("council template composition has %d entries; validate-council.py requires an odd count", doc.composition)
	}

	for _, section := range contract.requiredSections {
		if !doc.headings[section] {
			t.Errorf("council template body is missing section %q required by validate-council.py", section)
		}
	}
}

func TestNewCouncilScaffoldSatisfiesValidatorContract(t *testing.T) {
	root := councilContractRepoRoot(t)
	contract := readCouncilValidatorContract(t, root)

	scriptPath := filepath.Join(root, "content", "skills", "orchestration", "scripts", "new-council.sh")
	doc := parseCouncilDocument(t, councilScaffoldHeredoc(t, readCouncilContractFile(t, scriptPath)))

	for _, field := range contract.requiredFields {
		if _, ok := doc.fields[field]; !ok {
			t.Errorf("new-council.sh scaffold frontmatter is missing field %q required by validate-council.py", field)
		}
	}

	status := doc.fields["status"]
	if !slices.Contains(contract.validStatuses, status) {
		t.Errorf("new-council.sh scaffold status %q is not accepted by validate-council.py (valid: %v)", status, contract.validStatuses)
	}

	// Composition entries are interpolated at runtime; the scaffold itself
	// enforces the >=5 and odd participant checks before writing the file.

	for _, section := range contract.requiredSections {
		if !doc.headings[section] {
			t.Errorf("new-council.sh scaffold body is missing section %q required by validate-council.py", section)
		}
	}
}

func councilContractRepoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func readCouncilContractFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(body)
}

func readCouncilValidatorContract(t *testing.T, root string) councilValidatorContract {
	t.Helper()
	source := readCouncilContractFile(t, filepath.Join(root, "content", "skills", "orchestration", "scripts", "validate-council.py"))
	return councilValidatorContract{
		requiredFields:   councilPythonListLiteral(t, source, "required_fields"),
		validStatuses:    councilPythonListLiteral(t, source, "valid_statuses"),
		requiredSections: councilPythonListLiteral(t, source, "required_sections"),
	}
}

func councilPythonListLiteral(t *testing.T, source, name string) []string {
	t.Helper()
	listPattern := regexp.MustCompile(name + `\s*=\s*\[([^\]]*)\]`)
	match := listPattern.FindStringSubmatch(source)
	if match == nil {
		t.Fatalf("validate-council.py no longer declares %s as a list literal", name)
	}
	var items []string
	for _, item := range regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(match[1], -1) {
		items = append(items, item[1])
	}
	if len(items) == 0 {
		t.Fatalf("no items parsed from %s in validate-council.py", name)
	}
	return items
}

// fencedCouncilBlock extracts the council file inside the template's fenced
// yaml code block.
func fencedCouncilBlock(t *testing.T, template string) string {
	t.Helper()
	const fence = "```yaml\n"
	start := strings.Index(template, fence)
	if start == -1 {
		t.Fatal("council template has no ```yaml fenced block")
	}
	block := template[start+len(fence):]
	end := strings.Index(block, "\n```")
	if end == -1 {
		t.Fatal("council template fenced block is not closed")
	}
	return block[:end]
}

// councilScaffoldHeredoc extracts the council file heredoc that new-council.sh
// writes.
func councilScaffoldHeredoc(t *testing.T, script string) string {
	t.Helper()
	const open = "<< EOF\n"
	start := strings.Index(script, open)
	if start == -1 {
		t.Fatal("new-council.sh has no << EOF heredoc")
	}
	block := script[start+len(open):]
	end := strings.Index(block, "\nEOF")
	if end == -1 {
		t.Fatal("new-council.sh heredoc is not closed")
	}
	return block[:end]
}

func parseCouncilDocument(t *testing.T, doc string) councilDocument {
	t.Helper()
	lines := strings.Split(doc, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		t.Fatalf("council document does not start with frontmatter delimiter, got %q", lines[0])
	}
	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontmatterEnd = i
			break
		}
	}
	if frontmatterEnd == -1 {
		t.Fatal("council document frontmatter is not closed")
	}

	parsed := councilDocument{fields: map[string]string{}, headings: map[string]bool{}}
	fieldPattern := regexp.MustCompile(`^  ([a-z_]+):(.*)$`)
	compositionEntry := regexp.MustCompile(`^    - `)
	inComposition := false
	for _, line := range lines[1:frontmatterEnd] {
		if inComposition && compositionEntry.MatchString(line) {
			parsed.composition++
			continue
		}
		inComposition = false
		if match := fieldPattern.FindStringSubmatch(line); match != nil {
			key := match[1]
			value := strings.SplitN(match[2], "#", 2)[0]
			parsed.fields[key] = strings.Trim(strings.TrimSpace(value), `"`)
			inComposition = key == "composition"
		}
	}

	for _, line := range lines[frontmatterEnd+1:] {
		heading := strings.TrimSpace(line)
		if strings.HasPrefix(heading, "##") {
			parsed.headings[heading] = true
		}
	}
	return parsed
}
