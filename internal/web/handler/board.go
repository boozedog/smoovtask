package handler

import (
	"net/http"

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

// PartialBoard renders just the board content for htmx swaps.
func (h *Handler) PartialBoard(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildBoardData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.BoardContent(data).Render(r.Context(), w)
}

func (h *Handler) buildBoardData() (templates.BoardData, error) {
	tickets, err := h.store.List(ticket.ListFilter{})
	if err != nil {
		return templates.BoardData{}, err
	}

	groups := groupByStatus(tickets)

	var columns []templates.BoardColumn
	for _, status := range statusOrder {
		columns = append(columns, templates.BoardColumn{
			Status:  status,
			Tickets: groups[status],
		})
	}

	return templates.BoardData{Columns: columns}, nil
}
