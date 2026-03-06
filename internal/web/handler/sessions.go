package handler

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// Sessions renders the sessions page.
func (h *Handler) Sessions(w http.ResponseWriter, r *http.Request) {
	data := h.buildSessionsData(r)
	_ = templates.SessionsPage(data).Render(r.Context(), w)
}

// PartialSessions renders the sessions partial for htmx swaps.
func (h *Handler) PartialSessions(w http.ResponseWriter, r *http.Request) {
	data := h.buildSessionsData(r)
	_ = templates.SessionsPartial(data).Render(r.Context(), w)
}

// SessionDetail renders the session detail modal content.
func (h *Handler) SessionDetail(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runID")
	if runID == "" {
		http.Error(w, "missing runID", http.StatusBadRequest)
		return
	}

	const maxDetailEvents = 500
	events, err := event.QueryEvents(h.eventsDir, event.Query{RunID: runID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalEvents := len(events)
	if len(events) > maxDetailEvents {
		events = events[len(events)-maxDetailEvents:]
	}

	// Extract session metadata from events (before reversing).
	var source, project, ticket, ticketTitle string
	var firstTS, lastTS time.Time
	active := true

	for _, ev := range events {
		if firstTS.IsZero() || ev.TS.Before(firstTS) {
			firstTS = ev.TS
		}
		if ev.TS.After(lastTS) {
			lastTS = ev.TS
		}
		if ev.Source != "" && source == "" {
			source = ev.Source
		}
		if ev.Project != "" {
			project = ev.Project
		}
		if ev.Ticket != "" {
			ticket = ev.Ticket
		}
		if ev.Event == event.HookSessionEnd || ev.Event == event.HookStop {
			active = false
		} else if strings.HasPrefix(ev.Event, "hook.") {
			active = true
		}
	}

	// Resolve ticket title.
	if ticket != "" {
		tk, err := h.store.Get(ticket)
		if err == nil && tk != nil {
			ticketTitle = tk.Title
		}
	}

	// Check stale threshold for active sessions.
	const staleThreshold = 10 * time.Minute
	if active && !lastTS.IsZero() && time.Since(lastTS) > staleThreshold {
		active = false
	}

	// Reverse to newest-first for display.
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	data := templates.SessionDetailData{
		RunID:        runID,
		Source:       source,
		Project:      project,
		Ticket:       ticket,
		TicketTitle:  ticketTitle,
		FirstEventTS: firstTS,
		LastEventTS:  lastTS,
		Active:       active,
		Events:       events,
		TotalEvents:  totalEvents,
	}
	_ = templates.SessionDetailPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildSessionsData(r *http.Request) templates.SessionsData {
	const stalledThreshold = 2 * time.Minute
	const staleThreshold = 10 * time.Minute
	const maxEventsPerSession = 15
	const maxEndedSessions = 50

	filterProject := r.URL.Query().Get("project")

	// Query events from the last 3 days.
	after := time.Now().UTC().Add(-3 * 24 * time.Hour)
	events, _ := event.QueryEvents(h.eventsDir, event.Query{After: after})

	type sessionState struct {
		runID       string
		source      string
		ticket      string
		project     string
		firstTS     time.Time
		lastEventTS time.Time
		ended       bool
		eventCount  int
		events      []event.Event
	}

	sessions := make(map[string]*sessionState)
	for _, ev := range events {
		if !strings.HasPrefix(ev.Event, "hook.") {
			continue
		}
		if ev.RunID == "" {
			continue
		}

		s, ok := sessions[ev.RunID]
		if !ok {
			s = &sessionState{runID: ev.RunID, firstTS: ev.TS}
			sessions[ev.RunID] = s
		}

		if ev.Source != "" && s.source == "" {
			s.source = ev.Source
		}
		if ev.Ticket != "" {
			s.ticket = ev.Ticket
		}
		if ev.Project != "" {
			s.project = ev.Project
		}
		if s.firstTS.IsZero() || ev.TS.Before(s.firstTS) {
			s.firstTS = ev.TS
		}
		if ev.TS.After(s.lastEventTS) {
			s.lastEventTS = ev.TS
		}

		if ev.Event == event.HookSessionEnd || ev.Event == event.HookStop {
			s.ended = true
		} else {
			s.ended = false
		}

		s.eventCount++
		s.events = append(s.events, ev)
	}

	now := time.Now().UTC()
	var active, ended []templates.SessionInfo

	for _, s := range sessions {
		// Determine active status: not ended and not stale.
		isActive := !s.ended && now.Sub(s.lastEventTS) <= staleThreshold

		// Resolve ticket title.
		var ticketTitle string
		if s.ticket != "" {
			tk, err := h.store.Get(s.ticket)
			if err == nil && tk != nil {
				ticketTitle = tk.Title
			}
		}

		// Project filtering: use event Project field for all sessions.
		if filterProject != "" {
			// Check ticket project or event project.
			matchesProject := false
			if s.ticket != "" {
				tk, err := h.store.Get(s.ticket)
				if err == nil && tk != nil && tk.Project == filterProject {
					matchesProject = true
				}
			}
			if s.project == filterProject {
				matchesProject = true
			}
			if !matchesProject {
				continue
			}
		}

		// Compute heat state.
		heatState := "ended"
		stalled := false
		if isActive {
			age := now.Sub(s.lastEventTS)
			stalled = age > stalledThreshold
			switch {
			case age <= 60*time.Second:
				heatState = "hot"
			case age <= 120*time.Second:
				heatState = "warm"
			default:
				heatState = "cold"
			}
		}

		// Cap recent events (newest first).
		recentEvents := s.events
		for i, j := 0, len(recentEvents)-1; i < j; i, j = i+1, j-1 {
			recentEvents[i], recentEvents[j] = recentEvents[j], recentEvents[i]
		}
		if len(recentEvents) > maxEventsPerSession {
			recentEvents = recentEvents[:maxEventsPerSession]
		}

		info := templates.SessionInfo{
			RunID:        s.runID,
			Source:       s.source,
			Ticket:       s.ticket,
			TicketTitle:  ticketTitle,
			Project:      s.project,
			HeatState:    heatState,
			FirstEventTS: s.firstTS,
			LastEventTS:  s.lastEventTS,
			Active:       isActive,
			Stalled:      stalled,
			EventCount:   s.eventCount,
			Events:       recentEvents,
		}

		if isActive {
			active = append(active, info)
		} else {
			ended = append(ended, info)
		}
	}

	// Sort active: hot first, then warm, then cold; within same heat, newest first.
	heatOrder := map[string]int{"hot": 0, "warm": 1, "cold": 2}
	sort.Slice(active, func(i, j int) bool {
		hi, hj := heatOrder[active[i].HeatState], heatOrder[active[j].HeatState]
		if hi != hj {
			return hi < hj
		}
		return active[i].LastEventTS.After(active[j].LastEventTS)
	})

	// Sort ended: newest first.
	sort.Slice(ended, func(i, j int) bool {
		return ended[i].LastEventTS.After(ended[j].LastEventTS)
	})
	if len(ended) > maxEndedSessions {
		ended = ended[:maxEndedSessions]
	}

	// Combine: active first, then ended.
	result := make([]templates.SessionInfo, 0, len(active)+len(ended))
	result = append(result, active...)
	result = append(result, ended...)

	return templates.SessionsData{
		Sessions:       result,
		CurrentProject: filterProject,
		Projects:       h.allProjects(),
	}
}
