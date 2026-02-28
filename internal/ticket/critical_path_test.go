package ticket

import "testing"

func TestComputeCriticalPathsLongestFirst(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusOpen, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen, DependsOn: []string{"st_c"}},
		{ID: "st_c", Status: StatusOpen, DependsOn: []string{}},
		{ID: "st_x", Status: StatusOpen, DependsOn: []string{"st_y"}},
		{ID: "st_y", Status: StatusOpen, DependsOn: []string{}},
	}

	paths := ComputeCriticalPaths(tickets, 5)
	if len(paths) == 0 {
		t.Fatal("expected at least one path")
	}
	if got := len(paths[0].IDs); got != 3 {
		t.Fatalf("expected longest path length 3, got %d", got)
	}
	if paths[0].IDs[0] != "st_a" {
		t.Fatalf("expected first path root st_a, got %s", paths[0].IDs[0])
	}
}

func TestComputeCriticalPathsExcludesDone(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_a", Status: StatusDone, DependsOn: []string{"st_b"}},
		{ID: "st_b", Status: StatusOpen, DependsOn: []string{}},
	}

	paths := ComputeCriticalPaths(tickets, 5)
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if len(paths[0].IDs) != 1 || paths[0].IDs[0] != "st_b" {
		t.Fatalf("unexpected path: %#v", paths[0].IDs)
	}
}

func TestComputeCriticalPathsIncludesBranchedDependencies(t *testing.T) {
	tickets := []*Ticket{
		{ID: "st_root", Status: StatusOpen, DependsOn: []string{"st_a", "st_b"}},
		{ID: "st_a", Status: StatusOpen, DependsOn: []string{}},
		{ID: "st_b", Status: StatusOpen, DependsOn: []string{}},
	}

	paths := ComputeCriticalPaths(tickets, 10)
	seen := map[string]bool{}
	for _, p := range paths {
		if len(p.IDs) != 2 {
			continue
		}
		seen[p.IDs[0]+"->"+p.IDs[1]] = true
	}

	if !seen["st_root->st_a"] {
		t.Fatal("expected root->a chain")
	}
	if !seen["st_root->st_b"] {
		t.Fatal("expected root->b chain")
	}
}
