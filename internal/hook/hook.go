package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Input represents the common fields from Claude Code hook stdin JSON.
type Input struct {
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`

	// SessionStart-specific
	Source string `json:"source"`

	// SubagentStart-specific
	TaskPrompt string `json:"task_prompt"`

	// PreToolUse / PostToolUse
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`

	// Raw holds the full parsed JSON for any extra fields.
	Raw map[string]any `json:"-"`
}

// Output represents the JSON response to Claude Code hooks.
type Output struct {
	// AdditionalContext is injected into the session/subagent context.
	AdditionalContext string `json:"additionalContext,omitempty"`

	// Decision for permission hooks.
	Decision *Decision `json:"hookSpecificOutput,omitempty"`
}

// Decision represents a permission decision.
type Decision struct {
	Behavior string `json:"behavior,omitempty"` // "allow", "deny", "ask"
	Reason   string `json:"reason,omitempty"`
}

// ReadInput reads and parses hook input from stdin.
func ReadInput() (*Input, error) {
	return ReadInputFrom(os.Stdin)
}

// ReadInputFrom reads and parses hook input from the given reader.
func ReadInputFrom(r io.Reader) (*Input, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}

	if len(data) == 0 {
		return &Input{}, nil
	}

	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	// Also parse into raw map for extra fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err == nil {
		input.Raw = raw
	}

	return &input, nil
}

// WriteOutput writes hook output as JSON to stdout.
func WriteOutput(out Output) error {
	return WriteOutputTo(os.Stdout, out)
}

// WriteOutputTo writes hook output as JSON to the given writer.
func WriteOutputTo(w io.Writer, out Output) error {
	enc := json.NewEncoder(w)
	return enc.Encode(out)
}
