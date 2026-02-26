package workflow

import (
	"fmt"
	"slices"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// RequiresAssignee returns true if the target status requires a ticket assignee.
func RequiresAssignee(to ticket.Status) bool {
	return to == ticket.StatusInProgress
}

// RequiresNote returns true if the transition requires that a note was added
// since the ticket entered its current status.
func RequiresNote(from, to ticket.Status) bool {
	if from == ticket.StatusInProgress && to == ticket.StatusReview {
		return true
	}
	if from == ticket.StatusReview && (to == ticket.StatusDone || to == ticket.StatusRework) {
		return true
	}
	return false
}

// HasNoteSince checks whether a ticket.note event exists for the ticket
// since its last status change.
func HasNoteSince(eventsDir, ticketID string, since time.Time) (bool, error) {
	events, err := event.QueryEvents(eventsDir, event.Query{
		TicketID: ticketID,
		After:    since,
	})
	if err != nil {
		return false, fmt.Errorf("query events: %w", err)
	}

	for _, e := range events {
		if e.Event == event.TicketNote {
			return true, nil
		}
	}
	return false, nil
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
