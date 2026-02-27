package handler

import (
	"bytes"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/web/templates"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	goldhtml "github.com/yuin/goldmark/renderer/html"
)

// isoTimestampRe matches RFC3339/ISO timestamps like 2026-02-26T02:46:49Z or 2026-02-26T02:46:49+00:00
var isoTimestampRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})`)

// eventH2Re matches rendered h2 tags containing event type names.
var eventH2Re = regexp.MustCompile(`<h2>(Created|In Progress|Note|Review Requested|Done|Closed|Rework|Blocked|Backlog|Open)\b`)

// md is a shared goldmark instance configured with syntax highlighting and hard wraps.
var md = goldmark.New(
	goldmark.WithExtensions(
		highlighting.NewHighlighting(
			highlighting.WithStyle("dracula"),
			highlighting.WithFormatOptions(),
		),
	),
	goldmark.WithRendererOptions(
		goldhtml.WithHardWraps(),
		goldhtml.WithUnsafe(),
	),
)

// eventTypeClass maps event type names to CSS class suffixes.
var eventTypeClass = map[string]string{
	"Created":          "created",
	"In Progress":      "in-progress",
	"Note":             "note",
	"Review Requested": "review",
	"Done":             "done",
	"Closed":           "done",
	"Rework":           "rework",
	"Blocked":          "blocked",
	"Backlog":          "backlog",
	"Open":             "open",
}

// Ticket renders the ticket detail page.
func (h *Handler) Ticket(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildTicketData(r)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}
	_ = templates.TicketPage(data).Render(r.Context(), w)
}

// PartialTicket renders the ticket partial for htmx swaps.
// When the request targets the modal (HX-Target or ?modal=1), it renders
// the modal-specific partial with OOB header swap; otherwise the standard partial.
func (h *Handler) PartialTicket(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildTicketData(r)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}
	if r.Header.Get("HX-Target") == "ticket-modal-body" || r.URL.Query().Get("modal") == "1" {
		_ = templates.TicketModalPartial(data).Render(r.Context(), w)
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
	if err := md.Convert([]byte(tk.Body), &buf); err != nil {
		// Fall back to raw body on render error.
		buf.Reset()
		buf.WriteString("<pre>" + html.EscapeString(tk.Body) + "</pre>")
	}

	rendered := buf.String()

	// Reformat ISO timestamps in the rendered HTML for consistency.
	rendered = isoTimestampRe.ReplaceAllStringFunc(rendered, func(match string) string {
		t, err := time.Parse(time.RFC3339, match)
		if err != nil {
			return match
		}
		return t.Format("2006-01-02 15:04")
	})

	// Add event-type classes to h2 headers for colored styling.
	rendered = eventH2Re.ReplaceAllStringFunc(rendered, func(match string) string {
		// Extract the event type name from the match.
		name := strings.TrimPrefix(match, "<h2>")
		if cls, ok := eventTypeClass[name]; ok {
			return `<h2 class="st-event-` + cls + `">` + name
		}
		return match
	})

	return templates.TicketData{
		Ticket:   tk,
		BodyHTML: rendered,
	}, nil
}
