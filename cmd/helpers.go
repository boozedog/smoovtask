package cmd

import (
	"fmt"
	"os"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// findProjectFromCwd detects the project from the current working directory.
func findProjectFromCwd(cfg *config.Config, cwd string) string {
	return project.Detect(cfg, cwd)
}

// resolveCurrentTicket finds the ticket to operate on.
// Priority: ticketOverride (from --ticket flag) > scan for ticket assigned to current session.
func resolveCurrentTicket(store *ticket.Store, cfg *config.Config, sessionID, ticketOverride string) (*ticket.Ticket, error) {
	if ticketOverride != "" {
		return store.Get(ticketOverride)
	}

	if sessionID == "" {
		return nil, fmt.Errorf("no --ticket specified and no CLAUDE_SESSION_ID set — use --ticket <id> or run `st pick` first")
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

	for _, tk := range tickets {
		if tk.Assignee == sessionID &&
			(tk.Status == ticket.StatusInProgress || tk.Status == ticket.StatusRework) {
			return tk, nil
		}
	}

	return nil, fmt.Errorf("no active ticket found for session %q — use `st pick` first or specify --ticket", sessionID)
}
