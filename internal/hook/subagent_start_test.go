package hook

import (
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestTicketIDPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Work on st_a7Kx2m: add rate limiting", "st_a7Kx2m"},
		{"st_Qr9fZw is the target", "st_Qr9fZw"},
		{"no ticket here", ""},
		{"st_ too short", ""},
		{"st_!@#$%^ invalid chars", ""},
	}

	for _, tt := range tests {
		got := ticketIDPattern.FindString(tt.input)
		if got != tt.want {
			t.Errorf("FindString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHandleSubagentStartOpenTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	store := ticket.NewStore(env.ticketsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_open01",
		Title:    "Implement feature",
		Project:  "test-project",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		TaskPrompt: "Work on st_open01: implement feature",
	}

	out, err := HandleSubagentStart(input)
	if err != nil {
		t.Fatalf("HandleSubagentStart() error: %v", err)
	}

	ctx := out.AdditionalContext
	if !strings.Contains(ctx, "st_open01") {
		t.Error("missing ticket ID in context")
	}
	if !strings.Contains(ctx, "st pick st_open01") {
		t.Error("missing st pick instruction for OPEN ticket")
	}
	if !strings.Contains(ctx, "st note") {
		t.Error("missing st note instruction")
	}
	if !strings.Contains(ctx, "st status --ticket st_open01 --run-id") {
		t.Error("missing st status --ticket instruction")
	}
	if !strings.Contains(ctx, "st note --ticket st_open01") {
		t.Error("missing st note --ticket instruction")
	}
	if !strings.Contains(ctx, "ALWAYS pass --ticket and --run-id") {
		t.Error("missing ALWAYS pass --ticket and --run-id warning")
	}
	if strings.Contains(ctx, "st review") {
		t.Error("OPEN ticket should not get reviewer workflow")
	}
}

func TestHandleSubagentStartReviewTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	store := ticket.NewStore(env.ticketsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_revi01",
		Title:    "Review this PR",
		Project:  "test-project",
		Status:   ticket.StatusReview,
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		TaskPrompt: "Review st_revi01: check the PR",
	}

	out, err := HandleSubagentStart(input)
	if err != nil {
		t.Fatalf("HandleSubagentStart() error: %v", err)
	}

	ctx := out.AdditionalContext
	if !strings.Contains(ctx, "st_revi01") {
		t.Error("missing ticket ID in context")
	}
	if !strings.Contains(ctx, "REVIEW") {
		t.Error("missing REVIEW label in context")
	}
	if !strings.Contains(ctx, "st review --ticket st_revi01") {
		t.Error("missing st review --ticket instruction for REVIEW ticket")
	}
	if !strings.Contains(ctx, "st note") {
		t.Error("missing st note instruction")
	}
	if !strings.Contains(ctx, "st status --ticket st_revi01 --run-id") {
		t.Error("missing st status --ticket instruction")
	}
	if !strings.Contains(ctx, "done") {
		t.Error("missing done instruction")
	}
	if !strings.Contains(ctx, "rework") {
		t.Error("missing rework instruction")
	}
	if !strings.Contains(ctx, "st note --ticket st_revi01 --run-id") {
		t.Error("missing st note --ticket instruction")
	}
	if !strings.Contains(ctx, "ALWAYS pass --ticket and --run-id") {
		t.Error("missing ALWAYS pass --ticket and --run-id warning")
	}
	if !strings.Contains(ctx, "Do NOT approve or reject") {
		t.Error("missing warning about documenting findings first")
	}
	if strings.Contains(ctx, "st pick") {
		t.Error("REVIEW ticket should not get implementer workflow with st pick")
	}
}

func TestHandleSubagentStartNoTicket(t *testing.T) {
	setupTestEnv(t, t.TempDir())

	input := &Input{
		TaskPrompt: "Do something without a ticket reference",
	}

	out, err := HandleSubagentStart(input)
	if err != nil {
		t.Fatalf("HandleSubagentStart() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected empty context, got: %q", out.AdditionalContext)
	}
}
