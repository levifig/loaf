package cli

import (
	"path/filepath"
	"strings"
	"unicode"
)

var safetyInertSearchCommands = map[string]bool{
	"egrep": true, "fgrep": true, "grep": true, "rg": true, "ripgrep": true,
}

var safetyShellCommands = map[string]bool{
	"bash": true, "dash": true, "fish": true, "ksh": true, "sh": true, "zsh": true,
}

var safetyExecutionOptions = map[string]bool{
	"-exec": true, "-execdir": true, "-ok": true, "-okdir": true, "--pre": true,
}

// safetyInspectableCommand returns the parts of a hook command that should be
// scanned for destructive execution. A clause is omitted only when every
// stage is a proven-inert search form: rg, grep (and aliases), or git grep
// with only allowlisted search options, uniformly quoted literal pattern and
// path arguments, no substitutions, no mixed quote fragments, and no
// execution options. Unknown git-grep options, including
// -O/--open-files-in-pager, keep the stage visible. Every other program is
// scanned, including awk, find, echo, sed, and shells whose inner script is
// not itself an inert search.
func safetyInspectableCommand(command string) string {
	var kept []string
	for _, clause := range splitSafetySegments(command, false) {
		if safetyClauseIsInertSearch(clause) {
			continue
		}
		kept = append(kept, clause)
	}
	// Keep independent clauses on separate lines so downstream .* patterns
	// cannot associate one command with another command's options.
	return strings.Join(kept, "\n")
}

func safetyClauseIsInertSearch(clause string) bool {
	stages := splitSafetySegments(clause, true)
	if len(stages) == 0 {
		return true
	}
	for _, stage := range stages {
		if script, ok := safetyNestedShellScript(stage); ok {
			if !safetyStageArgumentsAreLiteral(stage) || !safetyClauseIsInertSearch(script) {
				return false
			}
			continue
		}
		if !safetyStageIsInertSearch(stage) {
			return false
		}
	}
	return true
}

func safetyStageIsInertSearch(stage string) bool {
	return safetyStageArgumentsAreLiteral(stage) && safetyStageIsInertSearchCommand(stage)
}

func safetyStageArgumentsAreLiteral(stage string) bool {
	tokens, ok := hookCommandTokensForOS(stage, "")
	if !ok {
		return false
	}
	for _, token := range tokens {
		if token.mixed || safetyTokenHasSubstitution(token) || safetyTokenIsUnsupportedSyntax(token) || safetyTokenIsExecutionOption(token.value) {
			return false
		}
	}
	return true
}

func safetyTokenHasSubstitution(token hookCommandToken) bool {
	if !token.mixed && token.quote == hookTokenSingleQuoted {
		return false
	}
	return strings.Contains(token.value, "$(") || strings.Contains(token.value, "`") || strings.Contains(token.value, "<(") || strings.Contains(token.value, ">(")
}

func safetyTokenIsExecutionOption(value string) bool {
	lower := strings.ToLower(value)
	if safetyExecutionOptions[lower] {
		return true
	}
	for option := range safetyExecutionOptions {
		if strings.HasPrefix(lower, option+"=") {
			return true
		}
	}
	return false
}

func safetyTokenIsUnsupportedSyntax(token hookCommandToken) bool {
	if token.quote != hookTokenUnquoted {
		return false
	}
	switch token.value {
	case "&", "|", ";", ">", ">>", ">|", ">&", "<", "<<", "<<<", "<&", "|&", "(", ")":
		return true
	}
	return strings.Contains(token.value, "&")
}

func safetyStageIsInertSearchCommand(stage string) bool {
	words := safetySkipWrappers(safetyCommandWords(stage))
	if len(words) == 0 {
		return true
	}
	name := safetyCommandName(words[0])
	if safetyInertSearchCommands[name] {
		return true
	}
	if name != "git" {
		return false
	}
	for i := 1; i < len(words); {
		word := words[i]
		switch {
		case word == "-C" || word == "--git-dir" || word == "--work-tree":
			if i+1 >= len(words) {
				return false
			}
			i += 2
		case strings.HasPrefix(word, "--git-dir=") || strings.HasPrefix(word, "--work-tree="):
			i++
		case word == "--":
			i++
		case strings.HasPrefix(word, "-"):
			return false
		default:
			if strings.ToLower(word) != "grep" {
				return false
			}
			return safetyGitGrepOptionsAreInert(words[i+1:])
		}
	}
	return false
}

// safetyGitGrepInertShortOptions is the proven-inert git-grep short-option
// set. The value is the number of following arguments the flag consumes.
// -O/--open-files-in-pager is omitted because git runs the supplied pager.
var safetyGitGrepInertShortOptions = map[byte]int{
	'A': 1, 'B': 1, 'C': 1, 'E': 0, 'F': 0, 'G': 0, 'H': 0, 'I': 0, 'L': 0,
	'P': 0, 'W': 0, 'a': 0, 'c': 0, 'e': 1, 'f': 1, 'h': 0, 'i': 0, 'l': 0,
	'm': 1, 'n': 0, 'o': 0, 'p': 0, 'q': 0, 'r': 0, 'v': 0, 'w': 0, 'z': 0,
}

