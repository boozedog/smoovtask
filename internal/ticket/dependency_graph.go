package ticket

import (
	"encoding/json"
	"sort"
)

// GraphNode represents a ticket in the dependency graph, placed at a specific
// layer and position within that layer.
type GraphNode struct {
	ID       string
	Layer    int
	Position int
}

// GraphEdge represents a dependency edge from one ticket to another.
// Direction: FromID depends on ToID (arrow points FromID → ToID).
type GraphEdge struct {
	FromID string
	ToID   string
}

// DependencyGraph is a layered DAG of ticket dependencies.
// Layers flow left→right: layer 0 contains root tickets (nothing depends on
// them), and deeper layers contain their transitive dependencies.
type DependencyGraph struct {
	Layers [][]GraphNode
	Edges  []GraphEdge
	Empty  bool
}

// EdgesJSON returns the edges as a JSON array for use in the template's
// inline script.
func (g DependencyGraph) EdgesJSON() string {
	type edgeJSON struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	out := make([]edgeJSON, len(g.Edges))
	for i, e := range g.Edges {
		out[i] = edgeJSON{From: e.FromID, To: e.ToID}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// BuildDependencyGraph computes a layered dependency graph from the given
// tickets. It excludes DONE and CANCELLED tickets, and only includes tickets
// that participate in at least one dependency relationship.
func BuildDependencyGraph(tickets []*Ticket) DependencyGraph {
	// Filter out completed tickets.
	nodes := make(map[string]*Ticket)
	for _, tk := range tickets {
		if tk.Status == StatusDone || tk.Status == StatusCancelled {
			continue
		}
		nodes[tk.ID] = tk
	}

	// Build adjacency: "depends on" edges where both ends exist.
	// forward: id → list of IDs it depends on (children in the graph)
	// reverse: id → list of IDs that depend on it (parents in the graph)
	forward := make(map[string][]string)
	reverse := make(map[string][]string)
	participates := make(map[string]bool)

	for _, tk := range nodes {
		for _, depID := range tk.DependsOn {
			if _, ok := nodes[depID]; !ok {
				continue
			}
			forward[tk.ID] = append(forward[tk.ID], depID)
			reverse[depID] = append(reverse[depID], tk.ID)
			participates[tk.ID] = true
			participates[depID] = true
		}
	}

	if len(participates) == 0 {
		return DependencyGraph{Empty: true}
	}

	// Sort adjacency lists for deterministic output.
	for id := range forward {
		sort.Strings(forward[id])
	}
	for id := range reverse {
		sort.Strings(reverse[id])
	}

	// Compute layers using longest-path layering (modified Kahn's algorithm).
	// Leaves (layer 0) are tickets that depend on nothing (no entries in forward).
	// Tickets that depend on them go to deeper layers (left→right = dependency→dependant).
	layer := make(map[string]int)
	remaining := make(map[string]bool)
	for id := range participates {
		remaining[id] = true
	}

	currentLayer := 0
	for len(remaining) > 0 {
		// Find leaves for this iteration: nodes with no dependencies among
		// remaining nodes.
		var leaves []string
		for id := range remaining {
			deg := 0
			for _, depID := range forward[id] {
				if remaining[depID] {
					deg++
				}
			}
			if deg == 0 {
				leaves = append(leaves, id)
			}
		}

		// Cycle: no natural leaves. Break by picking the first remaining node.
		if len(leaves) == 0 {
			var ids []string
			for id := range remaining {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			leaves = []string{ids[0]}
		}

		sort.Strings(leaves)

		// Assign longest-path layers: each node goes to max(dep layers) + 1,
		// but leaves of this batch go to currentLayer.
		for _, id := range leaves {
			maxDep := -1
			for _, depID := range forward[id] {
				if l, ok := layer[depID]; ok && l > maxDep {
					maxDep = l
				}
			}
			layer[id] = max(maxDep+1, currentLayer)
			delete(remaining, id)
		}

		currentLayer++
	}

	// Push nodes to their longest-path layer: each node should be at
	// max(dependency layers) + 1. Iterate in topological order.
	sorted := topologicalSort(participates, forward)
	for _, id := range sorted {
		maxDep := -1
		for _, depID := range forward[id] {
			if l, ok := layer[depID]; ok && l > maxDep {
				maxDep = l
			}
		}
		if maxDep+1 > layer[id] {
			layer[id] = maxDep + 1
		}
	}

	// Build layer buckets.
	maxLayer := 0
	for _, l := range layer {
		if l > maxLayer {
			maxLayer = l
		}
	}

	layerBuckets := make([][]string, maxLayer+1)
	for id, l := range layer {
		layerBuckets[l] = append(layerBuckets[l], id)
	}

	// Sort each layer alphabetically as initial ordering.
	for i := range layerBuckets {
		sort.Strings(layerBuckets[i])
	}

	// Barycenter ordering to reduce edge crossings.
	// Run a few iterations, sweeping forward and backward.
	for range 4 {
		// Forward sweep.
		for l := 1; l < len(layerBuckets); l++ {
			barycenterOrder(layerBuckets[l], layerBuckets[l-1], forward)
		}
		// Backward sweep.
		for l := len(layerBuckets) - 2; l >= 0; l-- {
			barycenterOrder(layerBuckets[l], layerBuckets[l+1], reverse)
		}
	}

	// Build result.
	layers := make([][]GraphNode, len(layerBuckets))
	for l, ids := range layerBuckets {
		layers[l] = make([]GraphNode, len(ids))
		for p, id := range ids {
			layers[l][p] = GraphNode{ID: id, Layer: l, Position: p}
		}
	}

	var edges []GraphEdge
	for id, deps := range forward {
		for _, depID := range deps {
			edges = append(edges, GraphEdge{FromID: id, ToID: depID})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID < edges[j].FromID
		}
		return edges[i].ToID < edges[j].ToID
	})

	return DependencyGraph{
		Layers: layers,
		Edges:  edges,
		Empty:  false,
	}
}

// topologicalSort returns IDs in topological order based on the given adjacency
// (predecessors before successors).
func topologicalSort(nodes map[string]bool, adjacency map[string][]string) []string {
	visited := make(map[string]bool)
	var order []string

	var visit func(string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, parentID := range adjacency[id] {
			if nodes[parentID] {
				visit(parentID)
			}
		}
		order = append(order, id)
	}

	// Visit in sorted order for determinism.
	var ids []string
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		visit(id)
	}
	return order
}

// barycenterOrder sorts the nodes in targetLayer based on the average position
// of their neighbors in refLayer.
func barycenterOrder(targetLayer, refLayer []string, adjacency map[string][]string) {
	if len(targetLayer) <= 1 {
		return
	}

	// Build position index for reference layer.
	refPos := make(map[string]int)
	for i, id := range refLayer {
		refPos[id] = i
	}

	// Compute barycenter for each node in target layer.
	bary := make(map[string]float64)
	for _, id := range targetLayer {
		neighbors := adjacency[id]
		sum := 0.0
		count := 0
		for _, nID := range neighbors {
			if pos, ok := refPos[nID]; ok {
				sum += float64(pos)
				count++
			}
		}
		if count > 0 {
			bary[id] = sum / float64(count)
		} else {
			bary[id] = -1 // Keep original position.
		}
	}

	sort.SliceStable(targetLayer, func(i, j int) bool {
		bi, bj := bary[targetLayer[i]], bary[targetLayer[j]]
		if bi < 0 && bj < 0 {
			return false // Both unconnected, keep original order.
		}
		if bi < 0 {
			return false
		}
		if bj < 0 {
			return true
		}
		return bi < bj
	})
}
