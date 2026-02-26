package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/ticket"
)

func TestUnhold_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	// Create a BLOCKED ticket with prior status IN-PROGRESS
	tk := env.createTicket(t, "unhold me", ticket.StatusBlocked)
	prior := ticket.StatusInProgress
	tk.PriorStatus = &prior
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "unhold", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Released hold on "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Released hold on "+tk.ID)
	}
	if !strings.Contains(out, "IN-PROGRESS") {
		t.Errorf("output = %q, want substring %q", out, "IN-PROGRESS")
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("status = %s, want IN-PROGRESS", updated.Status)
	}
	if updated.PriorStatus != nil {
		t.Errorf("prior_status = %v, want nil", updated.PriorStatus)
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
		t.Error("no status.in-progress event logged for unhold")
	}
}

func TestUnhold_NotBlocked(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "not blocked", ticket.StatusOpen)

	_, err := env.runCmd(t, "unhold", tk.ID)
	if err == nil {
		t.Fatal("expected error for ticket not BLOCKED")
	}
	if !strings.Contains(err.Error(), "not BLOCKED") {
		t.Errorf("error = %q, want substring %q", err.Error(), "not BLOCKED")
	}
}

func TestUnhold_NoPriorStatus(t *testing.T) {
	env := newTestEnv(t)

	// BLOCKED ticket with nil PriorStatus — edge case
	tk := env.createTicket(t, "no prior", ticket.StatusBlocked)
	// PriorStatus is nil by default from createTicket

	_, err := env.runCmd(t, "unhold", tk.ID)
	if err == nil {
		t.Fatal("expected error for nil PriorStatus")
	}
	if !strings.Contains(err.Error(), "no prior status") {
		t.Errorf("error = %q, want substring %q", err.Error(), "no prior status")
	}
}

func TestUnhold_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "unhold", "sb_zzzzzz")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}
