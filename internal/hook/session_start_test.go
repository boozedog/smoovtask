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
	if !strings.Contains(ctx, "Your run ID is sess-start-test") {
		t.Error("missing run ID in output")
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
	if !strings.Contains(ctx, "Your run ID is sess-minimal") {
		t.Error("missing run ID in output")
	}
	if !strings.Contains(ctx, "st list") {
		t.Error("missing quick reference commands")
	}
	if !strings.Contains(ctx, "st show") {
		t.Error("missing quick reference commands")
	}
	if !strings.Contains(ctx, "st <command> --help") {
		t.Error("missing help commands")
	}
}
