package handler

import (
	"net/http"
	"strings"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// Activity renders the activity feed page.
func (h *Handler) Activity(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildActivityData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ActivityPage(data).Render(r.Context(), w)
}

func (h *Handler) buildActivityData(r *http.Request) (templates.ActivityData, error) {
	filterProject := r.URL.Query().Get("project")
	filterEventType := r.URL.Query().Get("event_type")

	q := event.Query{
		Project: filterProject,
	}

	events := recentEvents(h.eventsDir, q, 200)

	// Filter by event type prefix if specified.
	if filterEventType != "" {
		var filtered []event.Event
		for _, e := range events {
			if strings.HasPrefix(e.Event, filterEventType+".") {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	// Collect unique project names.
	allTickets, _ := h.store.List(ticket.ListFilter{})
	projects := uniqueProjects(allTickets)

	return templates.ActivityData{
		Events:   events,
		Projects: projects,
		Filter: templates.ActivityFilter{
			Project:   filterProject,
			EventType: filterEventType,
		},
	}, nil
}
