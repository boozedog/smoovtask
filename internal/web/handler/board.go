package handler

import (
	"net/http"
	"sort"
	"time"

	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

func reviewTicketRank(status ticket.Status) int {
	switch status {
	case ticket.StatusReview:
		return 0
	case ticket.StatusHumanReview:
		return 1
	default:
		return 2
	}
}

func reviewTicketLess(a, b *ticket.Ticket) bool {
	aRank := reviewTicketRank(a.Status)
	bRank := reviewTicketRank(b.Status)
	if aRank != bRank {
		return aRank < bRank
	}

	aAssigned := a.Assignee != ""
	bAssigned := b.Assignee != ""
	if aAssigned != bAssigned {
		return aAssigned
	}

	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}

	return a.Created.Before(b.Created)
}

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

	// Sort tickets within each raw status bucket.
	for status, tks := range groups {
		if status == ticket.StatusDone || status == ticket.StatusCancelled {
			// Done/Cancelled: reverse chronological by Updated.
			sort.Slice(tks, func(i, j int) bool {
				return tks[i].Updated.After(tks[j].Updated)
			})
		} else if status == ticket.StatusReview || status == ticket.StatusHumanReview {
			// Review buckets: agent review first, then human review.
			// Within each, tickets with an active assignee first, then priority ascending,
			// then creation date ascending.
			sort.Slice(tks, func(i, j int) bool {
				return reviewTicketLess(tks[i], tks[j])
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
		columnTickets := append([]*ticket.Ticket{}, groups[status]...)
		if status == ticket.StatusOpen {
			columnTickets = append(columnTickets, groups[ticket.StatusRework]...)
		} else if status == ticket.StatusReview {
			columnTickets = append(columnTickets, groups[ticket.StatusHumanReview]...)
		}

		if status == ticket.StatusDone || status == ticket.StatusCancelled {
			sort.Slice(columnTickets, func(i, j int) bool {
				return columnTickets[i].Updated.After(columnTickets[j].Updated)
			})
		} else if status == ticket.StatusReview {
			sort.Slice(columnTickets, func(i, j int) bool {
				return reviewTicketLess(columnTickets[i], columnTickets[j])
			})
		} else {
			sort.Slice(columnTickets, func(i, j int) bool {
				if columnTickets[i].Priority != columnTickets[j].Priority {
					return columnTickets[i].Priority < columnTickets[j].Priority
				}
				return columnTickets[i].Created.Before(columnTickets[j].Created)
			})
		}

		columns = append(columns, templates.BoardColumn{
			Status:  status,
			Tickets: columnTickets,
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
