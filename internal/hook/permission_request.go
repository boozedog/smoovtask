package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
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

	proj := detectProject(cfg, input.CWD)

	data := make(map[string]any)
	if input.ToolName != "" {
		data["tool"] = input.ToolName
	}
	if input.ToolInput != nil {
		for _, key := range []string{"command", "file_path", "pattern", "description"} {
			if v, ok := input.ToolInput[key]; ok {
				data[key] = v
			}
		}
	}

	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPermissionReq,
		Ticket:  lookupActiveTicket(cfg, proj, input.SessionID),
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
		Data:    data,
	})

	return Output{}, nil
}
