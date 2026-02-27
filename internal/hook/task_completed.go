package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
)

// HandleTaskCompleted logs a task completion event.
// This is async/log-only — it does not block the agent.
func HandleTaskCompleted(input *Input) error {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return nil
	}

	proj := project.Detect(cfg, input.CWD)

	el := event.NewEventLog(eventsDir)
	return el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookTaskCompleted,
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
	})
}
