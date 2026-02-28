package handler

import (
	"net/http"
	"sort"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// Board renders the kanban board page.
func (h *Handler) Board(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildBoardData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.BoardPage(data).Render(r.Context(), w)
}

// PartialBoard renders the board partial (with SSE self-refresh wrapper) for htmx swaps.
func (h *Handler) PartialBoard(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildBoardData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.BoardPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildBoardData() (templates.BoardData, error) {
	tickets, err := h.store.List(ticket.ListFilter{})
	if err != nil {
		return templates.BoardData{}, err
	}

	groups := groupByStatus(tickets)

	// Sort tickets within each column.
	for status, tks := range groups {
		if status == ticket.StatusDone || status == ticket.StatusCancelled {
			// Done/Cancelled: reverse chronological by Updated.
			sort.Slice(tks, func(i, j int) bool {
				return tks[i].Updated.After(tks[j].Updated)
			})
		} else {
			// All others: priority ascending (P0 first), then creation date ascending.
			sort.Slice(tks, func(i, j int) bool {
				if tks[i].Priority != tks[j].Priority {
					return tks[i].Priority < tks[j].Priority
				}
				return tks[i].Created.Before(tks[j].Created)
			})
		}
	}

	var columns []templates.BoardColumn
	for _, status := range statusOrder {
		columns = append(columns, templates.BoardColumn{
			Status:  status,
			Tickets: groups[status],
		})
	}

	runSources := make(map[string]string)
	for _, tk := range tickets {
		if tk.Assignee == "" {
			continue
		}
		if _, ok := runSources[tk.Assignee]; ok {
			continue
		}
		runSources[tk.Assignee] = h.detectRunSource(tk.Assignee)
	}

	return templates.BoardData{Columns: columns, RunSources: runSources}, nil
}

func (h *Handler) detectRunSource(runID string) string {
	events := recentEvents(h.eventsDir, event.Query{RunID: runID}, 100)
	for _, ev := range events {
		if ev.Source != "" {
			return ev.Source
		}
	}
	return ""
}
