package cmd

import (
	"fmt"

	"github.com/boozedog/smoovbrain/internal/hook"
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

	input, err := hook.ReadInput()
	if err != nil {
		return fmt.Errorf("read hook input: %w", err)
	}

	switch eventType {
	case "session-start":
		out, err := hook.HandleSessionStart(input)
		if err != nil {
			return err
		}
		if out.AdditionalContext != "" {
			return hook.WriteOutput(out)
		}
		return nil

	case "subagent-start":
		out, err := hook.HandleSubagentStart(input)
		if err != nil {
			return err
		}
		if out.AdditionalContext != "" {
			return hook.WriteOutput(out)
		}
		return nil

	case "pre-tool":
		return hook.HandlePreTool(input)

	case "post-tool":
		return hook.HandlePostTool(input)

	case "stop":
		return hook.HandleStop(input)

	case "subagent-stop":
		return hook.HandleSubagentStop(input)

	case "task-completed":
		return hook.HandleTaskCompleted(input)

	case "teammate-idle":
		return hook.HandleTeammateIdle(input)

	case "permission-request":
		out, err := hook.HandlePermissionRequest(input)
		if err != nil {
			return err
		}
		if out.Decision != nil {
			return hook.WriteOutput(out)
		}
		return nil

	case "session-end":
		return hook.HandleSessionEnd(input)

	default:
		// Unknown hook events are silently ignored (don't break Claude Code)
		return nil
	}
}
