package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestCancel_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "cancel me", ticket.StatusOpen)

	out, err := env.runCmd(t, "cancel", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Cancelled "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Cancelled "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusCancelled {
		t.Errorf("status = %s, want CANCELLED", updated.Status)
	}
	if updated.PriorStatus != nil {
		t.Errorf("prior_status = %v, want nil", updated.PriorStatus)
	}
	if updated.Assignee != "" {
		t.Errorf("assignee = %q, want empty", updated.Assignee)
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.StatusCancelled {
			found = true
			break
		}
	}
	if !found {
		t.Error("no status.cancelled event logged")
	}
}

func TestCancel_WithReason(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "cancel with reason", ticket.StatusInProgress)

	out, err := env.runCmd(t, "cancel", tk.ID, "no longer needed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Cancelled "+tk.ID+": no longer needed") {
		t.Errorf("output = %q, want substring %q", out, "Cancelled "+tk.ID+": no longer needed")
	}

	// Verify reason in event data
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	for _, e := range events {
		if e.Event == event.StatusCancelled {
			if msg, ok := e.Data["message"].(string); !ok || msg != "no longer needed" {
				t.Errorf("event message = %v, want %q", e.Data["message"], "no longer needed")
			}
			break
		}
	}
}

func TestCancel_FromAnyStatus(t *testing.T) {
	env := newTestEnv(t)

	// Cancel from REVIEW (not a typical transition, but cancel forces it)
	tk := env.createTicket(t, "force cancel", ticket.StatusReview)

	_, err := env.runCmd(t, "cancel", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusCancelled {
		t.Errorf("status = %s, want CANCELLED", updated.Status)
	}
}

func TestCancel_ClearsAssignee(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "assigned ticket", ticket.StatusInProgress)
	tk.Assignee = "some-agent-session"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err := env.runCmd(t, "cancel", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Assignee != "" {
		t.Errorf("assignee = %q, want empty", updated.Assignee)
	}
}

func TestCancel_AlreadyCancelled(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "already cancelled", ticket.StatusCancelled)

	_, err := env.runCmd(t, "cancel", tk.ID)
	if err == nil {
		t.Fatal("expected error for already-cancelled ticket")
	}
	if !strings.Contains(err.Error(), "already CANCELLED") {
		t.Errorf("error = %q, want substring %q", err.Error(), "already CANCELLED")
	}
}

func TestCancel_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "cancel", "st_zzzzzz")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}

func TestCancel_AutoUnblocksDependents(t *testing.T) {
	env := newTestEnv(t)

	// Create ticket A (dependency)
	tkA := env.createTicket(t, "dep ticket A", ticket.StatusOpen)

	// Create ticket B that depends on A, manually BLOCKED
	tkB := env.createTicket(t, "dependent B", ticket.StatusBlocked)
	openStatus := ticket.StatusOpen
	tkB.PriorStatus = &openStatus
	tkB.DependsOn = []string{tkA.ID}
	if err := env.Store.Save(tkB); err != nil {
		t.Fatalf("save ticket B: %v", err)
	}

	// Cancel A
	out, err := env.runCmd(t, "cancel", tkA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Auto-unblocked") {
		t.Errorf("output = %q, want substring %q", out, "Auto-unblocked")
	}

	// Verify B is unblocked
	updatedB, err := env.Store.Get(tkB.ID)
	if err != nil {
		t.Fatalf("get ticket B: %v", err)
	}
	if updatedB.Status != ticket.StatusOpen {
		t.Errorf("ticket B status = %s, want OPEN", updatedB.Status)
	}
}
