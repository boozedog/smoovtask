package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovbrain/internal/ticket"
)

func TestList_All(t *testing.T) {
	env := newTestEnvResolved(t)

	tk1 := env.createTicket(t, "list ticket one", ticket.StatusOpen)
	tk2 := env.createTicket(t, "list ticket two", ticket.StatusInProgress)

	out, err := env.runCmd(t, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, tk1.ID) {
		t.Errorf("output = %q, want ticket 1 ID %s", out, tk1.ID)
	}
	if !strings.Contains(out, tk2.ID) {
		t.Errorf("output = %q, want ticket 2 ID %s", out, tk2.ID)
	}
}

func TestList_FilterByStatus(t *testing.T) {
	env := newTestEnvResolved(t)

	tkOpen := env.createTicket(t, "open ticket", ticket.StatusOpen)
	tkDone := env.createTicket(t, "done ticket", ticket.StatusDone)

	out, err := env.runCmd(t, "list", "--status", "open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, tkOpen.ID) {
		t.Errorf("output = %q, want open ticket ID %s", out, tkOpen.ID)
	}
	if strings.Contains(out, tkDone.ID) {
		t.Errorf("output = %q, should not contain done ticket ID %s", out, tkDone.ID)
	}
}

func TestList_FilterByProject(t *testing.T) {
	env := newTestEnvResolved(t)

	tk := env.createTicket(t, "project ticket", ticket.StatusOpen)

	// --project testproject should find the ticket
	out, err := env.runCmd(t, "list", "--project", "testproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, tk.ID) {
		t.Errorf("output = %q, want ticket ID %s", out, tk.ID)
	}

	// --project nonexistent should not find it
	out, err = env.runCmd(t, "list", "--project", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, tk.ID) {
		t.Errorf("output = %q, should not contain ticket for wrong project", out)
	}
}

func TestList_Empty(t *testing.T) {
	env := newTestEnvResolved(t)
	_ = env

	out, err := env.runCmd(t, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No tickets found") {
		t.Errorf("output = %q, want %q", out, "No tickets found")
	}
}
