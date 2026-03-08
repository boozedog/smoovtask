package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
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

	proj := detectProject(cfg, input.CWD)

	el := event.NewEventLog(eventsDir)
	return el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookTaskCompleted,
		Ticket:  lookupActiveTicket(cfg, proj, input.SessionID),
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
	})
}
