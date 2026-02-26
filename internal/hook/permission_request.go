package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
)

// HandlePermissionRequest logs a permission request event.
// Currently a pass-through — returns empty Output (no decision).
// The plugin system will handle auto-approve logic later.
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
		Project: proj,
		Actor:   "agent",
		Session: input.SessionID,
	})

	return Output{}, nil
}
