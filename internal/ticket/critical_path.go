package ticket

import (
	"sort"
	"strings"
)

// CriticalPath is an ordered dependency chain of ticket IDs.
type CriticalPath struct {
	IDs []string
}

// ComputeCriticalPaths returns up to limit longest dependency chains among
// non-DONE tickets. Chains are project-agnostic; callers should pre-filter.
//
// Unlike a single longest-path calculation, this returns branched chains too
// (e.g. A depends on B and C yields both A->B and A->C paths).
func ComputeCriticalPaths(tickets []*Ticket, limit int) []CriticalPath {
	if limit <= 0 {
		limit = 5
	}

	nodes := make(map[string]*Ticket)
	dependedBy := make(map[string]int)
	for _, tk := range tickets {
		if tk.Status == StatusDone {
			continue
		}
		nodes[tk.ID] = tk
	}

	for _, tk := range nodes {
		for _, depID := range tk.DependsOn {
			if _, ok := nodes[depID]; ok {
				dependedBy[depID]++
			}
		}
	}

	if len(nodes) == 0 {
		return nil
	}

	var roots []string
	for id := range nodes {
		if dependedBy[id] == 0 {
			roots = append(roots, id)
		}
	}
	if len(roots) == 0 {
		for id := range nodes {
			roots = append(roots, id)
		}
	}

	sort.Strings(roots)

	seen := make(map[string]struct{})
	var paths []CriticalPath

	var enumerate func(id string, visiting map[string]bool, chain []string)
	enumerate = func(id string, visiting map[string]bool, chain []string) {
		if visiting[id] {
			return
		}
		visiting[id] = true

		tk := nodes[id]
		deps := make([]string, 0, len(tk.DependsOn))
		for _, depID := range tk.DependsOn {
			if _, ok := nodes[depID]; ok && !visiting[depID] {
				deps = append(deps, depID)
			}
		}
		sort.Strings(deps)

		if len(deps) == 0 {
			key := strings.Join(chain, "->")
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				ids := append([]string(nil), chain...)
				paths = append(paths, CriticalPath{IDs: ids})
			}
			visiting[id] = false
			return
		}

		for _, depID := range deps {
			next := append(append([]string(nil), chain...), depID)
			enumerate(depID, visiting, next)
		}

		visiting[id] = false
	}

	for _, id := range roots {
		enumerate(id, map[string]bool{}, []string{id})
	}

	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i].IDs) != len(paths[j].IDs) {
			return len(paths[i].IDs) > len(paths[j].IDs)
		}
		return strings.Join(paths[i].IDs, "->") < strings.Join(paths[j].IDs, "->")
	})

	if len(paths) > limit {
		paths = paths[:limit]
	}
	return paths
}
