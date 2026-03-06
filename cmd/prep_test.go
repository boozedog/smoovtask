package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestPrep_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}

	runGitCmd(t, wd, "branch", "-M", "master")

	tk := env.createTicket(t, "test prep feature", ticket.StatusInProgress)
	tk.Assignee = "test-session-prep"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	workBranch := "st/" + tk.ID
	runGitCmd(t, wd, "checkout", "-b", workBranch)

	testFile := filepath.Join(wd, "feature.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write feature.go: %v", err)
	}
	runGitCmd(t, wd, "add", "feature.go")
	runGitCmd(t, wd, "commit", "-m", "add feature")

	runGitCmd(t, wd, "checkout", "master")

	out, err := env.runCmd(t, "--human", "prep", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, tk.ID) {
		t.Errorf("output missing ticket ID %s: %s", tk.ID, out)
	}
	if !strings.Contains(out, workBranch) {
		t.Errorf("output missing work branch %s: %s", workBranch, out)
	}
	if !strings.Contains(out, "PR Worktree Created") {
		t.Errorf("output missing 'PR Worktree Created': %s", out)
	}
	if !strings.Contains(out, "merge --squash") {
		t.Errorf("output missing merge command: %s", out)
	}
	if !strings.Contains(out, "test prep feature") {
		t.Errorf("output missing suggested commit message: %s", out)
	}

	// Should have created a pr- worktree
	prPath := filepath.Join(wd, ".worktrees", "pr-"+tk.ID)
	if _, err := os.Stat(prPath); os.IsNotExist(err) {
		t.Errorf("expected PR worktree at %s to be created", prPath)
	}
}

func TestPrep_NoWorkBranch(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tk := env.createTicket(t, "no branch yet", ticket.StatusInProgress)
	tk.Assignee = "test-session"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err = env.runCmd(t, "--human", "prep", tk.ID)
	if err == nil {
		t.Fatal("expected error when work branch doesn't exist")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want substring 'not found'", err.Error())
	}
}

func TestPrep_NoCommits(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tk := env.createTicket(t, "empty branch", ticket.StatusInProgress)
	tk.Assignee = "test-session"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Create branch with no commits beyond master
	workBranch := "st/" + tk.ID
	runGitCmd(t, wd, "branch", workBranch)

	out, err := env.runCmd(t, "--human", "prep", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "nothing to merge") {
		t.Errorf("output missing 'nothing to merge': %s", out)
	}
}

// --- Batch mode tests ---

// createTicketWithBranch creates a ticket with the given status, then creates
// an st/<id> branch with a unique file committed on it.
func createTicketWithBranch(t *testing.T, env *testEnv, wd, title string, status ticket.Status, filename, content string) *ticket.Ticket {
	t.Helper()

	tk := env.createTicket(t, title, status)

	workBranch := "st/" + tk.ID
	runGitCmd(t, wd, "checkout", "-b", workBranch)

	testFile := filepath.Join(wd, filename)
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	runGitCmd(t, wd, "add", filename)
	runGitCmd(t, wd, "commit", "-m", "add "+filename)

	runGitCmd(t, wd, "checkout", "master")

	return tk
}

func TestPrepBatch_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tk1 := createTicketWithBranch(t, env, wd, "batch feature one", ticket.StatusReview, "one.go", "package main\n")
	tk2 := createTicketWithBranch(t, env, wd, "batch feature two", ticket.StatusDone, "two.go", "package main\n")

	out, err := env.runCmd(t, "--human", "prep")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, tk1.ID) {
		t.Errorf("output missing ticket ID %s: %s", tk1.ID, out)
	}
	if !strings.Contains(out, tk2.ID) {
		t.Errorf("output missing ticket ID %s: %s", tk2.ID, out)
	}
	if !strings.Contains(out, "2 ticket(s)") {
		t.Errorf("output missing '2 ticket(s)': %s", out)
	}
	if !strings.Contains(out, "PR Worktree Created") {
		t.Errorf("output missing 'PR Worktree Created': %s", out)
	}
	if !strings.Contains(out, "merge --squash") {
		t.Errorf("output missing merge commands: %s", out)
	}

	// Should have created a single pr- worktree
	entries, _ := os.ReadDir(filepath.Join(wd, ".worktrees"))
	var prWorktrees []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "pr-") {
			prWorktrees = append(prWorktrees, e.Name())
		}
	}
	if len(prWorktrees) != 1 {
		t.Errorf("expected exactly 1 PR worktree, found %d: %v", len(prWorktrees), prWorktrees)
	}
}

func TestPrepBatch_NoMergeable(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	// Create a ticket that's IN-PROGRESS (not mergeable)
	_ = env.createTicket(t, "still working", ticket.StatusInProgress)

	out, err := env.runCmd(t, "--human", "prep")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "No mergeable tickets") {
		t.Errorf("output missing 'No mergeable tickets': %s", out)
	}
}

