package handler

import (
	"net/http"

	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

func (h *Handler) CriticalPath(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildCriticalPathData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.CriticalPathPage(data).Render(r.Context(), w)
}

func (h *Handler) PartialCriticalPath(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildCriticalPathData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.CriticalPathPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildCriticalPathData(r *http.Request) (templates.CriticalPathData, error) {
	filterProject := r.URL.Query().Get("project")

	all, err := h.store.ListMeta(ticket.ListFilter{Project: filterProject})
	if err != nil {
		return templates.CriticalPathData{}, err
	}

	graph := ticket.BuildDependencyGraph(all)

	byID := make(map[string]*ticket.Ticket)
	runIDSet := make(map[string]struct{})
	for _, tk := range all {
		byID[tk.ID] = tk
		if tk.Assignee != "" {
			runIDSet[tk.Assignee] = struct{}{}
		}
	}

	runIDs := make([]string, 0, len(runIDSet))
	for runID := range runIDSet {
		runIDs = append(runIDs, runID)
	}
	runSources := h.resolveRunSources(runIDs)

	return templates.CriticalPathData{
		Project:        h.project,
		Graph:          graph,
		ByID:           byID,
		RunSources:     runSources,
		CurrentProject: filterProject,
		Projects:       h.allProjects(),
	}, nil
}
