package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/guidance"
)

// HandleUserPrompt logs a user prompt submission event and injects a note
// reminder when a ticket is actively being worked on.
func HandleUserPrompt(input *Input) (*Output, error) {
	cfg, err := config.Load()
	if err != nil {
		return &Output{}, nil
	}

	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return &Output{}, nil
	}

	proj := detectProject(cfg, input.CWD)
	ticketID := lookupActiveTicket(cfg, proj, input.SessionID)

	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookUserPrompt,
		Ticket:  ticketID,
		Project: proj,
		Actor:   "user",
		RunID:   input.SessionID,
		Source:  input.Source,
	})

	// Inject a note reminder when the agent has an active ticket.
	if ticketID != "" {
		return &Output{
			AdditionalContext: wrapAdditionalContext(guidance.PromptReminder()),
		}, nil
	}

	return &Output{}, nil
}
