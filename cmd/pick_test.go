package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestPick_ByExplicitID(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "pick me", ticket.StatusOpen)

	out, err := env.runCmd(t, "--run-id", "test-session-pick", "pick", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Picked up "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Picked up "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("status = %s, want IN-PROGRESS", updated.Status)
	}
	if updated.Assignee != "test-session-pick" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "test-session-pick")
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.StatusInProgress {
			found = true
			break
		}
	}
	if !found {
		t.Error("no status.in-progress event logged")
	}
}

func TestPick_ByTicketFlag(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "flag pick me", ticket.StatusOpen)

	out, err := env.runCmd(t, "--run-id", "test-session-flag", "pick", "--ticket", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Picked up "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Picked up "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("status = %s, want IN-PROGRESS", updated.Status)
	}
	if updated.Assignee != "test-session-flag" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "test-session-flag")
	}
}

func TestPick_TicketFlagPrecedence(t *testing.T) {
	env := newTestEnv(t)

	tkFlag := env.createTicket(t, "flag target", ticket.StatusOpen)
	tkPos := env.createTicket(t, "positional target", ticket.StatusOpen)

	// --ticket flag should take precedence over positional arg
	out, err := env.runCmd(t, "--run-id", "test-session-precedence", "pick", "--ticket", tkFlag.ID, tkPos.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Picked up "+tkFlag.ID) {
		t.Errorf("output = %q, want picked up flag ticket %s", out, tkFlag.ID)
	}

	updated, err := env.Store.Get(tkFlag.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("flag ticket status = %s, want IN-PROGRESS", updated.Status)
	}

	// Positional ticket should remain unchanged
	posUpdated, err := env.Store.Get(tkPos.ID)
	if err != nil {
		t.Fatalf("get positional ticket: %v", err)
	}
	if posUpdated.Status != ticket.StatusOpen {
		t.Errorf("positional ticket status = %s, want OPEN (unchanged)", posUpdated.Status)
	}
}

func TestPick_RequiresTicketID(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.runCmd(t, "--run-id", "test-session-auto", "pick")
	if err == nil {
		t.Fatal("expected error when no ticket ID provided")
	}
	if !strings.Contains(err.Error(), "ticket ID required") {
		t.Errorf("error = %q, want substring %q", err.Error(), "ticket ID required")
	}
}

func TestPick_FromRework(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "rework ticket", ticket.StatusRework)

	out, err := env.runCmd(t, "--run-id", "test-session-rework", "pick", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Picked up "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Picked up "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("status = %s, want IN-PROGRESS", updated.Status)
	}
}

func TestPick_AlreadyInProgress(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "already started", ticket.StatusInProgress)

	_, err := env.runCmd(t, "--run-id", "test-session", "pick", tk.ID)
	if err == nil {
		t.Fatal("expected error for already IN-PROGRESS ticket")
	}
}

func TestPick_RejectsSecondActiveTicket(t *testing.T) {
	env := newTestEnv(t)

	active := env.createTicket(t, "already active", ticket.StatusInProgress)
	active.Assignee = "test-session"
	if err := env.Store.Save(active); err != nil {
		t.Fatalf("save active ticket: %v", err)
	}

	newTicket := env.createTicket(t, "new pick", ticket.StatusOpen)

	_, err := env.runCmd(t, "--run-id", "test-session", "pick", newTicket.ID)
	if err == nil {
		t.Fatal("expected error when run already has active ticket")
	}
	if !strings.Contains(err.Error(), "already has active ticket") {
		t.Errorf("error = %q, want substring %q", err.Error(), "already has active ticket")
	}
}

func TestPick_HumanAssignsHumanActor(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "no session pick", ticket.StatusOpen)

	_, err := env.runCmd(t, "--human", "pick", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Assignee != "human" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "human")
	}
}

func TestPick_AgentRequiresRunID(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "missing run-id", ticket.StatusOpen)

	_, err := env.runCmdRaw(t, "pick", tk.ID)
	if err == nil {
		t.Fatal("expected error when agent command omits --run-id")
	}
	if !strings.Contains(err.Error(), "run ID required") {
		t.Errorf("error = %q, want run ID required", err.Error())
	}
}
