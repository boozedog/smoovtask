package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestAssign_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "assign me", ticket.StatusOpen)

	out, err := env.runCmd(t, "assign", tk.ID, "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Assigned "+tk.ID+" to agent-123") {
		t.Errorf("output = %q, want substring %q", out, "Assigned "+tk.ID+" to agent-123")
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Assignee != "agent-123" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "agent-123")
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.TicketAssigned {
			found = true
			break
		}
	}
	if !found {
		t.Error("no ticket.assigned event logged")
	}
}

func TestAssign_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "assign", "st_zzzzzz", "agent-123")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}

func TestAssign_OverwriteExisting(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "reassign me", ticket.StatusInProgress)
	tk.Assignee = "old-agent"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err := env.runCmd(t, "assign", tk.ID, "new-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Assignee != "new-agent" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "new-agent")
	}
}
