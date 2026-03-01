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
	fmt.Fprintf(&b, "You are working in a tracked smoovtask project called %s. ", proj)
	fmt.Fprintf(&b, "Your run ID is %s; include `--run-id %s` on smoovtask commands. ", input.SessionID, input.SessionID)
	b.WriteString("Before making code changes, ensure you have an active ticket assigned to this run. ")
	b.WriteString("If no ticket ID has been provided, ask the user before picking or creating a ticket.\n\n")
	b.WriteString(quickRef)

	return &Output{AdditionalContext: b.String()}, nil
}

const quickRef = "Quick workflow:\n" +
	"- `st list --run-id <run-id>`                           view candidate tickets\n" +
	"- `st pick <ticket-id> --run-id <run-id>`              claim a specific ticket\n" +
	"- `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`   create a new ticket (after user confirmation)\n" +
	"- `st note --run-id <run-id> \"message\"`                log significant progress, findings, major changes, and notable user interactions\n" +
	"- `st status review --run-id <run-id>`                 submit ticket for review when done\n\n" +
	"Useful extras:\n" +
	"- `st show <ticket-id> --run-id <run-id>`              view full ticket details\n" +
	"- `st context --run-id <run-id>`                       check current session context\n\n" +
	"Help:\n" +
	"- `st --help`\n" +
	"- `st <command> --help`\n"
