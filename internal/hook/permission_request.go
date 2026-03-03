package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
)

// HandlePermissionRequest logs a permission request event.
// Rule evaluation is handled by HandlePreTool; this handler only logs.
func HandlePermissionRequest(input *Input) (Output, error) {
	cfg, err := config.Load()
	if err != nil {
		return Output{}, nil
	}

	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return Output{}, nil
	}

	proj := project.Detect(cfg, input.CWD)

	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPermissionReq,
		Ticket:  lookupActiveTicket(cfg, proj, input.SessionID),
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
	})

	return Output{}, nil
}
