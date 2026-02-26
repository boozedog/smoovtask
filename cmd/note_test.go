package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/ticket"
)

func TestNote_OnCurrentTicket(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-note")

	tk := env.createTicket(t, "note target", ticket.StatusInProgress)
	tk.Assignee = "test-session-note"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "note", "this is a note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Note added to "+tk.ID)
	}

	// Verify note appears in ticket body
	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "this is a note") {
		t.Errorf("body = %q, want note content in body", updated.Body)
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == "ticket.note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no ticket.note event logged")
	}
}

func TestNote_WithTicketFlag(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-note")

	tk := env.createTicket(t, "flagged note target", ticket.StatusOpen)

	out, err := env.runCmd(t, "note", "--ticket", tk.ID, "flagged note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Note added to "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "flagged note") {
		t.Errorf("body = %q, want note content in body", updated.Body)
	}
}

func TestNote_NoActiveTicketAndNoFlag(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-note")
	_ = env

	// No tickets assigned to this session
	_, err := env.runCmd(t, "note", "orphan note")
	if err == nil {
		t.Fatal("expected error when no active ticket and no --ticket flag")
	}
}

func TestNote_NoSession(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "")
	_ = env

	// No session and no --ticket flag
	_, err := env.runCmd(t, "note", "no session note")
	if err == nil {
		t.Fatal("expected error when no session and no --ticket flag")
	}
}
