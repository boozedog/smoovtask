package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// findProjectFromCwd detects the project from the current working directory.
func findProjectFromCwd(cfg *config.Config, cwd string) string {
	return project.Detect(cfg, cwd)
}

// resolveCurrentTicket finds the ticket to operate on.
// Priority: ticketOverride (from --ticket flag) > scan for ticket assigned to current run.
func resolveCurrentTicket(store *ticket.Store, cfg *config.Config, runID, ticketOverride string) (*ticket.Ticket, error) {
	if ticketOverride != "" {
		return store.Get(ticketOverride)
	}

	if runID == "" {
		return nil, fmt.Errorf("no --ticket specified and no run ID set — use --ticket <id> or run `st pick` first")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	proj := ""
	if cfg != nil {
		proj = findProjectFromCwd(cfg, cwd)
	}

	tickets, err := store.List(ticket.ListFilter{Project: proj})
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	var matches []*ticket.Ticket
	for _, tk := range tickets {
		if tk.Assignee == runID &&
			(tk.Status == ticket.StatusInProgress || tk.Status == ticket.StatusRework) {
			matches = append(matches, tk)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, tk := range matches {
			ids = append(ids, tk.ID)
		}
		return nil, fmt.Errorf("multiple active tickets found for run %q: %s — use --ticket <id>", runID, strings.Join(ids, ", "))
	}

	return nil, fmt.Errorf("no active ticket found for run %q — use `st pick` first or specify --ticket", runID)
}

// resolveReviewTicket finds a ticket to review.
// A ticket must be specified explicitly to avoid accidentally claiming a different ticket.
func resolveReviewTicket(store *ticket.Store, ticketID string) (*ticket.Ticket, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("no ticket specified — use `st review <id>` or `st review --ticket <id>`")
	}

	return store.Get(ticketID)
}
