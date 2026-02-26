package handler

import (
	"net/http"
	"sort"

	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// List renders the list view page.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildListData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ListPage(data).Render(r.Context(), w)
}

// PartialList renders just the list content for htmx swaps.
func (h *Handler) PartialList(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildListData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ListContent(data).Render(r.Context(), w)
}

func (h *Handler) buildListData(r *http.Request) (templates.ListData, error) {
	filterProject := r.URL.Query().Get("project")
	filterStatus := r.URL.Query().Get("status")

	filter := ticket.ListFilter{
		Project: filterProject,
		Status:  ticket.Status(filterStatus),
	}

	tickets, err := h.store.List(filter)
	if err != nil {
		return templates.ListData{}, err
	}

	// Collect unique project names for the filter dropdown.
	allTickets, _ := h.store.List(ticket.ListFilter{})
	projects := uniqueProjects(allTickets)

	return templates.ListData{
		Tickets:  tickets,
		Projects: projects,
		Filter: templates.ListFilter{
			Project: filterProject,
			Status:  filterStatus,
		},
	}, nil
}

func uniqueProjects(tickets []*ticket.Ticket) []string {
	seen := make(map[string]struct{})
	var projects []string
	for _, tk := range tickets {
		if tk.Project == "" {
			continue
		}
		if _, ok := seen[tk.Project]; !ok {
			seen[tk.Project] = struct{}{}
			projects = append(projects, tk.Project)
		}
	}
	sort.Strings(projects)
	return projects
}
