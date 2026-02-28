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
	_ = templates.ListPage(data).Render(r.Context(), w)
}

// PartialList renders the list partial (with filters + SSE self-refresh wrapper) for htmx swaps.
func (h *Handler) PartialList(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildListData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.ListPartial(data).Render(r.Context(), w)
}

// PartialListContent renders just the list table content for filter/sort swaps.
func (h *Handler) PartialListContent(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildListData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Push canonical /list URL so filters are bookmarkable.
	pushURL := "/list"
	params := r.URL.Query()
	for k := range params {
		if params.Get(k) == "" {
			params.Del(k)
		}
	}
	if q := params.Encode(); q != "" {
		pushURL += "?" + q
	}
	w.Header().Set("HX-Push-Url", pushURL)

	_ = templates.ListContent(data).Render(r.Context(), w)
}

func (h *Handler) buildListData(r *http.Request) (templates.ListData, error) {
	filterProject := r.URL.Query().Get("project")
	filterStatus := r.URL.Query().Get("status")

	filter := ticket.ListFilter{
		Project: filterProject,
		Status:  ticket.Status(filterStatus),
	}

	tickets, err := h.store.ListMeta(filter)
	if err != nil {
		return templates.ListData{}, err
	}

	sortField := r.URL.Query().Get("sort")
	sortDir := r.URL.Query().Get("dir")
	if sortDir == "" {
		sortDir = "asc"
	}

	// Apply sort based on field.
	switch sortField {
	case "id":
		sort.Slice(tickets, func(i, j int) bool {
			if sortDir == "desc" {
				return tickets[i].ID > tickets[j].ID
			}
			return tickets[i].ID < tickets[j].ID
		})
	case "title":
		sort.Slice(tickets, func(i, j int) bool {
			if sortDir == "desc" {
				return tickets[i].Title > tickets[j].Title
			}
			return tickets[i].Title < tickets[j].Title
		})
	case "status":
		sort.Slice(tickets, func(i, j int) bool {
			wi, wj := statusWeight(tickets[i].Status), statusWeight(tickets[j].Status)
			if wi != wj {
				if sortDir == "desc" {
					return wi > wj
				}
				return wi < wj
			}
			if sortDir == "desc" {
				return tickets[i].Updated.Before(tickets[j].Updated)
			}
			return tickets[i].Updated.After(tickets[j].Updated)
		})
	case "priority":
		sort.Slice(tickets, func(i, j int) bool {
			if sortDir == "desc" {
				return tickets[i].Priority > tickets[j].Priority
			}
			return tickets[i].Priority < tickets[j].Priority
		})
	case "project":
		sort.Slice(tickets, func(i, j int) bool {
			if sortDir == "desc" {
				return tickets[i].Project > tickets[j].Project
			}
			return tickets[i].Project < tickets[j].Project
		})
	case "updated":
		sort.Slice(tickets, func(i, j int) bool {
			if sortDir == "desc" {
				return tickets[i].Updated.Before(tickets[j].Updated)
			}
			return tickets[i].Updated.After(tickets[j].Updated)
		})
	default:
		// Default sort: active tickets first, then by updated desc.
		sort.Slice(tickets, func(i, j int) bool {
			wi, wj := statusWeight(tickets[i].Status), statusWeight(tickets[j].Status)
			if wi != wj {
				return wi < wj
			}
			return tickets[i].Updated.After(tickets[j].Updated)
		})
	}

	// Collect unique project names for the filter dropdown.
	allTickets, _ := h.store.ListMeta(ticket.ListFilter{})
	projects := uniqueProjects(allTickets)

	return templates.ListData{
		Tickets:  tickets,
		Projects: projects,
		Filter: templates.ListFilter{
			Project: filterProject,
			Status:  filterStatus,
			Sort:    sortField,
			Dir:     sortDir,
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
