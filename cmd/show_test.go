package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovbrain/internal/ticket"
)

func TestShow_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "show me details", ticket.StatusOpen)

	out, err := env.runCmd(t, "show", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// show renders the full markdown (frontmatter + body)
	if !strings.Contains(out, tk.ID) {
		t.Errorf("output = %q, want ticket ID %s", out, tk.ID)
	}
	if !strings.Contains(out, "show me details") {
		t.Errorf("output = %q, want title in output", out)
	}
	if !strings.Contains(out, "OPEN") {
		t.Errorf("output = %q, want status OPEN in output", out)
	}
}

func TestShow_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "show", "sb_zzzzzz")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}
