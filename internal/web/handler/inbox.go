package handler

import (
	"fmt"
	"net/http"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// inboxEventTypes are events that represent "agent is waiting on the user".
var inboxEventTypes = map[string]bool{
	event.HookPermissionReq: true,
	event.HookStop:          true,
	event.HookTaskCompleted: true,
	event.StatusHumanReview: true,
}

// Inbox renders the inbox page.
func (h *Handler) Inbox(w http.ResponseWriter, r *http.Request) {
	data := h.buildInboxData(r)
	_ = templates.InboxPage(data).Render(r.Context(), w)
}

// PartialInbox renders the inbox partial for htmx swaps.
func (h *Handler) PartialInbox(w http.ResponseWriter, r *http.Request) {
	data := h.buildInboxData(r)
	_ = templates.InboxPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildInboxData(r *http.Request) templates.InboxData {
	filterProject := r.URL.Query().Get("project")

	q := event.Query{
		Project: filterProject,
	}

	allEvents := recentEvents(h.eventsDir, q, 500)

	// Filter to only inbox-worthy events.
	var items []templates.InboxItem
	for _, ev := range allEvents {
		if !inboxEventTypes[ev.Event] {
			continue
		}

		subject := inboxSubject(ev)
		preview := inboxPreview(ev)

		var ticketTitle string
		if ev.Ticket != "" {
			tk, err := h.store.Get(ev.Ticket)
			if err == nil && tk != nil {
				ticketTitle = tk.Title
			}
		}

		items = append(items, templates.InboxItem{
			Event:       ev,
			Subject:     subject,
			Preview:     preview,
			TicketTitle: ticketTitle,
		})
	}

	return templates.InboxData{
		Items:          items,
		CurrentProject: filterProject,
		Projects:       h.allProjects(),
	}
}

func inboxSubject(ev event.Event) string {
	switch ev.Event {
	case event.HookPermissionReq:
		tool := toString(ev.Data["tool"])
		if tool != "" {
			return "Permission requested: " + tool
		}
		return "Permission requested"
	case event.HookStop:
		return "Agent stopped"
	case event.HookTaskCompleted:
		return "Task completed"
	case event.StatusHumanReview:
		return "Ready for human review"
	default:
		return ev.Event
	}
}

func inboxPreview(ev event.Event) string {
	if ev.Data == nil {
		return ""
	}

	// For permission requests, build a rich preview showing what's being requested.
	if ev.Event == event.HookPermissionReq {
		return permissionPreview(ev.Data)
	}

	// Try common data fields for a preview snippet.
	for _, key := range []string{"message", "command", "file_path", "pattern", "title"} {
		if v, ok := ev.Data[key]; ok {
			return truncate(toString(v), 120)
		}
	}
	return ""
}

func permissionPreview(data map[string]any) string {
	var parts []string

	// Show the command or file path being requested.
	if cmd := toString(data["command"]); cmd != "" {
		parts = append(parts, "$ "+cmd)
	}
	if fp := toString(data["file_path"]); fp != "" {
		parts = append(parts, fp)
	}
	if pat := toString(data["pattern"]); pat != "" {
		parts = append(parts, pat)
	}

	// Fall back to a message or description if present.
	if len(parts) == 0 {
		if msg := toString(data["message"]); msg != "" {
			parts = append(parts, msg)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for _, p := range parts[1:] {
		result += " — " + p
	}
	return truncate(result, 150)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return s
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
