package hook

import (
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHandleSessionStartLogsEvent(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a ticket so the handler has something to count
	store := ticket.NewStore(env.ticketsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_test01",
		Title:    "Test ticket",
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
		SessionID: "sess-start-test",
		CWD:       projectPath,
	}

	out, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookSessionStart, "sess-start-test", "test-project")

	openCount, _ := ev.Data["open_count"].(float64)
	if openCount != 1 {
		t.Errorf("open_count = %v, want 1", ev.Data["open_count"])
	}

	// Output should be minimal — no ticket listings
	ctx := out.AdditionalContext
	if !strings.Contains(ctx, "Your run ID is `sess-start-test") {
		t.Error("missing run ID in output")
	}
	if !strings.Contains(ctx, "<smoovtask>") || !strings.Contains(ctx, "</smoovtask>") {
		t.Error("missing smoovtask context envelope")
	}
	if strings.Contains(ctx, "st_test01") {
		t.Error("output should not contain ticket IDs")
	}
	if strings.Contains(ctx, "tickets ready") {
		t.Error("output should not contain ticket counts")
	}
}

func TestHandleSessionStartNoProject(t *testing.T) {
	setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	_, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}
}

func TestHandleSessionStartMinimalOutput(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-minimal",
		CWD:       projectPath,
	}

	out, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}

	ctx := out.AdditionalContext
	if !strings.Contains(ctx, "project called test-project") {
		t.Error("missing project name in output")
	}
	if !strings.Contains(ctx, "<smoovtask>") || !strings.Contains(ctx, "</smoovtask>") {
		t.Error("missing smoovtask context envelope")
	}
	if !strings.Contains(ctx, "Your run ID is `sess-minimal") {
		t.Error("missing run ID in output")
	}
	if !strings.Contains(ctx, "st list --run-id <run-id>") {
		t.Error("missing quick reference commands with run-id placeholder")
	}
	if !strings.Contains(ctx, "st show <ticket-id> --run-id <run-id>") {
		t.Error("missing quick reference commands with run-id placeholder")
	}
	if !strings.Contains(ctx, "st handoff <ticket-id> --run-id <run-id>") {
		t.Error("missing handoff command in implementing quick reference")
	}
	if !strings.Contains(ctx, "st --help") {
		t.Error("missing help reference")
	}
	if !strings.Contains(ctx, "Use heredoc for all notes content") {
		t.Error("missing heredoc note guidance")
	}
	if !strings.Contains(ctx, "do not guess") {
		t.Error("missing instruction to ask user about role")
	}
	if !strings.Contains(ctx, "st status review` moves work to `REVIEW`") {
		t.Error("missing clarification for REVIEW transition")
	}
	if !strings.Contains(ctx, "st status human-review") {
		t.Error("missing HUMAN-REVIEW handoff command")
	}
}

func TestHandleSessionStartRoleAwareImplementer(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)
	t.Setenv("ST_ROLE", "implementer")

	input := &Input{
		SessionID: "sess-impl",
		CWD:       projectPath,
	}

	out, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}

	ctx := out.AdditionalContext
	if !strings.Contains(ctx, "Session role: `implementer`") {
		t.Error("missing implementer role banner")
	}
	if strings.Contains(ctx, "do not guess") {
		t.Error("implementer role should not include generic ask/guess prompt")
	}
	if !strings.Contains(ctx, "st pick <ticket-id> --run-id <run-id>") {
		t.Error("missing implementer pick command")
	}
}

func TestHandleSessionStartRoleAwareWorker(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)
	t.Setenv("ST_ROLE", "worker")

	input := &Input{
		SessionID: "sess-worker",
		CWD:       projectPath,
	}

	out, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}

	ctx := out.AdditionalContext
	if !strings.Contains(ctx, "Session role: `worker`") {
		t.Error("missing worker role banner")
	}
	if !strings.Contains(ctx, "st status <status> --run-id <run-id>") {
		t.Error("missing worker status command")
	}
	if strings.Contains(ctx, "st pick <ticket-id>") {
		t.Error("worker context should not include implementer commands")
	}
}
