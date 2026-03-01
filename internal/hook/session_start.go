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

	humanReviewTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusHumanReview,
	})
	if err != nil {
		return nil, fmt.Errorf("list human review tickets: %w", err)
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
			"open_count":         len(openTickets),
			"review_count":       len(reviewTickets),
			"human_review_count": len(humanReviewTickets),
		},
	})

	var b strings.Builder
	fmt.Fprintf(&b, "You are working in a tracked smoovtask project called %s. ", proj)
	fmt.Fprintf(&b, "Your run ID is `%s`; include `--run-id <run-id>` on every `st` command.\n", input.SessionID)
	b.WriteString("Ask the user whether you are implementing or reviewing — do not guess.\n")
	b.WriteString("Before moving a ticket to `review`, confirm with the user that implementation is actually done.\n\n")
	b.WriteString(quickRef)

	return &Output{AdditionalContext: b.String()}, nil
}

const quickRef = "## Review Semantics\n" +
	"`st status review` moves work to `REVIEW` (agentic review queue), and `st status human-review` moves it to `HUMAN-REVIEW` (human sign-off queue).\n" +
	"Use `st review` only for claiming agent-review tickets.\n\n" +
	"## Implementing\n" +
	"Before making code changes, claim a ticket. Ask the user before picking or creating one.\n" +
	"- `st list --run-id <run-id>`              view candidate tickets\n" +
	"- `st pick <ticket-id> --run-id <run-id>`  claim a ticket\n" +
	"- `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`  create a new ticket\n" +
	"- `st status review --run-id <run-id>`    move ticket to REVIEW when implementation is done\n\n" +
	"## Reviewing\n" +
	"Use `st review` to claim the agentic review pass — it prints the checklist and ticket context.\n" +
	"- `st review <ticket-id> --run-id <run-id>`  claim a ticket for review\n" +
	"- `st status human-review --run-id <run-id>` hand off to human review\n" +
	"- `st status done --run-id <run-id>`         mark done after human review\n" +
	"- `st status rework --run-id <run-id>`      send back for changes\n\n" +
	"## Always\n" +
	"- `st note \"message\" --run-id <run-id>`     log progress, decisions, and user interactions frequently\n" +
	"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
	"- `st context --run-id <run-id>`            check current session context\n\n" +
	"Run `st --help` for more.\n"