// safetyGitGrepInertLongOptions is the proven-inert git-grep long-option set.
// --textconv and --open-files-in-pager are omitted because they can execute
// a configured filter or pager. The value is the number of following arguments.
var safetyGitGrepInertLongOptions = map[string]int{
	"after-context": 1, "all-match": 0, "and": 0, "basic-regexp": 0, "before-context": 1,
	"break": 0, "cached": 0, "color": 0, "column": 0, "context": 1, "count": 0,
	"exclude-standard": 0, "extended-regexp": 0, "files-with-matches": 0,
	"files-without-match": 0, "fixed-strings": 0, "full-name": 0, "function-context": 0,
	"heading": 0, "ignore-case": 0, "invert-match": 0, "line-number": 0, "max-count": 1,
	"max-depth": 1, "name-only": 0, "no-color": 0, "no-exclude-standard": 0,
	"no-index": 0, "no-recursive": 0, "no-textconv": 0, "not": 0, "null": 0,
	"only-matching": 0, "or": 0, "parent-basename": 1, "perl-regexp": 0, "quiet": 0,
	"recurse-submodules": 0, "recursive": 0, "show-function": 0, "text": 0,
	"threads": 1, "untracked": 0, "word-regexp": 0,
}

func safetyGitGrepOptionsAreInert(words []string) bool {
	for i := 0; i < len(words); {
		word := words[i]
		if word == "--" {
			return true
		}
		if word == "-" || !strings.HasPrefix(word, "-") {
			i++
			continue
		}
		if strings.HasPrefix(word, "--") {
			name, _, attached := strings.Cut(strings.TrimPrefix(word, "--"), "=")
			takes, ok := safetyGitGrepInertLongOptions[strings.ToLower(name)]
			if !ok {
				return false
			}
			if attached || takes == 0 {
				i++
				continue
			}
			if i+1 >= len(words) {
				return false
			}
			i += 2
			continue
		}
		flags := word[1:]
		consumedValue := false
		for j := 0; j < len(flags); {
			takes, ok := safetyGitGrepInertShortOptions[flags[j]]
			if !ok {
				return false
			}
			j++
			if takes == 0 {
				continue
			}
			if j < len(flags) {
				break
			}
			if i+1 >= len(words) {
				return false
			}
			consumedValue = true
			break
		}
		if consumedValue {
			i += 2
			continue
		}
		i++
	}
	return true
}

func safetyNestedShellScript(stage string) (string, bool) {
	words := safetySkipWrappers(safetyCommandWords(stage))
	if len(words) == 0 || !safetyShellCommands[safetyCommandName(words[0])] {
		return "", false
	}
	for i := 1; i < len(words); i++ {
		word := words[i]
		if word == "-c" || word == "--command" {
			if i+1 < len(words) {
				return words[i+1], true
			}
			return "", false
		}
		if strings.HasPrefix(word, "--") || word == "--" {
			continue
		}
		if strings.HasPrefix(word, "-") && strings.Contains(strings.TrimPrefix(word, "-"), "c") {
			if i+1 < len(words) {
				return words[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

func safetyCommandWords(stage string) []string {
	tokens, ok := hookCommandTokens(stage)
	if ok {
		return tokens
	}
	return strings.Fields(stage)
}

func safetySkipWrappers(words []string) []string {
	i := 0
	for i < len(words) && isSafetyEnvAssignment(words[i]) {
		i++
	}
	if i >= len(words) {
		return nil
	}
	switch safetyCommandName(words[i]) {
	case "sudo", "doas":
		i++
		for i < len(words) {
			word := words[i]
			if word == "--" {
				i++
				break
			}
			if word == "-u" || word == "-g" || word == "-p" || word == "--user" {
				i += 2
				continue
			}
			if strings.HasPrefix(word, "-") {
				i++
				continue
			}
			break
		}
		return safetySkipWrappers(words[i:])
	case "env":
		i++
		for i < len(words) && (strings.HasPrefix(words[i], "-") || isSafetyEnvAssignment(words[i])) {
			if words[i] == "-u" || words[i] == "--unset" {
				i += 2
				continue
			}
			i++
		}
		return safetySkipWrappers(words[i:])
	case "command", "nohup", "exec", "builtin", "then":
		i++
		for i < len(words) && strings.HasPrefix(words[i], "-") {
			i++
		}
		return safetySkipWrappers(words[i:])
	case "nice", "time":
		i++
		if i < len(words) && (words[i] == "-n" || words[i] == "--adjustment") {
			i += 2
		} else if i < len(words) && strings.HasPrefix(words[i], "-") {
			i++
		}
		return safetySkipWrappers(words[i:])
	default:
		return words[i:]
	}
}

func safetyCommandName(word string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(word)), ".exe")
}

func isSafetyEnvAssignment(word string) bool {
	eq := strings.IndexByte(word, '=')
	if eq <= 0 || strings.HasPrefix(word, "-") {
		return false
	}
	for _, r := range word[:eq] {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}

func splitSafetySegments(command string, splitPipes bool) []string {
	var segments []string
	var current strings.Builder
	quote := rune(0)
	escaped := false
	flush := func() {
		segment := strings.TrimSpace(current.String())
		if segment != "" {
			segments = append(segments, segment)
		}
		current.Reset()
	}
	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if quote == '\'' {
			current.WriteRune(ch)
			if ch == '\'' {
				quote = 0
			}
			continue
		}
		if ch == '\\' && !escaped && i+1 < len(runes) {
			if runes[i+1] == '\n' {
				// A shell continuation joins words within one command; it is
				// not a boundary between independent commands.
				i++
				continue
			}
			if quote == 0 {
				current.WriteRune(ch)
				current.WriteRune(runes[i+1])
				i++
				continue
			}
		}
		if quote == '"' {
			current.WriteRune(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			current.WriteRune(ch)
			continue
		}
		if ch == '\n' || ch == ';' {
			flush()
			continue
		}
		if ch == '&' && i+1 < len(runes) && runes[i+1] == '&' {
			flush()
			i++
			continue
		}
		if ch == '|' && i+1 < len(runes) && runes[i+1] == '|' {
			flush()
			i++
			continue
		}
		if splitPipes && ch == '|' {
			flush()
			continue
		}
		current.WriteRune(ch)
	}
	flush()
	return segments
}
