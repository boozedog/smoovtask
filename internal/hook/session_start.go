package hook

import (
	"fmt"
	"os"
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
		Ticket:  lookupActiveTicket(cfg, proj, input.SessionID),
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
	role := normalizeRole(os.Getenv("ST_ROLE"))
	fmt.Fprintf(&b, "You are working in a tracked smoovtask project called %s. ", proj)
	fmt.Fprintf(&b, "Your run ID is `%s`; include `--run-id <run-id>` on every `st` command.\n", input.SessionID)
	if role == "" {
		b.WriteString("Ask the user whether you are implementing or reviewing — do not guess.\n")
		b.WriteString("Before moving a ticket to `review`, confirm with the user that implementation is actually done.\n\n")
	} else {
		fmt.Fprintf(&b, "Session role: `%s`.\n\n", role)
	}
	b.WriteString(quickRefForRole(role))

	return &Output{AdditionalContext: wrapAdditionalContext(b.String())}, nil
}

// noteGuidance tells agents how to write notes using the file-based drop approach.
// Agents write their note content to .st/notes/<run-id>.md using the Write tool
// (which avoids shell escaping issues), then run `st note` to append it.
const noteGuidance = "- To add notes: use the Write tool to write your note content to `.st/notes/<run-id>.md`, then run `st note --run-id <run-id>` (no message arg needed — it reads from the file)\n"

const quickRefGeneric = "## Review Semantics\n" +
	"`st status review` moves work to `REVIEW` (agentic review queue), and `st status human-review` moves it to `HUMAN-REVIEW` (human sign-off queue).\n" +
	"Use `st review` only for claiming agent-review tickets.\n\n" +
	"## Implementing\n" +
	"Before making code changes, claim a ticket. Ask the user before picking or creating one.\n" +
	"- `st list --run-id <run-id>`              view candidate tickets\n" +
	"- `st pick <ticket-id> --run-id <run-id>`  claim a ticket\n" +
	"- `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`  create a new ticket\n" +
	"- `st handoff <ticket-id> --run-id <run-id>`  return a claimed ticket to OPEN\n" +
	"- `st status review --run-id <run-id>`    move ticket to REVIEW when implementation is done\n\n" +
	"## Reviewing\n" +
	"Use `st review` to claim the agentic review pass — it prints the checklist and ticket context.\n" +
	"- `st review <ticket-id> --run-id <run-id>`  claim a ticket for review\n" +
	"- `st status done --run-id <run-id>`          approve directly (only if absolutely certain you can fully verify correctness yourself)\n" +
	"- `st status human-review --run-id <run-id>`  hand off to human review (default — use when in any doubt)\n" +
	"- `st status rework --run-id <run-id>`        send back for changes\n\n" +
	"## Always\n" +
	noteGuidance +
	"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
	"- `st context --run-id <run-id>`            check current session context\n\n" +
	"Run `st --help` for more.\n"

const quickRefLeader = "## Leader\n" +
	"- Launch implementers with `st work` (or `st work --cli opencode`)\n" +
	"- Launch reviewers with `st review <ticket-id>`\n" +
	"- Launch background workers with `st spawn <ticket-id> --run-id <run-id>`\n" +
	"- Monitor work with `st list --run-id <run-id>` and inspect details via `st show <ticket-id>`\n" +
	noteGuidance +
	"Run `st --help` for more.\n"

const quickRefImplementer = "## Implementing\n" +
	"Before making code changes, claim a ticket. Ask the user before picking or creating one.\n" +
	"- `st list --run-id <run-id>`              view candidate tickets\n" +
	"- `st pick <ticket-id> --run-id <run-id>`  claim a ticket\n" +
	"- `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`  create a new ticket\n" +
	"- `st handoff <ticket-id> --run-id <run-id>`  return a claimed ticket to OPEN\n" +
	"- `st status review --run-id <run-id>`    move ticket to REVIEW when implementation is done\n\n" +
	"## Always\n" +
	noteGuidance +
	"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
	"- `st context --run-id <run-id>`            check current session context\n\n" +
	"Run `st --help` for more.\n"

const quickRefReviewer = "## Reviewing\n" +
	"- Claim a ticket with `st review <ticket-id> --run-id <run-id>` (eligibility enforced)\n" +
	"- `st status done --run-id <run-id>`          approve directly (only if you are absolutely certain you can fully verify correctness yourself)\n" +
	"- `st status human-review --run-id <run-id>`  hand off to human review (default — use when in any doubt)\n" +
	"- `st status rework --run-id <run-id>`        send back for changes\n\n" +
	"## Always\n" +
	noteGuidance +
	"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
	"- `st context --run-id <run-id>`            check current session context\n\n" +
	"Run `st --help` for more.\n"

const quickRefWorker = "## Worker\n" +
	noteGuidance +
	"- Update status with `st status <status> --run-id <run-id>`\n" +
	"- Check context with `st context --run-id <run-id>`\n\n" +
	"Run `st --help` for more.\n"

func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "leader", "implementer", "reviewer", "worker":
		return role
	case "work":
		return "implementer"
	case "review":
		return "reviewer"
	default:
		return ""
	}
}

func quickRefForRole(role string) string {
	switch role {
	case "leader":
		return quickRefLeader
	case "implementer":
		return quickRefImplementer
	case "reviewer":
		return quickRefReviewer
	case "worker":
		return quickRefWorker
	default:
		return quickRefGeneric
	}
}
