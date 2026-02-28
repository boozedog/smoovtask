package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

func (h *Handler) NewTicket(w http.ResponseWriter, r *http.Request) {
	data := templates.TicketFormData{
		Mode: "new",
		Values: templates.TicketFormValues{
			Project:  h.project,
			Status:   string(ticket.StatusOpen),
			Priority: string(ticket.DefaultPriority),
		},
	}
	_ = templates.TicketFormPage(data).Render(r.Context(), w)
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, r, "new", "", "invalid form submission")
		return
	}

	values := h.formValuesFromRequest(r)
	if err := validateFormValues(values); err != nil {
		h.renderFormErrorWithValues(w, r, "new", "", values, err.Error())
		return
	}

	now := time.Now().UTC()
	tk := &ticket.Ticket{
		Title:     values.Title,
		Project:   values.Project,
		Status:    ticket.Status(values.Status),
		Priority:  ticket.Priority(values.Priority),
		DependsOn: splitCSV(values.DependsOn),
		Tags:      splitCSV(values.Tags),
		Created:   now,
		Updated:   now,
	}
	if tk.DependsOn == nil {
		tk.DependsOn = []string{}
	}
	if tk.Tags == nil {
		tk.Tags = []string{}
	}

	body := values.Description
	if body == "" {
		body = values.Title
	}
	ticket.AppendSection(tk, "Created", "web", "", body, nil, now)

	if err := h.store.Create(tk); err != nil {
		h.renderFormErrorWithValues(w, r, "new", "", values, "failed to create ticket")
		return
	}

	el := event.NewEventLog(h.eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   event.TicketCreated,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   "web",
		Data: map[string]any{
			"title":    tk.Title,
			"priority": string(tk.Priority),
		},
	})

	http.Redirect(w, r, fmt.Sprintf("/ticket/%s", tk.ID), http.StatusSeeOther)
}

func (h *Handler) EditTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tk, err := h.store.Get(id)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	data := templates.TicketFormData{
		Mode:     "edit",
		TicketID: tk.ID,
		Values: templates.TicketFormValues{
			Title:     tk.Title,
			Project:   tk.Project,
			Status:    string(tk.Status),
			Priority:  string(tk.Priority),
			DependsOn: strings.Join(tk.DependsOn, ","),
			Tags:      strings.Join(tk.Tags, ","),
		},
	}
	_ = templates.TicketFormPage(data).Render(r.Context(), w)
}

func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tk, err := h.store.Get(id)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, r, "edit", tk.ID, "invalid form submission")
		return
	}

	values := h.formValuesFromRequest(r)
	if err := validateFormValues(values); err != nil {
		h.renderFormErrorWithValues(w, r, "edit", tk.ID, values, err.Error())
		return
	}

	now := time.Now().UTC()
	oldStatus := tk.Status

	tk.Title = values.Title
	tk.Project = values.Project
	tk.Status = ticket.Status(values.Status)
	tk.Priority = ticket.Priority(values.Priority)
	tk.DependsOn = splitCSV(values.DependsOn)
	tk.Tags = splitCSV(values.Tags)
	tk.Updated = now

	if values.Description != "" {
		ticket.AppendSection(tk, "Edited", "web", "", values.Description, nil, now)
	}

	if err := h.store.Save(tk); err != nil {
		h.renderFormErrorWithValues(w, r, "edit", tk.ID, values, "failed to save ticket")
		return
	}

	el := event.NewEventLog(h.eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   event.TicketNote,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   "web",
		Data: map[string]any{
			"message": "ticket edited in web view",
		},
	})
	if oldStatus != tk.Status {
		_ = el.Append(event.Event{
			TS:      now,
			Event:   "status." + strings.ToLower(string(tk.Status)),
			Ticket:  tk.ID,
			Project: tk.Project,
			Actor:   "web",
			Data: map[string]any{
				"from": string(oldStatus),
			},
		})
	}

	http.Redirect(w, r, fmt.Sprintf("/ticket/%s", tk.ID), http.StatusSeeOther)
}

func (h *Handler) formValuesFromRequest(r *http.Request) templates.TicketFormValues {
	return templates.TicketFormValues{
		Title:       strings.TrimSpace(r.FormValue("title")),
		Project:     strings.TrimSpace(r.FormValue("project")),
		Status:      strings.TrimSpace(r.FormValue("status")),
		Priority:    strings.TrimSpace(r.FormValue("priority")),
		DependsOn:   strings.TrimSpace(r.FormValue("depends_on")),
		Tags:        strings.TrimSpace(r.FormValue("tags")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}
}

func validateFormValues(v templates.TicketFormValues) error {
	if v.Title == "" {
		return fmt.Errorf("title is required")
	}
	if v.Project == "" {
		return fmt.Errorf("project is required")
	}
	if !ticket.ValidStatuses[ticket.Status(v.Status)] {
		return fmt.Errorf("invalid status")
	}
	if !ticket.ValidPriorities[ticket.Priority(v.Priority)] {
		return fmt.Errorf("invalid priority")
	}
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func (h *Handler) renderFormError(w http.ResponseWriter, r *http.Request, mode, ticketID, msg string) {
	h.renderFormErrorWithValues(w, r, mode, ticketID, h.formValuesFromRequest(r), msg)
}

func (h *Handler) renderFormErrorWithValues(w http.ResponseWriter, r *http.Request, mode, ticketID string, values templates.TicketFormValues, msg string) {
	if mode == "new" && values.Project == "" {
		values.Project = h.project
	}
	data := templates.TicketFormData{
		Mode:     mode,
		TicketID: ticketID,
		Values:   values,
		Error:    msg,
	}
	w.WriteHeader(http.StatusBadRequest)
	_ = templates.TicketFormPage(data).Render(r.Context(), w)
}
