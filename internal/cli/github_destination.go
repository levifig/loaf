package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/levifig/loaf/internal/project"
)

// This is routing from the current project, never an observation of GitHub.
// A configured account alone does not elect a tracker.
type githubDestinationStatus struct {
	Repository       string   `json:"repository"`
	ProjectID        string   `json:"project_id"`
	LiveVerification string   `json:"live_verification"`
	Errors           []string `json:"errors,omitempty"`
}

var githubDestinationRepositoryRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}$`)

// Validate only a bounded opaque token shape. Object type and usability belong
// to the agent's live exact-node read; GitHub IDs have multiple formats.
var githubDestinationProjectIDRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+/=-]{7,255}$`)

func configuredGitHubDestination(config map[string]any) *githubDestinationStatus {
	integrations, _ := config["integrations"].(map[string]any)
	github, _ := integrations["github"].(map[string]any)
	selected, _ := github["tracker"].(bool)
	if !selected {
		return nil
	}
	result := &githubDestinationStatus{LiveVerification: "not-performed"}
	repository, _ := github["repository"].(string)
	_, name, _ := strings.Cut(repository, "/")
	if !githubDestinationRepositoryRE.MatchString(repository) || name == "." || name == ".." {
		result.Errors = append(result.Errors, "integrations.github.repository must be an explicit owner/repository when integrations.github.tracker is true")
	} else {
		result.Repository = repository
	}
	projectID, _ := github["project_id"].(string)
	if !githubDestinationProjectIDRE.MatchString(projectID) {
		result.Errors = append(result.Errors, "integrations.github.project_id must be an opaque Projects v2 node ID, not a board title, URL or number, when integrations.github.tracker is true")
	} else {
		result.ProjectID = projectID
	}
	return result
}

func readProjectGitHubDestination(start string) (*githubDestinationStatus, error) {
	root, err := project.ResolveRoot(start)
	if err != nil {
		return nil, err
	}
	body, err := readRegularFile(filepath.Join(root.Path(), ".agents", "loaf.json"), projectFileReadLimit)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read project .agents/loaf.json: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var config map[string]any
	if err = decoder.Decode(&config); err != nil || config == nil {
		return nil, fmt.Errorf("project .agents/loaf.json must contain a valid JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("project .agents/loaf.json must contain one JSON object without trailing content")
	}
	integrations, _ := config["integrations"].(map[string]any)
	github, _ := integrations["github"].(map[string]any)
	if selected, exists := github["tracker"]; exists {
		if _, ok := selected.(bool); !ok {
			return nil, fmt.Errorf("integrations.github.tracker must be a boolean")
		}
	}
	return configuredGitHubDestination(config), nil
}

func writeGitHubDestinationText(out io.Writer, destination *githubDestinationStatus) {
	if destination == nil {
		return
	}
	repository := destination.Repository
	if repository == "" {
		repository = "missing or malformed"
	}
	projectID := destination.ProjectID
	if projectID == "" {
		projectID = "missing or malformed"
	}
	fmt.Fprintf(out, "\n%s\n  Repository: %s\n  Board ID: %s\n  Live board verification: not performed (local configuration only).\n", ansiBold("GitHub destination (project):"), repository, projectID)
	for _, finding := range destination.Errors {
		fmt.Fprintf(out, "  %s %s\n", ansiYellow("!"), finding)
	}
}
