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

	case "pi-event":
		eventData, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read event data: %w", err)
		}

		var event map[string]any
		if err := json.Unmarshal(eventData, &event); err != nil {
			return nil
		}

		input := &hook.Input{Source: "pi"}
		if sessionID, ok := event["session_id"].(string); ok {
			input.SessionID = sessionID
		}
		if cwd, ok := event["cwd"].(string); ok {
			input.CWD = cwd
		}
		if toolName, ok := event["tool_name"].(string); ok {
			input.ToolName = toolName
		}
		if taskPrompt, ok := event["task_prompt"].(string); ok {
			input.TaskPrompt = taskPrompt
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "session_start":
			out, err := hook.HandleSessionStart(input)
			if err != nil {
				return err
			}
			return hook.WriteOutput(*out)
		case "tool_call":
			out, err := hook.HandlePreTool(input)
			if err != nil {
				return err
			}
			if out.AdditionalContext != "" {
				return hook.WriteOutput(out)
			}
			return nil
		case "tool_result", "tool_execution_end":
			return hook.HandlePostTool(input)
		case "permission_request":
			out, err := hook.HandlePermissionRequest(input)
			if err != nil {
				return err
			}
			if out.Decision != nil {
				return hook.WriteOutput(out)
			}
			return nil
		case "agent_end", "task_completed":
			return hook.HandleTaskCompleted(input)
		case "teammate_idle", "turn_end":
			return hook.HandleTeammateIdle(input)
		case "subagent_start":
			out, err := hook.HandleSubagentStart(input)
			if err != nil {
				return err
			}
			if out.AdditionalContext != "" {
				return hook.WriteOutput(out)
			}
			return nil
		case "subagent_stop":
			return hook.HandleSubagentStop(input)
		case "session_shutdown":
			if err := hook.HandleStop(input); err != nil {
				return err
			}
			return hook.HandleSessionEnd(input)
		case "stop":
			return hook.HandleStop(input)
		case "session_end":
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
