package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/boozedog/smoovtask/internal/ticket"
)

type searchTicket struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Project  string    `json:"project"`
	Status   string    `json:"status"`
	Priority string    `json:"priority"`
	Updated  time.Time `json:"updated"`
}

// SearchTickets returns all ticket metadata as JSON for client-side search.
func (h *Handler) SearchTickets(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.store.ListMeta(ticket.ListFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]searchTicket, len(tickets))
	for i, tk := range tickets {
		result[i] = searchTicket{
			ID:       tk.ID,
			Title:    tk.Title,
			Project:  tk.Project,
			Status:   string(tk.Status),
			Priority: string(tk.Priority),
			Updated:  tk.Updated,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	_ = json.NewEncoder(w).Encode(result)
}
