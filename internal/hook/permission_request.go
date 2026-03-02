package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/rules"
)

// HandlePermissionRequest logs a permission request event and evaluates
// rules to return allow/deny decisions for tool use requests.
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

	// Evaluate rules
	rulesDir, err := cfg.RulesDir()
	if err != nil {
		return Output{}, nil
	}

	result := rules.Evaluate(rulesDir, input.HookEventName, input.ToolName, input.ToolInput)
	if result == nil {
		return Output{}, nil
	}

	switch result.Decision {
	case rules.ActionAllow:
		return Output{
			Decision: &Decision{Behavior: "allow", Reason: result.Reason},
		}, nil
	case rules.ActionDeny:
		return Output{
			Decision: &Decision{Behavior: "deny", Reason: result.Reason},
		}, nil
	default:
		// ActionAsk or unknown — passthrough
		return Output{}, nil
	}
}
