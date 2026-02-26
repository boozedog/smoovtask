package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/ticket"
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

func TestList_HidesDoneByDefault(t *testing.T) {
	env := newTestEnvResolved(t)

	tkOpen := env.createTicket(t, "open ticket", ticket.StatusOpen)
	tkDone := env.createTicket(t, "done ticket", ticket.StatusDone)

	out, err := env.runCmd(t, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, tkOpen.ID) {
		t.Errorf("output should contain open ticket %s", tkOpen.ID)
	}
	if strings.Contains(out, tkDone.ID) {
		t.Errorf("output should not contain done ticket %s without --all", tkDone.ID)
	}
}

func TestList_AllShowsDone(t *testing.T) {
	env := newTestEnvResolved(t)

	tkOpen := env.createTicket(t, "open ticket", ticket.StatusOpen)
	tkDone := env.createTicket(t, "done ticket", ticket.StatusDone)

	out, err := env.runCmd(t, "list", "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, tkOpen.ID) {
		t.Errorf("output should contain open ticket %s", tkOpen.ID)
	}
	if !strings.Contains(out, tkDone.ID) {
		t.Errorf("output should contain done ticket %s with --all", tkDone.ID)
	}
}

func TestList_SortOrder(t *testing.T) {
	env := newTestEnvResolved(t)

	tkOpen := env.createTicket(t, "open ticket", ticket.StatusOpen)
	tkReview := env.createTicket(t, "review ticket", ticket.StatusReview)

	out, err := env.runCmd(t, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reviewIdx := strings.Index(out, tkReview.ID)
	openIdx := strings.Index(out, tkOpen.ID)

	if reviewIdx < 0 || openIdx < 0 {
		t.Fatalf("output missing ticket IDs: %q", out)
	}
	if reviewIdx > openIdx {
		t.Errorf("REVIEW ticket should appear before OPEN ticket in output")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"over", "hello world", 5, "hell…"},
		{"empty", "", 5, ""},
		{"multibyte_under", "éàü", 5, "éàü"},
		{"multibyte_over", "éàüöñç", 5, "éàüö…"},
		{"cjk_boundary", "世界你好吗啊", 5, "世界你好…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
