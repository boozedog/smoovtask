package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestSpawn_DryRun(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "Test spawn dry run", ticket.StatusOpen)

	out, err := env.runCmd(t, "spawn", "--dry-run", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Dry Run: Prompt") {
		t.Errorf("output = %q, want 'Dry Run: Prompt'", out)
	}
	if !strings.Contains(out, tk.ID) {
		t.Errorf("output = %q, want ticket ID %s", out, tk.ID)
	}
	if !strings.Contains(out, "Test spawn dry run") {
		t.Errorf("output = %q, want ticket title", out)
	}
	if !strings.Contains(out, "st note") {
		t.Errorf("output = %q, want st note instructions", out)
	}
	if !strings.Contains(out, "st status done") {
		t.Errorf("output = %q, want st status done instructions", out)
	}
}

func TestSpawn_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "spawn", "--dry-run", "st_zzzzzz")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}

func TestSpawn_UnknownBackend(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "Test bad backend", ticket.StatusOpen)

	_, err := env.runCmd(t, "spawn", "--backend", "nonexistent", tk.ID)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
