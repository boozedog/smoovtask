package hook

import (
	"fmt"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// HandleSessionStart processes the SessionStart hook.
// It logs a session-start event with ticket counts and returns
// minimal context (run ID + command reference) for injection.
func HandleSessionStart(input *Input) (*Output, error) {
	cfg, err := config.Load()
	if err != nil {
		return &Output{}, fmt.Errorf("load config: %w", err)
	}

	proj := project.Detect(cfg, input.CWD)
	if proj == "" {
		return &Output{}, nil
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return nil, fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)

	// Count tickets for event logging only — not shown to agent.
	openTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	reviewTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusReview,
	})
	if err != nil {
		return nil, fmt.Errorf("list review tickets: %w", err)
	}

	// Log session-start event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return nil, fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookSessionStart,
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
		Data: map[string]any{
			"open_count":   len(openTickets),
			"review_count": len(reviewTickets),
		},
	})

	var b strings.Builder
	fmt.Fprintf(&b, "smoovtask — %s\n", proj)
	fmt.Fprintf(&b, "Run: %s\n", input.SessionID)
	b.WriteString(quickRef)

	return &Output{AdditionalContext: b.String()}, nil
}

const quickRef = "\nOther commands (always pass --run-id <your-run-id>):\n" +
	"  st new \"title\" [-p P0..P5] [-d \"desc\"]       — create a ticket\n" +
	"  st list [--project X] [--status open|review]  — filter tickets\n" +
	"  st show <id>                                  — full ticket detail\n" +
	"  st context                                    — current session info\n" +
	"All commands support --help for full usage.\n"
