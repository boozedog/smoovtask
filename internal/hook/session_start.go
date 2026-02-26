package hook

import (
	"fmt"
	"strings"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/project"
	"github.com/boozedog/smoovbrain/internal/ticket"
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

// buildBoardSummary formats the board summary for session context injection.
func buildBoardSummary(proj string, openTickets, reviewTickets []*ticket.Ticket) string {
	// Prefer review tickets if any exist (clearing the review queue unblocks others)
	var tickets []*ticket.Ticket
	var statusLabel string

	if len(reviewTickets) > 0 {
		tickets = reviewTickets
		statusLabel = "REVIEW"
	} else if len(openTickets) > 0 {
		tickets = openTickets
		statusLabel = "OPEN"
	} else {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "smoovbrain — %s — %d %s tickets ready\n\n", proj, len(tickets), statusLabel)

	for _, tk := range tickets {
		fmt.Fprintf(&b, "  %-12s %-30s %s\n", tk.ID, tk.Title, tk.Priority)
	}

	b.WriteString("\n")
	if statusLabel == "OPEN" {
		b.WriteString("Pick a ticket with `sb pick sb_xxxxxx`.\n")
	} else {
		b.WriteString("Review a ticket with `sb review sb_xxxxxx`.\n")
	}

	return b.String()
}
