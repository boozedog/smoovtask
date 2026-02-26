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

// PartialActivity renders the activity partial (with filters + SSE self-refresh wrapper) for htmx swaps.
func (h *Handler) PartialActivity(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildActivityData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.ActivityPartial(data).Render(r.Context(), w)
}

// PartialActivityContent renders just the activity event list for filter swaps.
func (h *Handler) PartialActivityContent(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildActivityData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Push canonical /activity URL so filters are bookmarkable.
	pushURL := "/activity"
	params := r.URL.Query()
	for k := range params {
		if params.Get(k) == "" {
			params.Del(k)
		}
	}
	if q := params.Encode(); q != "" {
		pushURL += "?" + q
	}
	w.Header().Set("HX-Push-Url", pushURL)

	_ = templates.ActivityContent(data).Render(r.Context(), w)
}

func (h *Handler) buildActivityData(r *http.Request) (templates.ActivityData, error) {
	filterProject := r.URL.Query().Get("project")
	filterEventType := r.URL.Query().Get("event_type")

	// Default to "ticket_status" filter when no event_type is specified.
	if filterEventType == "" && !r.URL.Query().Has("event_type") {
		filterEventType = "ticket_status"
	}

	q := event.Query{
		Project: filterProject,
	}

	events := recentEvents(h.eventsDir, q, 200)

	// Filter by event type.
	switch filterEventType {
	case "ticket_status":
		var filtered []event.Event
		for _, e := range events {
			if strings.HasPrefix(e.Event, "ticket.") || strings.HasPrefix(e.Event, "status.") {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	case "ticket", "status", "hook":
		var filtered []event.Event
		for _, e := range events {
			if strings.HasPrefix(e.Event, filterEventType+".") {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	default:
		// "all" or empty with explicit param — show everything.
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
