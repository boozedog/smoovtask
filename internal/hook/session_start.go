package hook

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/guidance"
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

	proj := detectProject(cfg, input.CWD)
	if proj == "" {
		return &Output{}, nil
	}

	projectsDir, err := cfg.ProjectsDir()
	if err != nil {
		return nil, fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(projectsDir)

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
	b.WriteString("## Commit Rules\n")
	b.WriteString(guidance.CommitStyle())
	b.WriteString("\n\n")
	b.WriteString(quickRefForRole(role))

	// Detect recently handed-off tickets (plan-mode-exit) and inject pickup instructions.
	if pickup := recentHandoffPickup(store, proj, input.SessionID); pickup != "" {
		b.WriteString("\n")
		b.WriteString(pickup)
	}

	return &Output{AdditionalContext: wrapAdditionalContext(b.String())}, nil
}

// recentHandoffPickup checks for OPEN tickets with no assignee that were
// updated within the last 5 minutes — these are likely plan-mode handoffs
// waiting for the new build session to pick them up.
func recentHandoffPickup(store *ticket.Store, proj, sessionID string) string {
	tickets, err := store.ListMeta(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusOpen,
	})
	if err != nil {
		return ""
	}

	cutoff := time.Now().UTC().Add(-5 * time.Minute)
	var candidates []*ticket.Ticket
	for _, tk := range tickets {
		if tk.Assignee == "" && tk.Updated.After(cutoff) {
			candidates = append(candidates, tk)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Plan Continuation\n")
	if len(candidates) == 1 {
		tk := candidates[0]
		fmt.Fprintf(&b, "A ticket was recently handed off and is ready for implementation.\n")
		fmt.Fprintf(&b, "Run `st pick %s --run-id %s` to claim it, then save your implementation plan\n", tk.ID, sessionID)
		fmt.Fprintf(&b, "as a note on the ticket: write the plan to `%s-note.md` using the Write tool,\n", tk.ID)
		fmt.Fprintf(&b, "then run `st note --file %s-note.md --ticket %s --run-id %s`.\n", tk.ID, tk.ID, sessionID)
	} else {
		b.WriteString("Multiple tickets were recently handed off and are ready for implementation:\n")
		for _, tk := range candidates {
			fmt.Fprintf(&b, "- `%s` — %s\n", tk.ID, tk.Title)
		}
		fmt.Fprintf(&b, "Run `st pick <ticket-id> --run-id %s` to claim one.\n", sessionID)
	}
	return b.String()
}

// noteGuidance returns the note-writing instruction for hook injection.
func noteGuidance() string {
	return "- To add notes: use the Write tool to write your note content to `<ticket-id>-note.md` in the current directory, " +
		"then run `st note --file <ticket-id>-note.md --ticket <ticket-id> --run-id <run-id>` (the file is deleted after reading)\n"
}

func quickRefGeneric() string {
	return "## Review Semantics\n" +
		"`st status review` moves work to `REVIEW` (agentic review queue), and `st status human-review` moves it to `HUMAN-REVIEW` (human sign-off queue).\n" +
		"Use `st review` only for claiming agent-review tickets.\n\n" +
		"## Implementing\n" +
		"Before making code changes, claim a ticket. Ask the user before picking or creating one.\n" +
		"- `st list --run-id <run-id>`              view candidate tickets\n" +
		"- `st pick <ticket-id> --run-id <run-id>`  claim a ticket\n" +
		"- `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`  create a new ticket\n" +
		"- `st handoff <ticket-id> --run-id <run-id>`  return a claimed ticket to OPEN\n" +
		"- `st status review --run-id <run-id>`    move ticket to REVIEW when implementation is done\n" +
		"- Always commit all changes in the ticket worktree before requesting review — `st status review` will reject uncommitted work\n\n" +
		"## Reviewing\n" +
		"Use `st review` to claim the agentic review pass — it prints the checklist and ticket context.\n" +
		"- `st review <ticket-id> --run-id <run-id>`  claim a ticket for review\n" +
		"- `st status done --run-id <run-id>`          approve directly (only if absolutely certain you can fully verify correctness yourself)\n" +
		"- `st status human-review --run-id <run-id>`  hand off to human review (default — use when in any doubt)\n" +
		"- `st status rework --run-id <run-id>`        send back for changes\n\n" +
		"## Always\n" +
		noteGuidance() +
		"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
		"- `st context --run-id <run-id>`            check current session context\n\n" +
		"Run `st --help` for more.\n"
}

func quickRefLeader() string {
	return "## Leader\n" +
		"- Launch implementers with `st work` (or `st work --cli opencode`)\n" +
		"- Launch reviewers with `st review <ticket-id>`\n" +
		"- Launch background workers with `st spawn <ticket-id> --run-id <run-id>`\n" +
		"- Create tickets with `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`\n" +
		"- Monitor work with `st list --run-id <run-id>` and inspect details via `st show <ticket-id>`\n" +
		noteGuidance() +
		"Run `st --help` for more.\n"
}

func quickRefImplementer() string {
	return "## Implementing\n" +
		"Before making code changes, claim a ticket. Ask the user before picking or creating one.\n" +
		"- `st list --run-id <run-id>`              view candidate tickets\n" +
		"- `st pick <ticket-id> --run-id <run-id>`  claim a ticket\n" +
		"- `st new \"title\" -p P3 -d \"desc\" --run-id <run-id>`  create a new ticket\n" +
		"- `st handoff <ticket-id> --run-id <run-id>`  return a claimed ticket to OPEN\n" +
		"- `st status review --run-id <run-id>`    move ticket to REVIEW when implementation is done\n" +
		"- Always commit all changes in the ticket worktree before requesting review — `st status review` will reject uncommitted work\n\n" +
		"## Always\n" +
		noteGuidance() +
		"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
		"- `st context --run-id <run-id>`            check current session context\n\n" +
		"Run `st --help` for more.\n"
}

func quickRefReviewer() string {
	return "## Reviewing\n" +
		"- Claim a ticket with `st review <ticket-id> --run-id <run-id>` (eligibility enforced)\n" +
		"- `st status done --run-id <run-id>`          approve directly (only if you are absolutely certain you can fully verify correctness yourself)\n" +
		"- `st status human-review --run-id <run-id>`  hand off to human review (default — use when in any doubt)\n" +
		"- `st status rework --run-id <run-id>`        send back for changes\n\n" +
		"## Always\n" +
		noteGuidance() +
		"- `st show <ticket-id> --run-id <run-id>`   view full ticket details\n" +
		"- `st context --run-id <run-id>`            check current session context\n\n" +
		"Run `st --help` for more.\n"
}

func quickRefWorker() string {
	return "## Worker\n" +
		noteGuidance() +
		"- Update status with `st status <status> --run-id <run-id>`\n" +
		"- Check context with `st context --run-id <run-id>`\n\n" +
		"Run `st --help` for more.\n"
}

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
		return quickRefLeader()
	case "implementer":
		return quickRefImplementer()
	case "reviewer":
		return quickRefReviewer()
	case "worker":
		return quickRefWorker()
	default:
		return quickRefGeneric()
	}
}
