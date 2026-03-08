package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
)

// HandlePostTool logs a post-tool event to the JSONL event log.
func HandlePostTool(input *Input) error {
	cfg, err := config.Load()
	if err != nil {
		return nil // Don't fail on config errors for async hooks
	}

	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return nil
	}

	proj := detectProject(cfg, input.CWD)

	data := map[string]any{
		"tool": input.ToolName,
	}
	if input.ToolName == "Bash" {
		if code, ok := input.ToolResponse["exit_code"]; ok {
			data["exit_code"] = code
		}
	}

	el := event.NewEventLog(eventsDir)
	return el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPostTool,
		Ticket:  lookupActiveTicket(cfg, proj, input.SessionID),
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
		Data:    data,
	})
}
