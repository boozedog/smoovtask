package workflow

import (
	"fmt"
	"slices"

	"github.com/boozedog/smoovbrain/internal/ticket"
)

// transitions defines the valid status transitions.
var transitions = map[ticket.Status][]ticket.Status{
	ticket.StatusBacklog:    {ticket.StatusOpen, ticket.StatusBlocked},
	ticket.StatusOpen:       {ticket.StatusInProgress, ticket.StatusBlocked},
	ticket.StatusInProgress: {ticket.StatusReview, ticket.StatusBlocked},
	ticket.StatusReview:     {ticket.StatusDone, ticket.StatusRework, ticket.StatusBlocked},
	ticket.StatusRework:     {ticket.StatusInProgress, ticket.StatusBlocked},
	ticket.StatusBlocked:    {}, // unblocks to prior status, handled separately
}

// CanTransition returns true if the transition from → to is valid.
func CanTransition(from, to ticket.Status) bool {
	// BLOCKED can snap back to any prior status.
	if from == ticket.StatusBlocked {
		return true
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}

	return slices.Contains(allowed, to)
}

// ValidateTransition checks if the transition is valid and returns an error with guidance if not.
func ValidateTransition(from, to ticket.Status) error {
	if from == to {
		return fmt.Errorf("ticket is already %s", from)
	}

	if !CanTransition(from, to) {
		return fmt.Errorf("cannot move from %s to %s", from, to)
	}

	return nil
}

// StatusFromAlias resolves status aliases to canonical status values.
func StatusFromAlias(s string) (ticket.Status, error) {
	aliases := map[string]ticket.Status{
		"backlog":     ticket.StatusBacklog,
		"open":        ticket.StatusOpen,
		"in-progress": ticket.StatusInProgress,
		"in_progress": ticket.StatusInProgress,
		"inprogress":  ticket.StatusInProgress,
		"start":       ticket.StatusInProgress,
		"begin":       ticket.StatusInProgress,
		"review":      ticket.StatusReview,
		"submit":      ticket.StatusReview,
		"done":        ticket.StatusDone,
		"complete":    ticket.StatusDone,
		"rework":      ticket.StatusRework,
		"reject":      ticket.StatusRework,
		"blocked":     ticket.StatusBlocked,
		"block":       ticket.StatusBlocked,
	}

	if status, ok := aliases[s]; ok {
		return status, nil
	}

	// Try as canonical status
	status := ticket.Status(s)
	if ticket.ValidStatuses[status] {
		return status, nil
	}

	return "", fmt.Errorf("unknown status %q — use one of: backlog, open, in-progress, review, done, rework, blocked", s)
}
