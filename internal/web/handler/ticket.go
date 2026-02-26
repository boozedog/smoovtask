package handler

import (
	"bytes"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/web/templates"
	"github.com/yuin/goldmark"
)

// isoTimestampRe matches RFC3339/ISO timestamps like 2026-02-26T02:46:49Z or 2026-02-26T02:46:49+00:00
var isoTimestampRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})`)

// Ticket renders the ticket detail page.
func (h *Handler) Ticket(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildTicketData(r)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}
	_ = templates.TicketPage(data).Render(r.Context(), w)
}

// PartialTicket renders the ticket partial (with SSE self-refresh wrapper) for htmx swaps.
func (h *Handler) PartialTicket(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildTicketData(r)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}
	_ = templates.TicketPartial(data).Render(r.Context(), w)
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
		buf.WriteString("<pre>" + html.EscapeString(tk.Body) + "</pre>")
	}

	// Reformat ISO timestamps in the rendered HTML for consistency.
	html := isoTimestampRe.ReplaceAllStringFunc(buf.String(), func(match string) string {
		t, err := time.Parse(time.RFC3339, match)
		if err != nil {
			return match
		}
		return t.Format("2006-01-02 15:04")
	})

	backURL, backName := backNav(r)

	return templates.TicketData{
		Ticket:   tk,
		BodyHTML: html,
		BackURL:  backURL,
		BackName: backName,
	}, nil
}

// backNav determines the back navigation target from the Referer header or query param.
func backNav(r *http.Request) (string, string) {
	// Explicit query param takes priority.
	if from := r.URL.Query().Get("from"); from == "list" {
		return "/list", "List"
	}

	// Fall back to Referer header.
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil {
			if strings.HasPrefix(u.Path, "/list") || strings.HasPrefix(u.Path, "/partials/list") {
				return "/list", "List"
			}
		}
	}

	return "/", "Board"
}
