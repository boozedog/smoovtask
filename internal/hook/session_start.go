package hook

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/guidance"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// HandleSessionStart processes the SessionStart hook.
// It prints the board summary directly to stdout (plain text),
// which Claude Code automatically injects into the agent's context
// for SessionStart hooks.
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

	// Get open tickets for this project
	openTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	// Get review tickets for this project
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

	summary := buildBoardSummary(proj, input.SessionID, openTickets, reviewTickets)
	if summary == "" {
		runID := input.SessionID
		b := strings.Builder{}
		fmt.Fprintf(&b, "smoovtask — %s — no tickets\nRun: %s\n\n", proj, runID)
		b.WriteString("Workflow: new → pick → note → review\n\n")
		b.WriteString("1. `st new \"title\"`\n")
		fmt.Fprintf(&b, "2. `st pick <id> --run-id %s`\n", runID)
		fmt.Fprintf(&b, "3. `st note --ticket <id> --run-id %s \"log progress\"`\n", runID)
		fmt.Fprintf(&b, "4. `st status --ticket <id> --run-id %s review`\n\n", runID)
		b.WriteString("Always use --ticket and --run-id.\n\n")
		b.WriteString("Use `st note` often: decisions, approvals, surprises.\n")
		b.WriteString(quickRef)
		summary = b.String()
	}
	o := &Output{AdditionalContext: summary}
	return o, nil
}

const quickRef = "\nOther commands (always pass --run-id <your-run-id>):\n" +
	"  st new \"title\" [-p P0..P5] [-d \"desc\"]       — create a ticket\n" +
	"  st list [--project X] [--status open|review]  — filter tickets\n" +
	"  st show <id>                                  — full ticket detail\n" +
	"  st context                                    — current session info\n" +
	"All commands support --help for full usage.\n"

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
// It uses score-based batch selection to determine which batch to show first,
// then appends a summary line for the other batch so the agent has full context.
func buildBoardSummary(proj, sessionID string, openTickets, reviewTickets []*ticket.Ticket) string {
	if len(openTickets) == 0 && len(reviewTickets) == 0 {
		return ""
	}

	// Find the max score across both lists to determine which batch to show first.
	maxScore := -1
	showReviewFirst := false

	for _, tk := range openTickets {
		if s := ticketScore(tk); s > maxScore {
			maxScore = s
			showReviewFirst = false
		}
	}
	for _, tk := range reviewTickets {
		if s := ticketScore(tk); s > maxScore {
			maxScore = s
			showReviewFirst = true
		}
	}

	// Build sorted copies of each batch.
	open := make([]*ticket.Ticket, len(openTickets))
	copy(open, openTickets)
	sortByPriority(open)

	review := make([]*ticket.Ticket, len(reviewTickets))
	copy(review, reviewTickets)
	sortByPriority(review)

	total := len(open) + len(review)

	var b strings.Builder
	fmt.Fprintf(&b, "smoovtask — %s — %d tickets ready", proj, total)
	if len(open) > 0 && len(review) > 0 {
		fmt.Fprintf(&b, " (%d open, %d review)", len(open), len(review))
	}
	b.WriteString("\n")
	if sessionID != "" {
		fmt.Fprintf(&b, "Run: %s\n", sessionID)
	}

	// Show the primary batch first, then the secondary batch.
	if showReviewFirst {
		writeTicketBatch(&b, "Review", review)
		writeTicketBatch(&b, "Open", open)
	} else {
		writeTicketBatch(&b, "Open", open)
		writeTicketBatch(&b, "Review", review)
	}

	b.WriteString("Follow the user's instructions. Default workflow when no specific direction is given:\n")
	b.WriteString("  Open tickets:   `st pick` → `st note` → `st status review`\n")
	b.WriteString("  Review tickets: `st review` → `st note` → `st status done` / `st status rework`\n")
	b.WriteString("Always pass --ticket and --run-id to st commands. Pick a ticket before editing code.\n")
	fmt.Fprintf(&b, "\n%s\n", guidance.CompactImplementation)

	b.WriteString(quickRef)

	return b.String()
}

// writeTicketBatch writes a labeled section of tickets to the builder.
// Does nothing if the slice is empty.
func writeTicketBatch(b *strings.Builder, label string, tickets []*ticket.Ticket) {
	if len(tickets) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s:\n", label)
	for _, tk := range tickets {
		fmt.Fprintf(b, "  %-12s %-30s %s\n", tk.ID, tk.Title, tk.Priority)
	}
}
