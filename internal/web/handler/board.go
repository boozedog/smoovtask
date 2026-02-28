package handler

import (
	"net/http"
	"sort"
	"time"

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
	tickets, err := h.store.ListMeta(ticket.ListFilter{})
	if err != nil {
		return templates.BoardData{}, err
	}

	groups := groupByStatus(tickets)

	// Done column: only show tickets completed in the past 24 hours.
	cutoff := time.Now().Add(-24 * time.Hour)
	if done, ok := groups[ticket.StatusDone]; ok {
		recent := done[:0]
		for _, tk := range done {
			if tk.Updated.After(cutoff) {
				recent = append(recent, tk)
			}
		}
		groups[ticket.StatusDone] = recent
	}

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

	runIDSet := make(map[string]struct{})
	for _, tk := range tickets {
		if tk.Assignee == "" {
			continue
		}
		runIDSet[tk.Assignee] = struct{}{}
	}

	runIDs := make([]string, 0, len(runIDSet))
	for runID := range runIDSet {
		runIDs = append(runIDs, runID)
	}
	runSources := h.resolveRunSources(runIDs)

	return templates.BoardData{Columns: columns, RunSources: runSources}, nil
}
