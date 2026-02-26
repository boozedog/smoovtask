package ticket

import (
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir)
}

func testTicket(id, project string, status Status, dependsOn []string) *Ticket {
	now := time.Now().UTC()
	if dependsOn == nil {
		dependsOn = []string{}
	}
	return &Ticket{
		ID:        id,
		Title:     "Test " + id,
		Project:   project,
		Status:    status,
		Priority:  PriorityP3,
		DependsOn: dependsOn,
		Created:   now,
		Updated:   now,
		Tags:      []string{},
	}
}

func TestCheckDependencies_AllResolved(t *testing.T) {
	store := testStore(t)

	depA := testTicket("sb_aaaaaa", "proj", StatusDone, nil)
	depB := testTicket("sb_bbbbbb", "proj", StatusDone, nil)
	if err := store.Create(depA); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(depB); err != nil {
		t.Fatal(err)
	}

	tk := testTicket("sb_cccccc", "proj", StatusOpen, []string{"sb_aaaaaa", "sb_bbbbbb"})
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	unresolved, err := CheckDependencies(store, tk)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 0 {
		t.Errorf("expected 0 unresolved, got %d: %v", len(unresolved), unresolved)
	}
}

func TestCheckDependencies_SomeUnresolved(t *testing.T) {
	store := testStore(t)

	depA := testTicket("sb_aaaaaa", "proj", StatusDone, nil)
	depB := testTicket("sb_bbbbbb", "proj", StatusInProgress, nil)
	if err := store.Create(depA); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(depB); err != nil {
		t.Fatal(err)
	}

	tk := testTicket("sb_cccccc", "proj", StatusOpen, []string{"sb_aaaaaa", "sb_bbbbbb"})
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	unresolved, err := CheckDependencies(store, tk)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved, got %d: %v", len(unresolved), unresolved)
	}
	if unresolved[0] != "sb_bbbbbb" {
		t.Errorf("expected sb_bbbbbb, got %s", unresolved[0])
	}
}

func TestCheckDependencies_MissingDep(t *testing.T) {
	store := testStore(t)

	tk := testTicket("sb_cccccc", "proj", StatusOpen, []string{"sb_missing"})
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	unresolved, err := CheckDependencies(store, tk)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved, got %d", len(unresolved))
	}
	if unresolved[0] != "sb_missing" {
		t.Errorf("expected sb_missing, got %s", unresolved[0])
	}
}

func TestFindDependents(t *testing.T) {
	store := testStore(t)

	depA := testTicket("sb_aaaaaa", "proj", StatusDone, nil)
	child1 := testTicket("sb_bbbbbb", "proj", StatusOpen, []string{"sb_aaaaaa"})
	child2 := testTicket("sb_cccccc", "proj2", StatusOpen, []string{"sb_aaaaaa"})
	unrelated := testTicket("sb_dddddd", "proj", StatusOpen, nil)

	for _, tk := range []*Ticket{depA, child1, child2, unrelated} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	dependents, err := FindDependents(store, "sb_aaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if len(dependents) != 2 {
		t.Fatalf("expected 2 dependents, got %d", len(dependents))
	}

	ids := map[string]bool{}
	for _, d := range dependents {
		ids[d.ID] = true
	}
	if !ids["sb_bbbbbb"] || !ids["sb_cccccc"] {
		t.Errorf("expected sb_bbbbbb and sb_cccccc in dependents, got %v", ids)
	}
}

func TestAutoUnblock(t *testing.T) {
	store := testStore(t)

	// depA is DONE
	depA := testTicket("sb_aaaaaa", "proj", StatusDone, nil)
	if err := store.Create(depA); err != nil {
		t.Fatal(err)
	}

	// child depends on depA, is BLOCKED with PriorStatus=OPEN
	child := testTicket("sb_bbbbbb", "proj", StatusBlocked, []string{"sb_aaaaaa"})
	priorOpen := StatusOpen
	child.PriorStatus = &priorOpen
	if err := store.Create(child); err != nil {
		t.Fatal(err)
	}

	unblocked, err := AutoUnblock(store, "sb_aaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if len(unblocked) != 1 {
		t.Fatalf("expected 1 unblocked, got %d", len(unblocked))
	}
	if unblocked[0].ID != "sb_bbbbbb" {
		t.Errorf("expected sb_bbbbbb, got %s", unblocked[0].ID)
	}
	if unblocked[0].Status != StatusOpen {
		t.Errorf("expected OPEN, got %s", unblocked[0].Status)
	}
	if unblocked[0].PriorStatus != nil {
		t.Errorf("expected nil PriorStatus, got %v", unblocked[0].PriorStatus)
	}

	// Re-read from disk to verify persistence
	reloaded, err := store.Get("sb_bbbbbb")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != StatusOpen {
		t.Errorf("persisted status should be OPEN, got %s", reloaded.Status)
	}
}

func TestAutoUnblock_StillBlocked(t *testing.T) {
	store := testStore(t)

	// depA is DONE, depB is still IN-PROGRESS
	depA := testTicket("sb_aaaaaa", "proj", StatusDone, nil)
	depB := testTicket("sb_bbbbbb", "proj", StatusInProgress, nil)
	if err := store.Create(depA); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(depB); err != nil {
		t.Fatal(err)
	}

	// child depends on both, BLOCKED
	child := testTicket("sb_cccccc", "proj", StatusBlocked, []string{"sb_aaaaaa", "sb_bbbbbb"})
	priorOpen := StatusOpen
	child.PriorStatus = &priorOpen
	if err := store.Create(child); err != nil {
		t.Fatal(err)
	}

	unblocked, err := AutoUnblock(store, "sb_aaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if len(unblocked) != 0 {
		t.Errorf("expected 0 unblocked (still has unresolved dep), got %d", len(unblocked))
	}

	// Verify child is still BLOCKED on disk
	reloaded, err := store.Get("sb_cccccc")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != StatusBlocked {
		t.Errorf("should still be BLOCKED, got %s", reloaded.Status)
	}
}

func TestAutoUnblock_NoPriorStatus(t *testing.T) {
	store := testStore(t)

	depA := testTicket("sb_aaaaaa", "proj", StatusDone, nil)
	if err := store.Create(depA); err != nil {
		t.Fatal(err)
	}

	// BLOCKED but no PriorStatus — should not be unblocked
	child := testTicket("sb_bbbbbb", "proj", StatusBlocked, []string{"sb_aaaaaa"})
	if err := store.Create(child); err != nil {
		t.Fatal(err)
	}

	unblocked, err := AutoUnblock(store, "sb_aaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if len(unblocked) != 0 {
		t.Errorf("expected 0 unblocked (no PriorStatus), got %d", len(unblocked))
	}
}

func TestAutoUnblock_CrossProject(t *testing.T) {
	store := testStore(t)

	depA := testTicket("sb_aaaaaa", "proj-a", StatusDone, nil)
	if err := store.Create(depA); err != nil {
		t.Fatal(err)
	}

	// Different project depends on depA
	child := testTicket("sb_bbbbbb", "proj-b", StatusBlocked, []string{"sb_aaaaaa"})
	priorOpen := StatusOpen
	child.PriorStatus = &priorOpen
	if err := store.Create(child); err != nil {
		t.Fatal(err)
	}

	unblocked, err := AutoUnblock(store, "sb_aaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if len(unblocked) != 1 {
		t.Fatalf("expected 1 unblocked (cross-project), got %d", len(unblocked))
	}
	if unblocked[0].ID != "sb_bbbbbb" {
		t.Errorf("expected sb_bbbbbb, got %s", unblocked[0].ID)
	}
}
