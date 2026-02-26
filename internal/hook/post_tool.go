package hook

import (
	"time"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/project"
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

	proj := project.Detect(cfg, input.CWD)

	el := event.NewEventLog(eventsDir)
	return el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPostTool,
		Project: proj,
		Actor:   "agent",
		Session: input.SessionID,
		Data: map[string]any{
			"tool": input.ToolName,
		},
	})
}
