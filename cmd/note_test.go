package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestNote_OnCurrentTicket(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "note target", ticket.StatusInProgress)
	tk.Assignee = "test-session-note"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", "this is a note")
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
		if e.Event == event.TicketNote {
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

	tk := env.createTicket(t, "flagged note target", ticket.StatusOpen)

	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", "--ticket", tk.ID, "flagged note")
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
	_ = env

	// No tickets assigned to this session
	_, err := env.runCmd(t, "--run-id", "test-session-note", "note", "orphan note")
	if err == nil {
		t.Fatal("expected error when no active ticket and no --ticket flag")
	}
}

func TestNote_WithPositionalTicketID(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "positional note target", ticket.StatusOpen)

	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", tk.ID, "positional note message")
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
	if !strings.Contains(updated.Body, "positional note message") {
		t.Errorf("body = %q, want note content in body", updated.Body)
	}
}

func TestNote_PositionalTicketID_FlagTakesPrecedence(t *testing.T) {
	env := newTestEnv(t)

	tkFlag := env.createTicket(t, "flag target", ticket.StatusOpen)
	tkPos := env.createTicket(t, "positional target", ticket.StatusOpen)

	// --ticket flag should take precedence over positional arg
	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", "--ticket", tkFlag.ID, tkPos.ID, "flag wins message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tkFlag.ID) {
		t.Errorf("output = %q, want note added to flag ticket %s", out, tkFlag.ID)
	}

	updated, err := env.Store.Get(tkFlag.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "flag wins message") {
		t.Errorf("flag ticket body = %q, want note content", updated.Body)
	}
}

func TestNote_SingleArgTicketIDErrors(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "target", ticket.StatusInProgress)
	tk.Assignee = "test-session-note"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Single arg that looks like a ticket ID should error helpfully
	_, err := env.runCmd(t, "--run-id", "test-session-note", "note", tk.ID)
	if err == nil {
		t.Fatal("expected error when single arg looks like a ticket ID")
	}
	if !strings.Contains(err.Error(), "looks like a ticket ID") {
		t.Errorf("error = %q, want helpful message about ticket ID", err.Error())
	}
}

func TestNote_NoRunID(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	// No --run-id and no --human
	_, err := env.runCmd(t, "note", "no session note")
	if err == nil {
		t.Fatal("expected error when no --run-id is provided")
	}
}
