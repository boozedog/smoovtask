package handler

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// Agents renders the agents page.
func (h *Handler) Agents(w http.ResponseWriter, r *http.Request) {
	data := h.buildAgentsData(r)
	_ = templates.AgentsPage(data).Render(r.Context(), w)
}

// PartialAgents renders the agents partial for htmx swaps.
func (h *Handler) PartialAgents(w http.ResponseWriter, r *http.Request) {
	data := h.buildAgentsData(r)
	_ = templates.AgentsPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildAgentsData(r *http.Request) templates.AgentsData {
	const recentLimit = 500
	const stalledThreshold = 2 * time.Minute
	const staleThreshold = 10 * time.Minute
	const maxEventsPerAgent = 15

	filterProject := r.URL.Query().Get("project")
	events := recentEvents(h.eventsDir, event.Query{}, recentLimit)

	type agentState struct {
		runID       string
		source      string
		ticket      string
		lastEventTS time.Time
		ended       bool
		events      []event.Event
	}

	agents := make(map[string]*agentState)
	// Process events newest-first (that's how recentEvents returns them).
	// We need to process in chronological order for correct ended detection.
	// Reverse to chronological order.
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if !strings.HasPrefix(ev.Event, "hook.") {
			continue
		}
		if ev.RunID == "" {
			continue
		}

		a, ok := agents[ev.RunID]
		if !ok {
			a = &agentState{runID: ev.RunID}
			agents[ev.RunID] = a
		}

		if ev.Source != "" && a.source == "" {
			a.source = ev.Source
		}
		if ev.Ticket != "" {
			a.ticket = ev.Ticket
		}
		if ev.TS.After(a.lastEventTS) {
			a.lastEventTS = ev.TS
		}

		// Track ended state: session-end or stop means ended.
		if ev.Event == event.HookSessionEnd || ev.Event == event.HookStop {
			a.ended = true
		} else {
			// A non-end event after an end event means the session restarted.
			a.ended = false
		}

		a.events = append(a.events, ev)
	}

	now := time.Now().UTC()
	var result []templates.AgentInfo

	for _, a := range agents {
		// Filter out ended sessions.
		if a.ended {
			continue
		}
		// Filter out stale sessions.
		if now.Sub(a.lastEventTS) > staleThreshold {
			continue
		}

		// Compute heat state.
		age := now.Sub(a.lastEventTS)
		stalled := age > stalledThreshold
		heat := "cold"
		switch {
		case age <= 60*time.Second:
			heat = "hot"
		case age <= 120*time.Second:
			heat = "warm"
		}

		// Resolve ticket title, but only if this agent is still the assignee.
		// Events may reference a ticket that has since been reassigned.
		var ticketTitle string
		if a.ticket != "" {
			tk, err := h.store.Get(a.ticket)
			if err == nil && tk != nil {
				if tk.Assignee == a.runID {
					ticketTitle = tk.Title
					// Filter by project if set.
					if filterProject != "" && tk.Project != filterProject {
						continue
					}
				} else {
					a.ticket = ""
				}
			}
		}
		// If filtering by project and agent has no ticket, skip.
		if filterProject != "" && a.ticket == "" {
			continue
		}

		// Collect recent events (newest first), cap at maxEventsPerAgent.
		agentEvents := a.events
		// Reverse to newest-first.
		for i, j := 0, len(agentEvents)-1; i < j; i, j = i+1, j-1 {
			agentEvents[i], agentEvents[j] = agentEvents[j], agentEvents[i]
		}
		if len(agentEvents) > maxEventsPerAgent {
			agentEvents = agentEvents[:maxEventsPerAgent]
		}

		result = append(result, templates.AgentInfo{
			RunID:       a.runID,
			Source:      a.source,
			Ticket:      a.ticket,
			TicketTitle: ticketTitle,
			HeatState:   heat,
			LastEventTS: a.lastEventTS,
			Stalled:     stalled,
			Events:      agentEvents,
		})
	}

	// Sort: hot first, then warm, then cold; within same heat, newest first.
	heatOrder := map[string]int{"hot": 0, "warm": 1, "cold": 2}
	sort.Slice(result, func(i, j int) bool {
		hi, hj := heatOrder[result[i].HeatState], heatOrder[result[j].HeatState]
		if hi != hj {
			return hi < hj
		}
		return result[i].LastEventTS.After(result[j].LastEventTS)
	})

	return templates.AgentsData{
		Agents:         result,
		CurrentProject: filterProject,
		Projects:       h.allProjects(),
	}
}
