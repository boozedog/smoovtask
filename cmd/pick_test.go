package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestPick_ByExplicitID(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-pick")

	tk := env.createTicket(t, "pick me", ticket.StatusOpen)

	out, err := env.runCmd(t, "pick", tk.ID)
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
	t.Setenv("CLAUDE_SESSION_ID", "test-session-flag")

	tk := env.createTicket(t, "flag pick me", ticket.StatusOpen)

	out, err := env.runCmd(t, "pick", "--ticket", tk.ID)
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
	t.Setenv("CLAUDE_SESSION_ID", "test-session-precedence")

	tkFlag := env.createTicket(t, "flag target", ticket.StatusOpen)
	tkPos := env.createTicket(t, "positional target", ticket.StatusOpen)

	// --ticket flag should take precedence over positional arg
	out, err := env.runCmd(t, "pick", "--ticket", tkFlag.ID, tkPos.ID)
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

func TestPick_AutoSelect(t *testing.T) {
	env := newTestEnvResolved(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-auto")

	tk := env.createTicket(t, "auto pick me", ticket.StatusOpen)

	out, err := env.runCmd(t, "pick")
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

func TestPick_FromRework(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-rework")

	tk := env.createTicket(t, "rework ticket", ticket.StatusRework)

	out, err := env.runCmd(t, "pick", tk.ID)
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
	t.Setenv("CLAUDE_SESSION_ID", "test-session")

	tk := env.createTicket(t, "already started", ticket.StatusInProgress)

	_, err := env.runCmd(t, "pick", tk.ID)
	if err == nil {
		t.Fatal("expected error for already IN-PROGRESS ticket")
	}
}

func TestPick_NoOpenTickets(t *testing.T) {
	env := newTestEnvResolved(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session")

	// No tickets at all
	_, err := env.runCmd(t, "pick")
	if err == nil {
		t.Fatal("expected error for no open tickets")
	}
}

func TestPick_NoSessionFallsBackToActor(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("CLAUDECODE", "")

	tk := env.createTicket(t, "no session pick", ticket.StatusOpen)

	_, err := env.runCmd(t, "pick", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	// With no session ID and no CLAUDECODE, actor is "human"
	if updated.Assignee != "human" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "human")
	}
}
