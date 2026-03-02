package hook

import (
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHandlePreTool(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-pre-tool",
		CWD:       projectPath,
		ToolName:  "Read",
	}

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPreTool, "sess-pre-tool", "test-project")

	// Verify tool name is in event data.
	if ev.Data == nil {
		t.Fatal("event data is nil, expected tool field")
	}
	tool, ok := ev.Data["tool"]
	if !ok {
		t.Fatal("event data missing 'tool' key")
	}
	if tool != "Read" {
		t.Errorf("event data tool = %q, want %q", tool, "Read")
	}
}

func TestHandlePreToolNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
		ToolName:  "Bash",
	}

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPreTool, "sess-no-proj", "")
}

func TestHandlePreToolNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
		ToolName:  "Edit",
	}

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() should not error on missing config, got: %v", err)
	}
}

func TestHandlePreToolBlocksWriteWithoutTicket(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-no-ticket",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if !strings.Contains(out.AdditionalContext, "BLOCKED") {
		t.Error("expected block guidance when editing without active ticket")
	}
	if !strings.Contains(out.AdditionalContext, "st pick") {
		t.Error("block guidance should mention st pick")
	}
	if !strings.Contains(out.AdditionalContext, "--run-id sess-no-ticket") {
		t.Error("block guidance should include the run id")
	}
	if out.Decision == nil {
		t.Fatal("expected deny decision when editing without active ticket")
	}
	if out.Decision.Behavior != "deny" {
		t.Errorf("decision behavior = %q, want %q", out.Decision.Behavior, "deny")
	}
	if !strings.Contains(out.Decision.Reason, "st pick") {
		t.Error("decision reason should include remediation")
	}
}

func TestHandlePreToolNoWarningWithActiveTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create an in-progress ticket assigned to our session
	store := ticket.NewStore(env.ticketsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_active",
		Title:    "Active ticket",
		Project:  "test-project",
		Status:   ticket.StatusInProgress,
		Assignee: "sess-with-ticket",
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		SessionID: "sess-with-ticket",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no warning with active ticket, got: %q", out.AdditionalContext)
	}
	if out.Decision != nil {
		t.Errorf("expected no deny decision with active ticket, got: %#v", out.Decision)
	}
}

func TestHandlePreToolNoWarningWithReworkTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a REWORK ticket assigned to our session
	store := ticket.NewStore(env.ticketsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_rework",
		Title:    "Rework ticket",
		Project:  "test-project",
		Status:   ticket.StatusRework,
		Assignee: "sess-with-rework",
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		SessionID: "sess-with-rework",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no warning with REWORK ticket, got: %q", out.AdditionalContext)
	}
	if out.Decision != nil {
		t.Errorf("expected no deny decision with REWORK ticket, got: %#v", out.Decision)
	}
}

func TestHandlePreToolNoWarningForReadTools(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-no-ticket",
		CWD:       projectPath,
		ToolName:  "Read",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no warning for Read tool, got: %q", out.AdditionalContext)
	}
	if out.Decision != nil {
		t.Errorf("expected no deny decision for Read tool, got: %#v", out.Decision)
	}
}

func TestHandlePreToolBlocksWriteWithoutRunID(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "",
		CWD:       projectPath,
		ToolName:  "Write",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if !strings.Contains(out.AdditionalContext, "BLOCKED") {
		t.Fatal("expected blocked guidance when run id is missing")
	}
	if !strings.Contains(out.AdditionalContext, "st list") {
		t.Fatal("expected fallback guidance with st list")
	}
	if out.Decision == nil || out.Decision.Behavior != "deny" {
		t.Fatalf("expected deny decision, got %#v", out.Decision)
	}
}
