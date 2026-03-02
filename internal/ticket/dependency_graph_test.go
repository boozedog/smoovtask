package ticket

import (
	"encoding/json"
	"testing"
)

func TestBuildDependencyGraphLinearChain(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusOpen, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen, DependsOn: []string{"st_c"}},
		{ID: "st_c", Status: StatusOpen},
	}

	g := BuildDependencyGraph(tickets)
	if g.Empty {
		t.Fatal("expected non-empty graph")
	}
	if len(g.Layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(g.Layers))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(g.Edges))
	}

	// st_c at layer 0 (no deps), st_b at 1, st_a at 2 (depends on chain).
	layerOf := nodeLayerMap(g)
	if layerOf["st_c"] != 0 {
		t.Errorf("expected st_c at layer 0, got %d", layerOf["st_c"])
	}
	if layerOf["st_b"] != 1 {
		t.Errorf("expected st_b at layer 1, got %d", layerOf["st_b"])
	}
	if layerOf["st_a"] != 2 {
		t.Errorf("expected st_a at layer 2, got %d", layerOf["st_a"])
	}
}

func TestBuildDependencyGraphBranching(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_root", Status: StatusOpen, DependsOn: []string{"st_a", "st_b"}},
		{ID: "st_a", Status: StatusOpen},
		{ID: "st_b", Status: StatusOpen},
	}

	g := BuildDependencyGraph(tickets)
	if g.Empty {
		t.Fatal("expected non-empty graph")
	}
	if len(g.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(g.Layers))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(g.Edges))
	}

	// Layer 0 should have st_a and st_b (no deps), layer 1 should have st_root.
	layerOf := nodeLayerMap(g)
	if layerOf["st_a"] != 0 {
		t.Errorf("expected st_a at layer 0, got %d", layerOf["st_a"])
	}
	if layerOf["st_b"] != 0 {
		t.Errorf("expected st_b at layer 0, got %d", layerOf["st_b"])
	}
	if layerOf["st_root"] != 1 {
		t.Errorf("expected st_root at layer 1, got %d", layerOf["st_root"])
	}
}

func TestBuildDependencyGraphExcludesDone(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusDone, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen},
	}

	g := BuildDependencyGraph(tickets)
	// st_b has no remaining dependencies, so it doesn't participate.
	if !g.Empty {
		t.Fatal("expected empty graph when DONE ticket is excluded")
	}
}

func TestBuildDependencyGraphExcludesCancelled(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusCancelled, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen},
	}

	g := BuildDependencyGraph(tickets)
	if !g.Empty {
		t.Fatal("expected empty graph when CANCELLED ticket is excluded")
	}
}

func TestBuildDependencyGraphExcludesIsolated(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusOpen, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen},
		{ID: "st_c", Status: StatusOpen}, // No deps, not depended on.
	}

	g := BuildDependencyGraph(tickets)
	if g.Empty {
		t.Fatal("expected non-empty graph")
	}

	// st_c should not appear.
	layerOf := nodeLayerMap(g)
	if _, ok := layerOf["st_c"]; ok {
		t.Fatal("expected st_c to be excluded (no dependencies)")
	}
	if _, ok := layerOf["st_a"]; !ok {
		t.Fatal("expected st_a to be included")
	}
	if _, ok := layerOf["st_b"]; !ok {
		t.Fatal("expected st_b to be included")
	}
}

func TestBuildDependencyGraphCycle(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusOpen, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen, DependsOn: []string{"st_a"}},
	}

	g := BuildDependencyGraph(tickets)
	if g.Empty {
		t.Fatal("expected non-empty graph even with cycle")
	}
	// Both should appear somewhere; the algorithm should not hang.
	layerOf := nodeLayerMap(g)
	if _, ok := layerOf["st_a"]; !ok {
		t.Fatal("expected st_a in graph")
	}
	if _, ok := layerOf["st_b"]; !ok {
		t.Fatal("expected st_b in graph")
	}
}

func TestBuildDependencyGraphEmpty(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusOpen},
		{ID: "st_b", Status: StatusOpen},
	}

	g := BuildDependencyGraph(tickets)
	if !g.Empty {
		t.Fatal("expected empty graph when no dependencies exist")
	}
}

func TestBuildDependencyGraphDiamond(t *testing.T) {
	// Diamond: A depends on B and C, both depend on D.
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusOpen, DependsOn: []string{"st_b", "st_c"}},
		{ID: "st_b", Status: StatusOpen, DependsOn: []string{"st_d"}},
		{ID: "st_c", Status: StatusOpen, DependsOn: []string{"st_d"}},
		{ID: "st_d", Status: StatusOpen},
	}

	g := BuildDependencyGraph(tickets)
	if g.Empty {
		t.Fatal("expected non-empty graph")
	}
	if len(g.Layers) != 3 {
		t.Fatalf("expected 3 layers for diamond, got %d", len(g.Layers))
	}

	layerOf := nodeLayerMap(g)
	if layerOf["st_d"] != 0 {
		t.Errorf("expected st_d at layer 0, got %d", layerOf["st_d"])
	}
	if layerOf["st_b"] != 1 {
		t.Errorf("expected st_b at layer 1, got %d", layerOf["st_b"])
	}
	if layerOf["st_c"] != 1 {
		t.Errorf("expected st_c at layer 1, got %d", layerOf["st_c"])
	}
	if layerOf["st_a"] != 2 {
		t.Errorf("expected st_a at layer 2, got %d", layerOf["st_a"])
	}
}

func TestEdgesJSON(t *testing.T) {
	g := DependencyGraph{
		Edges: []GraphEdge{
			{FromID: "st_a", ToID: "st_b"},
		},
	}
	s := g.EdgesJSON()
	var parsed []struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("failed to parse EdgesJSON: %v", err)
	}
	if len(parsed) != 1 || parsed[0].From != "st_a" || parsed[0].To != "st_b" {
		t.Fatalf("unexpected EdgesJSON result: %s", s)
	}
}

// nodeLayerMap builds a map of node ID → layer from the graph.
func nodeLayerMap(g DependencyGraph) map[string]int {
	m := make(map[string]int)
	for _, layer := range g.Layers {
		for _, node := range layer {
			m[node.ID] = node.Layer
		}
	}
	return m
}
