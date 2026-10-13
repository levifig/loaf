package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type cursorInstructionSpec struct {
	file, executable string
	arguments        []string
	post             bool
}

// Static guidance remains authored once under content/hooks/instructions.
// This closed adapter maps only the three instruction hooks Cursor ships.
var cursorInstructionSpecs = map[string]cursorInstructionSpec{
	"workflow-pre-merge":  {file: "pre-merge.md", executable: "gh", arguments: []string{"pr", "merge"}},
	"workflow-pre-push":   {file: "pre-push.md", executable: "git", arguments: []string{"push"}},
	"workflow-post-merge": {file: "post-merge.md", executable: "gh", arguments: []string{"pr", "merge"}, post: true},
}

func cursorInstructionPreviousEvent(target, event, id string) string {
	if target == "cursor" && event == "beforeShellExecution" && (id == "workflow-pre-merge" || id == "workflow-pre-push") {
		return "preToolUse"
	}
	return ""
}

func (r Runner) runCursorInstruction(spec cursorInstructionSpec, out io.Writer) error {
	message := ""
	var payload struct {
		Command   string `json:"command"`
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}
	body, err := io.ReadAll(io.LimitReader(firstReader(r.Stdin, os.Stdin), projectFileReadLimit+1))
	if err != nil || int64(len(body)) > projectFileReadLimit || json.Unmarshal(body, &payload) != nil {
		message = "[loaf] Cursor instruction guidance unavailable: unreadable or malformed hook payload"
	} else {
		command := payload.Command
		if spec.post {
			command = payload.ToolInput.Command
			if payload.ToolName != "Shell" {
				return writeJSON(out, map[string]string{})
			}
		}
		if strings.TrimSpace(command) == "" {
			message = "[loaf] Cursor instruction guidance unavailable: hook payload is missing command"
		} else if cursorInstructionCommandMatches(command, spec) {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				message = "[loaf] Cursor instruction guidance unavailable: " + homeErr.Error()
			} else {
				path := filepath.Join(home, ".cursor", "hooks", "instructions", spec.file)
				content, readErr := readRegularFileNoFollow(path, projectFileReadLimit)
				if readErr != nil {
					message = fmt.Sprintf("[loaf] Cursor instruction guidance unavailable at %s: %v", path, readErr)
				} else {
					message = strings.TrimSpace(string(content))
				}
			}
		}
	}
	if spec.post {
		output := map[string]string{}
		if message != "" {
			output["additional_context"] = message
		}
		return writeJSON(out, output)
	}
	output := map[string]string{"permission": "allow"}
	if message != "" {
		output["agent_message"] = message
	}
	return writeJSON(out, output)
}

// Match executable arguments rather than text mentions. The native Cursor
// matcher is a coarse filter; this guard keeps quoted examples and unrelated
// commands from receiving guidance, including in compound shell commands.
func cursorInstructionCommandMatches(command string, spec cursorInstructionSpec) bool {
	for _, segment := range splitSafetySegments(cursorInstructionWithoutComments(command), true) {
		words, valid := hookCommandTokens(segment)
		if !valid {
			continue
		}
		words = cursorInstructionExecutableWords(words)
		if len(words) < 2 || safetyCommandName(words[0]) != spec.executable {
			continue
		}
		i := 1
		for i < len(words) && strings.HasPrefix(words[i], "-") {
			option := words[i]
			i++
			if option == "--" {
				break
			}
			if spec.executable == "git" && (option == "-C" || option == "-c" || option == "--git-dir" || option == "--work-tree" || option == "--namespace" || option == "--config-env") || spec.executable == "gh" && (option == "-R" || option == "--repo" || option == "--hostname") {
				i++
			}
		}
		if len(words)-i < len(spec.arguments) {
			continue
		}
		match := true
		for j, argument := range spec.arguments {
			if words[i+j] != argument {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Shell comments begin only at an unquoted token boundary. Keep newlines so
// an actual command following a comment is still inspected independently.
func cursorInstructionWithoutComments(command string) string {
	var result strings.Builder
	quote := byte(0)
	boundary := true
	for i := 0; i < len(command); i++ {
		ch := command[i]
		if quote == '\'' {
			result.WriteByte(ch)
			if ch == '\'' {
				quote = 0
			}
			continue
		}
		if ch == '\\' && i+1 < len(command) {
			result.WriteByte(ch)
			i++
			result.WriteByte(command[i])
			if command[i] != '\n' {
				boundary = false
			}
			continue
		}
		if quote == '"' {
			result.WriteByte(ch)
			if ch == '"' {
				quote = 0
			}
			continue
		}
		if ch == '#' && boundary {
			for i < len(command) && command[i] != '\n' {
				i++
			}
			if i < len(command) {
				result.WriteByte('\n')
			}
			boundary = true
			continue
		}
		result.WriteByte(ch)
		if ch == '\'' || ch == '"' {
			quote = ch
			boundary = false
			continue
		}
		boundary = strings.ContainsRune(" \t\r\n;&|()<>", rune(ch))
	}
	return result.String()
}

// Enforcement's shared wrapper walk intentionally scans conservatively.
// Advisory guidance must distinguish command lookup from execution, including
// when the lookup is nested inside an env/sudo/builtin wrapper.
func cursorInstructionExecutableWords(words []string) []string {
	i := 0
	for i < len(words) && isSafetyEnvAssignment(words[i]) {
		i++
	}
	if i >= len(words) {
		return nil
	}
	name := safetyCommandName(words[i])
	i++
	switch name {
	case "sudo", "doas":
		for i < len(words) {
			option := words[i]
			if option == "--" {
				i++
				break
			}
			if option == "-u" || option == "-g" || option == "-p" || option == "--user" {
				i += 2
				continue
			}
			if strings.HasPrefix(option, "-") {
				i++
				continue
			}
			break
		}
	case "env":
		for i < len(words) && (strings.HasPrefix(words[i], "-") || isSafetyEnvAssignment(words[i])) {
			option := words[i]
			i++
			if option == "--" {
				break
			}
			if option == "-u" || option == "--unset" || option == "-C" || option == "--chdir" {
				i++
			}
		}
	case "command":
		for i < len(words) && strings.HasPrefix(words[i], "-") {
			option := words[i]
			i++
			if option == "--" {
				break
			}
			// -v and -V report command identity instead of running it. The only
			// execution flag is -p; unknown options cannot prove execution either.
			if option != "-p" {
				return nil
			}
		}
	case "nohup", "exec", "builtin", "then":
		for i < len(words) && strings.HasPrefix(words[i], "-") {
			i++
		}
	case "nice", "time":
		if i < len(words) && (words[i] == "-n" || words[i] == "--adjustment") {
			i += 2
		} else if i < len(words) && strings.HasPrefix(words[i], "-") {
			i++
		}
	default:
		return words[i-1:]
	}
	if i >= len(words) {
		return nil
	}
	return cursorInstructionExecutableWords(words[i:])
}
