package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Cursor permission hooks require a native decision, not just parseable JSON.
// Keep the CLI result fields for diagnostics and existing operator consumers.
type cursorCheckOutput struct {
	checkJSONOutput
	Permission   string `json:"permission"`
	UserMessage  string `json:"user_message,omitempty"`
	AgentMessage string `json:"agent_message,omitempty"`
}

func (r Runner) runCursorCheck(args []string, out io.Writer, runtimeRoot string) error {
	var nativeArgs []string
	for _, arg := range args {
		if arg != "--cursor-hook" {
			nativeArgs = append(nativeArgs, arg)
		}
	}
	options, err := parseCheckArgs(nativeArgs)
	if err != nil {
		return writeCursorCheck(out, checkJSONOutput{Errors: []string{err.Error()}, Blocked: true}, false)
	}
	// All enforcement payloads must be inspectable. A broken pipe or malformed
	// payload must never be mistaken for an empty, benign tool call.
	context, err := r.parseCheckContext(true)
	if err == nil && checkContextToolName(context) == "" {
		err = fmt.Errorf("Cursor hook payload is missing tool_name")
	}
	if err == nil {
		switch cursorCheckToolName(checkContextToolName(context)) {
		case "Bash":
			if strings.TrimSpace(checkContextCommand(context)) == "" {
				err = fmt.Errorf("Cursor Shell payload is missing command")
			}
		case "Edit", "Write":
			if strings.TrimSpace(checkContextFilePath(context)) == "" {
				err = fmt.Errorf("Cursor file-write payload is missing file_path")
			}
		}
	}
	if err != nil {
		return writeCursorCheck(out, checkJSONOutput{Hook: options.hook, Errors: []string{err.Error()}, Blocked: true}, options.advisory)
	}
	// User-level Cursor hooks run in the configuration directory; the native
	// payload identifies the repository whose command is being inspected.
	if context.Cwd != "" {
		info, cwdErr := os.Stat(context.Cwd)
		if !filepath.IsAbs(context.Cwd) || cwdErr != nil || !info.IsDir() {
			return writeCursorCheck(out, checkJSONOutput{Hook: options.hook, Blocked: true, Errors: []string{"Cursor hook cwd must be an accessible absolute directory"}}, options.advisory)
		}
		runtimeRoot = context.Cwd
	}
	context.ToolName = cursorCheckToolName(context.ToolName)
	context.Tool.Name = cursorCheckToolName(context.Tool.Name)
	// The native envelope has been consumed; evaluate the neutral check once.
	context.HookEventName = ""
	input, err := json.Marshal(context)
	if err != nil {
		return err
	}
	r.Stdin = bytes.NewReader(input)
	if !options.jsonOutput {
		nativeArgs = append(nativeArgs, "--json")
	}
	var buffer bytes.Buffer
	err = r.runCheck(nativeArgs, &buffer, runtimeRoot)
	var result checkJSONOutput
	if decodeErr := json.Unmarshal(buffer.Bytes(), &result); decodeErr != nil {
		message := "Loaf check returned an unreadable result"
		if err != nil {
			message = err.Error()
		}
		result = checkJSONOutput{Hook: options.hook, Blocked: true, Errors: []string{message}}
	}
	return writeCursorCheck(out, result, options.advisory)
}

func cursorCheckToolName(name string) string {
	switch name {
	case "Shell":
		return "Bash"
	case "StrReplace":
		return "Edit"
	default:
		return name
	}
}

func writeCursorCheck(out io.Writer, result checkJSONOutput, advisory bool) error {
	deny := result.Blocked && !advisory
	permission := "allow"
	result.ExitCode = 0
	result.Advisory = advisory
	if deny {
		permission = "deny"
		result.ExitCode = 2
	}
	message := strings.Join(append(append([]string{}, result.Errors...), result.Warnings...), "\n")
	output := cursorCheckOutput{checkJSONOutput: result, Permission: permission, AgentMessage: message}
	if deny {
		output.UserMessage = message
	}
	if err := writeJSON(out, output); err != nil {
		return err
	}
	if deny {
		return ExitError{Code: 2}
	}
	return nil
}
