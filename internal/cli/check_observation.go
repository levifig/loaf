package cli

import (
	"context"
	"os"
	"strings"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

// recordCheckObservation is deliberately outside policy evaluation. A local
// state failure cannot turn an allowed tool into a blocked tool or vice versa.
func (r Runner) recordCheckObservation(options checkOptions, hookContext checkHookContext, result checkResult, payloadErr error, runtimeRoot string) {
	if !validCheckHooks[options.hook] {
		return
	}
	if !options.cursorOutput && hookContext.HookEventName == "" && checkContextToolName(hookContext) == "" && payloadErr == nil {
		return
	}
	root, err := project.ResolveRoot(runtimeRoot)
	if err != nil {
		return
	}
	store, err := state.OpenProjectStoreForWrite(context.Background(), root, state.PathResolver{StateHome: r.StateHome})
	if err != nil {
		return
	}
	defer store.Close()
	projectID, err := store.ProjectIDForRoot(context.Background(), root)
	if err != nil {
		return
	}
	decision := "allow"
	reason := "passed"
	if result.Blocked && options.advisory {
		decision = "warn"
		reason = "advisory"
	} else if result.Blocked {
		decision = "block"
		reason = "policy"
		if payloadErr != nil {
			reason = "invalid-payload"
		}
	} else if len(result.Warnings) > 0 || len(result.Errors) > 0 {
		decision = "warn"
		reason = "advisory"
	}
	harness, version := gatewayHarness(hookContext, options)
	_ = store.RecordHookGatewayObservation(context.Background(), projectID, state.HookGatewayObservation{
		HookID: options.hook, ToolName: gatewayToolName(checkContextToolName(hookContext)),
		Decision: decision, ReasonClass: reason, ObservedHarness: harness, HarnessVersion: version,
	})
}

func gatewayHarness(hookContext checkHookContext, options checkOptions) (string, string) {
	if options.cursorOutput || hookContext.HookEventName == "preToolUse" {
		return "cursor", firstKnown(os.Getenv("CURSOR_VERSION"))
	}
	if hookContext.HookEventName == "PreToolUse" {
		return "claude-code", firstKnown(os.Getenv("CLAUDE_CODE_VERSION"))
	}
	return observedHarnessFromEnvironment()
}

func gatewayToolName(name string) string {
	switch strings.TrimSpace(name) {
	case "Bash", "Shell", "Edit", "StrReplace", "Write", "Read", "Glob", "Grep", "WebFetch", "WebSearch", "Task":
		return strings.TrimSpace(name)
	default:
		return "unknown"
	}
}
