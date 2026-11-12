package cli

import (
	"fmt"
	"slices"
	"strings"
)

// trackerStandingPermissions are the tracker writes a project can let agents
// make without drafting them first, recorded as tracker.standing_permission in
// .agents/loaf.json. Absent or empty means every one is drafted for the
// operator. The skills own the rules; Loaf only validates the shape.
var trackerStandingPermissions = []string{"comments", "status", "edits"}

// validateTrackerConfig checks the optional tracker section of .agents/loaf.json.
func validateTrackerConfig(config map[string]any, errors *[]string) {
	raw, exists := config["tracker"]
	if !exists {
		return
	}
	section, ok := raw.(map[string]any)
	if !ok {
		*errors = append(*errors, "tracker must be an object")
		return
	}
	value, exists := section["standing_permission"]
	if !exists {
		return
	}
	if !validStandingPermission(value) {
		*errors = append(*errors, fmt.Sprintf("tracker.standing_permission must be a list of distinct values from %s", strings.Join(trackerStandingPermissions, ", ")))
	}
}

func validStandingPermission(value any) bool {
	list, ok := value.([]any)
	if !ok {
		return false
	}
	seen := map[string]bool{}
	for _, item := range list {
		permission, ok := item.(string)
		if !ok || !slices.Contains(trackerStandingPermissions, permission) || seen[permission] {
			return false
		}
		seen[permission] = true
	}
	return true
}
