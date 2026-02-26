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
	_ = templates.ActivityPage(data).Render(r.Context(), w)
}

// PartialActivity renders just the activity content for htmx swaps.
func (h *Handler) PartialActivity(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildActivityData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Push canonical /activity URL so filters are bookmarkable.
	pushURL := "/activity"
	if q := r.URL.Query().Encode(); q != "" {
		pushURL += "?" + q
	}
	w.Header().Set("HX-Push-Url", pushURL)

	_ = templates.ActivityContent(data).Render(r.Context(), w)
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
