package workflow

import (
	"fmt"
	"slices"

	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/ticket"
)

// RequiresAssignee returns true if the target status requires a ticket assignee.
func RequiresAssignee(to ticket.Status) bool {
	return to == ticket.StatusInProgress
}

// RequiresNote returns true if the transition requires that a note was added
// since the ticket entered its current status.
func RequiresNote(from, to ticket.Status) bool {
	return from == ticket.StatusInProgress && to == ticket.StatusReview
}

// CanReview checks whether the given session is eligible to review a ticket.
// A session that has previously touched the ticket (appears in JSONL) is not eligible.
func CanReview(eventsDir, ticketID, sessionID string) (bool, error) {
	sessions, err := event.SessionsForTicket(eventsDir, ticketID)
	if err != nil {
		return false, fmt.Errorf("query sessions: %w", err)
	}

	return !slices.Contains(sessions, sessionID), nil
}
