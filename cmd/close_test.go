package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/ticket"
)

func TestClose_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "close me", ticket.StatusInProgress)

	out, err := env.runCmd(t, "close", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Closed "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Closed "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusDone {
		t.Errorf("status = %s, want DONE", updated.Status)
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
		if e.Event == event.StatusDone {
			found = true
			break
		}
	}
	if !found {
		t.Error("no status.done event logged")
	}
}

func TestClose_FromAnyStatus(t *testing.T) {
	env := newTestEnv(t)

	// Close from OPEN (normally not a valid transition, but close forces it)
	tk := env.createTicket(t, "force close", ticket.StatusOpen)

	_, err := env.runCmd(t, "close", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusDone {
		t.Errorf("status = %s, want DONE", updated.Status)
	}
}

func TestClose_AutoUnblocksDependents(t *testing.T) {
	env := newTestEnv(t)

	// Create ticket A (dependency)
	tkA := env.createTicket(t, "dep ticket A", ticket.StatusInProgress)

	// Create ticket B that depends on A, manually BLOCKED
	tkB := env.createTicket(t, "dependent B", ticket.StatusBlocked)
	openStatus := ticket.StatusOpen
	tkB.PriorStatus = &openStatus
	tkB.DependsOn = []string{tkA.ID}
	if err := env.Store.Save(tkB); err != nil {
		t.Fatalf("save ticket B: %v", err)
	}

	// Close A
	out, err := env.runCmd(t, "close", tkA.ID)
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
	if updatedB.PriorStatus != nil {
		t.Errorf("ticket B prior_status = %v, want nil", updatedB.PriorStatus)
	}
}

func TestClose_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "close", "sb_zzzzzz")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}
