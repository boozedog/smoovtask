package hook

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// HandleSessionStart processes the SessionStart hook.
// It prints the board summary directly to stdout (plain text),
// which Claude Code automatically injects into the agent's context
// for SessionStart hooks.
func HandleSessionStart(input *Input) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	proj := project.Detect(cfg, input.CWD)
	if proj == "" {
		return nil
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)

	// Get open tickets for this project
	openTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusOpen,
	})
	if err != nil {
		return fmt.Errorf("list tickets: %w", err)
	}

	// Get review tickets for this project
	reviewTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusReview,
	})
	if err != nil {
		return fmt.Errorf("list review tickets: %w", err)
	}

	// Log session-start event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookSessionStart,
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Data: map[string]any{
			"open_count":   len(openTickets),
			"review_count": len(reviewTickets),
		},
	})

	summary := buildBoardSummary(proj, input.SessionID, openTickets, reviewTickets)
	if summary != "" {
		fmt.Print(summary)
	} else {
		fmt.Printf("smoovtask — %s — no open tickets\n\nIf no ticket exists for your task, create one first with `st new \"title\"`.\n", proj)
	}

	return nil
}

// priorityWeight returns the scoring weight for a ticket priority.
// P0=60, P1=50, P2=40, P3=30, P4=20, P5=10.
var priorityWeight = map[ticket.Priority]int{
	ticket.PriorityP0: 60,
	ticket.PriorityP1: 50,
	ticket.PriorityP2: 40,
	ticket.PriorityP3: 30,
	ticket.PriorityP4: 20,
	ticket.PriorityP5: 10,
}

// statusBoost is the score bonus for REVIEW tickets.
const statusBoostReview = 5

// ticketScore calculates the priority score for a ticket.
// Score = priority weight + status boost (REVIEW gets +5, OPEN gets +0).
func ticketScore(tk *ticket.Ticket) int {
	score := priorityWeight[tk.Priority]
	if tk.Status == ticket.StatusReview {
		score += statusBoostReview
	}
	return score
}

// sortByPriority sorts tickets by priority (P0 first, P5 last).
func sortByPriority(tickets []*ticket.Ticket) {
	slices.SortFunc(tickets, func(a, b *ticket.Ticket) int {
		return cmp.Compare(priorityWeight[b.Priority], priorityWeight[a.Priority])
	})
}

// buildBoardSummary formats the board summary for session context injection.
// It uses score-based batch selection: score all tickets, find the highest,
// then present ALL tickets of that same status type sorted by priority.
func buildBoardSummary(proj, sessionID string, openTickets, reviewTickets []*ticket.Ticket) string {
	if len(openTickets) == 0 && len(reviewTickets) == 0 {
		return ""
	}

	// Find the max score across both lists to determine which batch to show.
	maxScore := -1
	showReview := false

	for _, tk := range openTickets {
		if s := ticketScore(tk); s > maxScore {
			maxScore = s
			showReview = false
		}
	}
	for _, tk := range reviewTickets {
		if s := ticketScore(tk); s > maxScore {
			maxScore = s
			showReview = true
		}
	}

	var tickets []*ticket.Ticket
	var statusLabel string

	if showReview {
		tickets = make([]*ticket.Ticket, len(reviewTickets))
		copy(tickets, reviewTickets)
		statusLabel = "REVIEW"
	} else {
		tickets = make([]*ticket.Ticket, len(openTickets))
		copy(tickets, openTickets)
		statusLabel = "OPEN"
	}

	sortByPriority(tickets)

	var b strings.Builder
	fmt.Fprintf(&b, "smoovtask — %s — %d %s tickets ready\n", proj, len(tickets), statusLabel)
	if sessionID != "" {
		fmt.Fprintf(&b, "Run: %s\n", sessionID)
	}
	b.WriteString("\n")

	for _, tk := range tickets {
		fmt.Fprintf(&b, "  %-12s %-30s %s\n", tk.ID, tk.Title, tk.Priority)
	}

	b.WriteString("\n")
	if statusLabel == "OPEN" {
		b.WriteString("REQUIRED workflow — you MUST follow these steps:\n")
		b.WriteString("1. `st pick st_xxxxxx --run-id <your-run-id>` — claim a ticket before starting any work\n")
		b.WriteString("2. `st note --ticket st_xxxxxx --run-id <your-run-id> \"message\"` — document progress as you work\n")
		b.WriteString("3. `st status --ticket st_xxxxxx --run-id <your-run-id> review` — submit when done\n")
		b.WriteString("\nALWAYS pass --ticket and --run-id to st commands. Your run ID is shown above. Do NOT start editing code without picking a ticket first.\n")
		b.WriteString("\nLOG FREQUENTLY: Use `st note` throughout your work — not just at the end. Log key decisions, discussions with the user (clarifications, scope changes, approvals), and anything surprising. Include brief code snippets where they help explain a change. Notes are the ticket's audit trail.\n")
	} else {
		b.WriteString("REQUIRED workflow — you MUST follow these steps:\n")
		b.WriteString("1. `st review --ticket st_xxxxxx --run-id <your-run-id>` — claim a ticket for review\n")
		b.WriteString("2. `st note --ticket st_xxxxxx --run-id <your-run-id> \"<findings>\"` — document your review findings\n")
		b.WriteString("3. `st status --ticket st_xxxxxx --run-id <your-run-id> done` (approve) or `st status --ticket st_xxxxxx --run-id <your-run-id> rework` (reject)\n")
		b.WriteString("\nALWAYS pass --ticket and --run-id to st commands. Your run ID is shown above. Do NOT approve or reject without documenting findings via `st note` first.\n")
		b.WriteString("\nLOG FREQUENTLY: Use `st note` throughout your work — not just at the end. Log key decisions, discussions with the user (clarifications, scope changes, approvals), and anything surprising. Include brief code snippets where they help explain a change. Notes are the ticket's audit trail.\n")
	}

	return b.String()
}
