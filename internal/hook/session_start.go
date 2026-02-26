package hook

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// HandleSessionStart processes the SessionStart hook and returns a board summary.
func HandleSessionStart(input *Input) (Output, error) {
	cfg, err := config.Load()
	if err != nil {
		return Output{}, fmt.Errorf("load config: %w", err)
	}

	proj := project.Detect(cfg, input.CWD)
	if proj == "" {
		return Output{}, nil
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return Output{}, fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)

	// Get open tickets for this project
	openTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusOpen,
	})
	if err != nil {
		return Output{}, fmt.Errorf("list tickets: %w", err)
	}

	// Get review tickets for this project
	reviewTickets, err := store.List(ticket.ListFilter{
		Project: proj,
		Status:  ticket.StatusReview,
	})
	if err != nil {
		return Output{}, fmt.Errorf("list review tickets: %w", err)
	}

	summary := buildBoardSummary(proj, openTickets, reviewTickets)
	if summary == "" {
		return Output{}, nil
	}

	return Output{AdditionalContext: summary}, nil
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
func buildBoardSummary(proj string, openTickets, reviewTickets []*ticket.Ticket) string {
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
	fmt.Fprintf(&b, "smoovtask — %s — %d %s tickets ready\n\n", proj, len(tickets), statusLabel)

	for _, tk := range tickets {
		fmt.Fprintf(&b, "  %-12s %-30s %s\n", tk.ID, tk.Title, tk.Priority)
	}

	b.WriteString("\n")
	if statusLabel == "OPEN" {
		b.WriteString("Pick a ticket with `st pick st_xxxxxx`.\n")
	} else {
		b.WriteString("Review a ticket with `st review st_xxxxxx`.\n")
	}

	return b.String()
}
