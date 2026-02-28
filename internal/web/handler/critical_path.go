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
	scope := r.URL.Query().Get("scope")
	if scope != "current" {
		scope = "all"
	}

	filter := ticket.ListFilter{}
	if scope == "current" {
		filter.Project = h.project
	}

	all, err := h.store.List(filter)
	if err != nil {
		return templates.CriticalPathData{}, err
	}

	view := r.URL.Query().Get("view")
	if view != "horizontal" {
		view = "vertical"
	}

	paths := ticket.ComputeCriticalPaths(all, 8)
	filtered := make([]ticket.CriticalPath, 0, len(paths))
	for _, path := range paths {
		if len(path.IDs) > 1 {
			filtered = append(filtered, path)
		}
	}
	byID := make(map[string]*ticket.Ticket)
	runSources := make(map[string]string)
	for _, tk := range all {
		byID[tk.ID] = tk
		if tk.Assignee != "" {
			if _, ok := runSources[tk.Assignee]; !ok {
				runSources[tk.Assignee] = h.detectRunSource(tk.Assignee)
			}
		}
	}

	return templates.CriticalPathData{
		Project:    h.project,
		Scope:      scope,
		View:       view,
		Paths:      filtered,
		ByID:       byID,
		RunSources: runSources,
	}, nil
}
