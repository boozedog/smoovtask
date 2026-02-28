package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/boozedog/smoovtask/internal/hook"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook <event-type>",
	Short: "Handle Claude Code hook events",
	Args:  cobra.ExactArgs(1),
	RunE:  runHook,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

func runHook(_ *cobra.Command, args []string) error {
	eventType := args[0]

	switch eventType {
	case "opencode-event":
		// Read the opencode event from stdin
		eventData, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read event data: %w", err)
		}
		var event map[string]any
		if err := json.Unmarshal(eventData, &event); err != nil {
			return nil // Skip invalid events
		}
		// Create input
		input := &hook.Input{Source: "opencode"}
		// Extract sessionID from event properties
		if props, ok := event["properties"].(map[string]any); ok {
			if sid, ok := props["sessionID"].(string); ok {
				input.SessionID = sid
			} else if info, ok := props["info"].(map[string]any); ok {
				if sid, ok := info["sessionID"].(string); ok {
					input.SessionID = sid
				}
			}
		}
		// Process based on event type
		eventType, _ := event["type"].(string)
		switch eventType {
		case "session.created":
			props, _ := event["properties"].(map[string]any)
			info, _ := props["info"].(map[string]any)
			sessionID, _ := info["id"].(string)
			cwd, _ := info["directory"].(string)
			if sessionID == "" {
				return nil
			}
			input := &hook.Input{
				Source:    "opencode",
				SessionID: sessionID,
				CWD:       cwd,
			}
			out, err := hook.HandleSessionStart(input)
			if err != nil {
				return err
			}
			return hook.WriteOutput(*out)
		case "tool.execute.before":
			out, err := hook.HandlePreTool(input)
			if err != nil {
				return err
			}
			if out.AdditionalContext != "" {
				return hook.WriteOutput(out)
			}
			return nil
		case "tool.execute.after":
			return hook.HandlePostTool(input)
		case "session.idle":
			return hook.HandleTeammateIdle(input)
		case "permission.asked":
			out, err := hook.HandlePermissionRequest(input)
			if err != nil {
				return err
			}
			if out.Decision != nil {
				return hook.WriteOutput(out)
			}
			return nil
		case "session.deleted":
			return hook.HandleSessionEnd(input)
		default:
			return nil
		}

	case "session-start":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		out, err := hook.HandleSessionStart(input)
		if err != nil {
			return err
		}
		if source == "claude" {
			fmt.Print(out.AdditionalContext)
			return nil
		}
		return hook.WriteOutput(*out)

	case "subagent-start":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		out, err := hook.HandleSubagentStart(input)
		if err != nil {
			return err
		}
		if out.AdditionalContext != "" {
			return hook.WriteOutput(out)
		}
		return nil

	case "pre-tool":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		out, err := hook.HandlePreTool(input)
		if err != nil {
			return err
		}
		if out.AdditionalContext != "" {
			return hook.WriteOutput(out)
		}
		return nil

	case "post-tool":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		return hook.HandlePostTool(input)

	case "stop":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		return hook.HandleStop(input)

	case "subagent-stop":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		return hook.HandleSubagentStop(input)

	case "task-completed":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		return hook.HandleTaskCompleted(input)

	case "teammate-idle":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		return hook.HandleTeammateIdle(input)

	case "permission-request":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		out, err := hook.HandlePermissionRequest(input)
		if err != nil {
			return err
		}
		if out.Decision != nil {
			return hook.WriteOutput(out)
		}
		return nil

	case "session-end":
		input, err := hook.ReadInput()
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
		source := "claude"
		if os.Getenv("OPENCODE_HOOK") == "1" {
			source = "opencode"
		}
		input.Source = source
		return hook.HandleSessionEnd(input)

	default:
		// Unknown hook events are silently ignored (don't break Claude Code)
		return nil
	}
}
