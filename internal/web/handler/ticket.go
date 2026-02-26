package handler

import (
	"bytes"
	"net/http"

	"github.com/boozedog/smoovtask/internal/web/templates"
	"github.com/yuin/goldmark"
)

// Ticket renders the ticket detail page.
func (h *Handler) Ticket(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildTicketData(r)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}
	templates.TicketPage(data).Render(r.Context(), w)
}

// PartialTicket renders just the ticket content for htmx swaps.
func (h *Handler) PartialTicket(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildTicketData(r)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}
	templates.TicketContent(data).Render(r.Context(), w)
}

func (h *Handler) buildTicketData(r *http.Request) (templates.TicketData, error) {
	id := r.PathValue("id")

	tk, err := h.store.Get(id)
	if err != nil {
		return templates.TicketData{}, err
	}

	// Render markdown body to HTML.
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(tk.Body), &buf); err != nil {
		// Fall back to raw body on render error.
		buf.Reset()
		buf.WriteString("<pre>" + tk.Body + "</pre>")
	}

	return templates.TicketData{
		Ticket:   tk,
		BodyHTML: buf.String(),
	}, nil
}
