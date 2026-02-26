package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHold_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "hold me", ticket.StatusInProgress)

	out, err := env.runCmd(t, "hold", tk.ID, "waiting on feedback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Held "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Held "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusBlocked {
		t.Errorf("status = %s, want BLOCKED", updated.Status)
	}
	if updated.PriorStatus == nil {
		t.Fatal("prior_status is nil, want IN-PROGRESS")
	}
	if *updated.PriorStatus != ticket.StatusInProgress {
		t.Errorf("prior_status = %s, want IN-PROGRESS", *updated.PriorStatus)
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.StatusBlocked {
			found = true
			break
		}
	}
	if !found {
		t.Error("no status.blocked event logged")
	}
}

func TestHold_AlreadyBlocked(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "already blocked", ticket.StatusBlocked)

	_, err := env.runCmd(t, "hold", tk.ID, "double block")
	if err == nil {
		t.Fatal("expected error for already BLOCKED ticket")
	}
	if !strings.Contains(err.Error(), "already BLOCKED") {
		t.Errorf("error = %q, want substring %q", err.Error(), "already BLOCKED")
	}
}

func TestHold_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "hold", "st_zzzzzz", "missing")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}