func TestPrepBatch_SkipsEmptyBranches(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	// Create one ticket with commits
	tk1 := createTicketWithBranch(t, env, wd, "has commits", ticket.StatusReview, "real.go", "package main\n")

	// Create another ticket with a branch but no commits
	tkEmpty := env.createTicket(t, "empty branch", ticket.StatusReview)
	runGitCmd(t, wd, "branch", "st/"+tkEmpty.ID)

	out, err := env.runCmd(t, "--human", "prep")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, tk1.ID) {
		t.Errorf("output missing ticket with commits %s: %s", tk1.ID, out)
	}
	if !strings.Contains(out, "Skipped") {
		t.Errorf("output missing 'Skipped': %s", out)
	}
	if !strings.Contains(out, tkEmpty.ID) {
		t.Errorf("output missing skipped ticket ID %s: %s", tkEmpty.ID, out)
	}
}

func TestPrepBatch_DependencyOrder(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tkA := createTicketWithBranch(t, env, wd, "dep A (base)", ticket.StatusReview, "a.go", "package main\n")

	tkB := createTicketWithBranch(t, env, wd, "dep B (depends on A)", ticket.StatusReview, "b.go", "package main\n")
	tkB.DependsOn = []string{tkA.ID}
	if err := env.Store.Save(tkB); err != nil {
		t.Fatalf("save ticket B: %v", err)
	}

	out, err := env.runCmd(t, "--human", "prep")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// A should appear before B in the output
	idxA := strings.Index(out, tkA.ID)
	idxB := strings.Index(out, tkB.ID)
	if idxA < 0 || idxB < 0 {
		t.Fatalf("both ticket IDs should appear in output: %s", out)
	}
	if idxA >= idxB {
		t.Errorf("ticket A (%s) should appear before B (%s) in output, but A at %d, B at %d", tkA.ID, tkB.ID, idxA, idxB)
	}
}

func TestPrepBatch_CrossTicketConflict(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	// Two tickets that both modify the same file (different content)
	tk1 := createTicketWithBranch(t, env, wd, "conflict one", ticket.StatusReview, "shared.go", "package one\n")
	tk2 := createTicketWithBranch(t, env, wd, "conflict two", ticket.StatusReview, "shared.go", "package two\n")

	out, err := env.runCmd(t, "--human", "prep")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Conflict Risk") {
		t.Errorf("output missing 'Conflict Risk': %s", out)
	}
	if !strings.Contains(out, "shared.go") {
		t.Errorf("output missing conflicting file name: %s", out)
	}
	if !strings.Contains(out, tk1.ID) || !strings.Contains(out, tk2.ID) {
		t.Errorf("output should list both ticket IDs in conflict section: %s", out)
	}
}

func TestPrepBatch_SingleTicketStillWorks(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tk := createTicketWithBranch(t, env, wd, "single ticket prep", ticket.StatusInProgress, "single.go", "package main\n")

	out, err := env.runCmd(t, "--human", "prep", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "PR Worktree Created") {
		t.Errorf("output missing 'PR Worktree Created': %s", out)
	}
	if !strings.Contains(out, "merge --squash") {
		t.Errorf("output missing merge command: %s", out)
	}
}

func TestSortByDependencyOrder(t *testing.T) {
	now := time.Now().UTC()

	t.Run("linear chain", func(t *testing.T) {
		a := &ticket.Ticket{ID: "a", Priority: ticket.PriorityP3, Updated: now}
		b := &ticket.Ticket{ID: "b", Priority: ticket.PriorityP3, Updated: now, DependsOn: []string{"a"}}
		c := &ticket.Ticket{ID: "c", Priority: ticket.PriorityP3, Updated: now, DependsOn: []string{"b"}}

		result := sortByDependencyOrder([]*ticket.Ticket{c, b, a})

		if len(result) != 3 {
			t.Fatalf("expected 3 tickets, got %d", len(result))
		}
		if result[0].ID != "a" || result[1].ID != "b" || result[2].ID != "c" {
			t.Errorf("expected [a, b, c], got [%s, %s, %s]", result[0].ID, result[1].ID, result[2].ID)
		}
	})

	t.Run("priority tiebreak", func(t *testing.T) {
		a := &ticket.Ticket{ID: "a", Priority: ticket.PriorityP3, Updated: now}
		b := &ticket.Ticket{ID: "b", Priority: ticket.PriorityP0, Updated: now}

		result := sortByDependencyOrder([]*ticket.Ticket{a, b})

		if result[0].ID != "b" {
			t.Errorf("P0 ticket should come first, got %s", result[0].ID)
		}
	})

	t.Run("updated time tiebreak", func(t *testing.T) {
		older := now.Add(-time.Hour)
		a := &ticket.Ticket{ID: "a", Priority: ticket.PriorityP3, Updated: older}
		b := &ticket.Ticket{ID: "b", Priority: ticket.PriorityP3, Updated: now}

		result := sortByDependencyOrder([]*ticket.Ticket{a, b})

		if result[0].ID != "b" {
			t.Errorf("newer ticket should come first, got %s", result[0].ID)
		}
	})

	t.Run("single ticket", func(t *testing.T) {
		a := &ticket.Ticket{ID: "a", Priority: ticket.PriorityP3, Updated: now}

		result := sortByDependencyOrder([]*ticket.Ticket{a})

		if len(result) != 1 || result[0].ID != "a" {
			t.Errorf("single ticket should pass through unchanged")
		}
	})
}
