package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHandoff_FromInProgress(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "handoff me", ticket.StatusInProgress)
	tk.Assignee = "test-session-1"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "--run-id", "test-session-1", "handoff", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Handed off "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Handed off "+tk.ID)
	}
	if !strings.Contains(out, "IN-PROGRESS → OPEN") {
		t.Errorf("output = %q, want substring %q", out, "IN-PROGRESS → OPEN")
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusOpen {
		t.Errorf("status = %s, want OPEN", updated.Status)
	}
	if updated.Assignee != "" {
		t.Errorf("assignee = %q, want empty", updated.Assignee)
	}
}

func TestHandoff_FromRework(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "rework handoff", ticket.StatusRework)
	tk.Assignee = "test-session-2"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "--run-id", "test-session-2", "handoff", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Handed off "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Handed off "+tk.ID)
	}
	if !strings.Contains(out, "REWORK → OPEN") {
		t.Errorf("output = %q, want substring %q", out, "REWORK → OPEN")
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusOpen {
		t.Errorf("status = %s, want OPEN", updated.Status)
	}
	if updated.Assignee != "" {
		t.Errorf("assignee = %q, want empty", updated.Assignee)
	}
}

func TestHandoff_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "open ticket", ticket.StatusOpen)
	tk.Assignee = "test-session-3"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err := env.runCmd(t, "--run-id", "test-session-3", "handoff", tk.ID)
	if err == nil {
		t.Fatal("expected error for OPEN ticket handoff")
	}
}

func TestHandoff_NoAssignee(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "unassigned ticket", ticket.StatusInProgress)
	// No assignee set

	_, err := env.runCmd(t, "--run-id", "test-session-4", "handoff", tk.ID)
	if err == nil {
		t.Fatal("expected error for ticket with no assignee")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("error = %q, want substring %q", err.Error(), "no assignee")
	}
}

func TestHandoff_EventLogged(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "event handoff", ticket.StatusInProgress)
	tk.Assignee = "test-session-5"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err := env.runCmd(t, "--run-id", "test-session-5", "handoff", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.TicketHandoff {
			found = true
			break
		}
	}
	if !found {
		t.Error("no ticket.handoff event logged")
	}
}

func TestHandoff_ByTicketFlag(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "flag handoff", ticket.StatusInProgress)
	tk.Assignee = "test-session-6"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "--run-id", "test-session-6", "handoff", "--ticket", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Handed off "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Handed off "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusOpen {
		t.Errorf("status = %s, want OPEN", updated.Status)
	}
	if updated.Assignee != "" {
		t.Errorf("assignee = %q, want empty", updated.Assignee)
	}
}
