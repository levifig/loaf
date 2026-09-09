package cli

import (
	"regexp"
	"strings"
)

var (
	sqlDropObjectRE = regexp.MustCompile(`(?i)\bDROP\s+(DATABASE|TABLE|SCHEMA)\b`)
	sqlTruncateRE   = regexp.MustCompile(`(?i)\bTRUNCATE\b`)
	sqlDeleteFromRE = regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`)
	sqlWhereRE      = regexp.MustCompile(`(?i)\bWHERE\b`)
	sqlDropColumnRE = regexp.MustCompile(`(?i)\bALTER\s+TABLE\b.*\bDROP\s+COLUMN\b`)
)

func runNativeValidateSQLSafety(context checkHookContext) checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	command := safetyInspectableCommand(checkContextCommand(context))
	if strings.TrimSpace(command) == "" {
		return result
	}
	if sqlDropObjectRE.MatchString(command) {
		return blockedCheckResult("DROP DATABASE/TABLE/SCHEMA detected. This is a destructive operation that cannot be undone. If intentional, ask the user to confirm before proceeding.")
	}
	if sqlTruncateRE.MatchString(command) {
		return blockedCheckResult("TRUNCATE detected. This deletes all rows and may not be reversible. If intentional, ask the user to confirm before proceeding.")
	}
	if sqlDeleteFromRE.MatchString(command) && !sqlWhereRE.MatchString(command) {
		result.Warnings = append(result.Warnings, "DELETE without WHERE clause detected. This will delete ALL rows in the table.")
	}
	if sqlDropColumnRE.MatchString(command) {
		result.Warnings = append(result.Warnings, "DROP COLUMN detected. This is a destructive migration. Ensure it's reversible.")
	}
	return result
}
