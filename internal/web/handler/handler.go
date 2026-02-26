package handler

import (
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/sse"
)

// Handler holds shared dependencies for HTTP handlers.
type Handler struct {
	store     *ticket.Store
	eventsDir string
	broker    *sse.Broker
}

// New creates a new Handler.
func New(ticketsDir, eventsDir string, broker *sse.Broker) *Handler {
	return &Handler{
		store:     ticket.NewStore(ticketsDir),
		eventsDir: eventsDir,
		broker:    broker,
	}
}

// statusOrder defines the column order for the kanban board.
var statusOrder = []ticket.Status{
	ticket.StatusBacklog,
	ticket.StatusOpen,
	ticket.StatusInProgress,
	ticket.StatusReview,
	ticket.StatusRework,
	ticket.StatusBlocked,
	ticket.StatusDone,
}

// groupByStatus organizes tickets into a map keyed by status.
func groupByStatus(tickets []*ticket.Ticket) map[ticket.Status][]*ticket.Ticket {
	groups := make(map[ticket.Status][]*ticket.Ticket)
	for _, tk := range tickets {
		groups[tk.Status] = append(groups[tk.Status], tk)
	}
	return groups
}

// statusWeight returns a sort weight for ticket statuses:
// active states first, then backlog, then done.
func statusWeight(s ticket.Status) int {
	switch s {
	case ticket.StatusOpen, ticket.StatusInProgress, ticket.StatusReview, ticket.StatusRework, ticket.StatusBlocked:
		return 0
	case ticket.StatusBacklog:
		return 1
	case ticket.StatusDone:
		return 2
	default:
		return 1
	}
}

// recentEvents queries the most recent events, limited to count.
func recentEvents(eventsDir string, q event.Query, limit int) []event.Event {
	events, err := event.QueryEvents(eventsDir, q)
	if err != nil {
		return nil
	}
	// Events come sorted chronologically; return the tail.
	if len(events) > limit {
		events = events[len(events)-limit:]
	}
	// Reverse so newest is first.
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events
}
