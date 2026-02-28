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
	if !strings.Contains(ctx, "Implement feature") {
		t.Error("missing ticket title in context")
	}
	if !strings.Contains(ctx, "test-project") {
		t.Error("missing project in context")
	}
	if !strings.Contains(ctx, "P2") {
		t.Error("missing priority in context")
	}
	if !strings.Contains(ctx, "open") {
		t.Error("missing status in context")
	}
	if !strings.Contains(ctx, "st note --ticket st_open01") {
		t.Error("missing st note --ticket instruction")
	}
	// Should NOT contain workflow directives
	if strings.Contains(ctx, "REQUIRED") {
		t.Error("subagent context should not contain REQUIRED directives")
	}
	if strings.Contains(ctx, "MUST") {
		t.Error("subagent context should not contain MUST directives")
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
	if !strings.Contains(ctx, "Review this PR") {
		t.Error("missing ticket title in context")
	}
	if !strings.Contains(ctx, "REVIEW") {
		t.Error("missing status in context")
	}
	if !strings.Contains(ctx, "st note --ticket st_revi01") {
		t.Error("missing st note --ticket instruction")
	}
	// Should NOT contain workflow directives
	if strings.Contains(ctx, "REQUIRED") {
		t.Error("subagent context should not contain REQUIRED directives")
	}
	if strings.Contains(ctx, "MUST") {
		t.Error("subagent context should not contain MUST directives")
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
