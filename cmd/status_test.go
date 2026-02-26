package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestStatus_ValidTransition(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-status")

	// Create an IN-PROGRESS ticket assigned to our session
	tk := env.createTicket(t, "status test", ticket.StatusInProgress)
	tk.Assignee = "test-session-status"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Add a note (required before review)
	env.addNoteEvent(t, tk.ID)

	out, err := env.runCmd(t, "status", "review")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "IN-PROGRESS → REVIEW") {
		t.Errorf("output = %q, want substring %q", out, "IN-PROGRESS → REVIEW")
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusReview {
		t.Errorf("status = %s, want REVIEW", updated.Status)
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.StatusReview {
			found = true
			break
		}
	}
	if !found {
		t.Error("no status.review event logged")
	}
}

func TestStatus_InvalidTransition(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-status")

	// OPEN ticket cannot go directly to REVIEW
	tk := env.createTicket(t, "bad transition", ticket.StatusOpen)
	tk.Assignee = "test-session-status"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err := env.runCmd(t, "status", "--ticket", tk.ID, "review")
	if err == nil {
		t.Fatal("expected error for invalid transition OPEN → REVIEW")
	}
}

func TestStatus_RequiresNoteBeforeReview(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-status")

	tk := env.createTicket(t, "note required test", ticket.StatusInProgress)
	tk.Assignee = "test-session-status"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Try to move to review without a note — should fail
	_, err := env.runCmd(t, "status", "--ticket", tk.ID, "review")
	if err == nil {
		t.Fatal("expected error when no note exists before review")
	}
	if !strings.Contains(err.Error(), "note is required") {
		t.Errorf("error = %q, want substring %q", err.Error(), "note is required")
	}
}

func TestStatus_WithTicketFlag(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-status")

	tk := env.createTicket(t, "ticket flag test", ticket.StatusInProgress)
	tk.Assignee = "test-session-status"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	env.addNoteEvent(t, tk.ID)

	out, err := env.runCmd(t, "status", "--ticket", tk.ID, "review")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "IN-PROGRESS → REVIEW") {
		t.Errorf("output = %q, want substring %q", out, "IN-PROGRESS → REVIEW")
	}
}

func TestStatus_AutoUnblock(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-status")

	// Create ticket B (will be a dependency)
	tkB := env.createTicket(t, "dependency B", ticket.StatusInProgress)
	tkB.Assignee = "test-session-status"
	if err := env.Store.Save(tkB); err != nil {
		t.Fatalf("save ticket B: %v", err)
	}

	// Create ticket A that depends on B, manually set to BLOCKED
	tkA := env.createTicket(t, "dependent A", ticket.StatusBlocked)
	openStatus := ticket.StatusOpen
	tkA.PriorStatus = &openStatus
	tkA.DependsOn = []string{tkB.ID}
	if err := env.Store.Save(tkA); err != nil {
		t.Fatalf("save ticket A: %v", err)
	}

	// Move B: IN-PROGRESS → REVIEW → DONE (need two transitions)
	// Add a note (required before review)
	env.addNoteEvent(t, tkB.ID)
	// First go to REVIEW
	_, err := env.runCmd(t, "status", "--ticket", tkB.ID, "review")
	if err != nil {
		t.Fatalf("review transition: %v", err)
	}

	// Then go to DONE
	out, err := env.runCmd(t, "status", "--ticket", tkB.ID, "done")
	if err != nil {
		t.Fatalf("done transition: %v", err)
	}

	if !strings.Contains(out, "Auto-unblocked") {
		t.Errorf("output = %q, want substring %q", out, "Auto-unblocked")
	}

	// Verify A is unblocked
	updatedA, err := env.Store.Get(tkA.ID)
	if err != nil {
		t.Fatalf("get ticket A: %v", err)
	}
	if updatedA.Status != ticket.StatusOpen {
		t.Errorf("ticket A status = %s, want OPEN (snapped back)", updatedA.Status)
	}
	if updatedA.PriorStatus != nil {
		t.Errorf("ticket A prior_status = %v, want nil", updatedA.PriorStatus)
	}
}

func TestStatus_NoActiveTicket(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-status")
	_ = env

	// No tickets at all — resolver should fail
	_, err := env.runCmd(t, "status", "review")
	if err == nil {
		t.Fatal("expected error when no active ticket")
	}
}
