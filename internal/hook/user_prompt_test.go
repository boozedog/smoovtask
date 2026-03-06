package hook

import (
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHandleUserPrompt(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-user-prompt",
		CWD:       projectPath,
		Prompt:    "fix the login bug",
	}

	if _, err := HandleUserPrompt(input); err != nil {
		t.Fatalf("HandleUserPrompt() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	if ev.Event != event.HookUserPrompt {
		t.Errorf("event type = %q, want %q", ev.Event, event.HookUserPrompt)
	}
	if ev.RunID != "sess-user-prompt" {
		t.Errorf("run_id = %q, want %q", ev.RunID, "sess-user-prompt")
	}
	if ev.Project != "test-project" {
		t.Errorf("project = %q, want %q", ev.Project, "test-project")
	}
	if ev.Actor != "user" {
		t.Errorf("actor = %q, want %q", ev.Actor, "user")
	}
}

func TestHandleUserPromptNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
		Prompt:    "hello",
	}

	if _, err := HandleUserPrompt(input); err != nil {
		t.Fatalf("HandleUserPrompt() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	if ev.Event != event.HookUserPrompt {
		t.Errorf("event type = %q, want %q", ev.Event, event.HookUserPrompt)
	}
	if ev.RunID != "sess-no-proj" {
		t.Errorf("run_id = %q, want %q", ev.RunID, "sess-no-proj")
	}
	if ev.Actor != "user" {
		t.Errorf("actor = %q, want %q", ev.Actor, "user")
	}
}

func TestHandleUserPromptWithActiveTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create an in-progress ticket assigned to our session
	store := ticket.NewStore(env.projectsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_active",
		Title:    "Active ticket",
		Project:  "test-project",
		Status:   ticket.StatusInProgress,
		Assignee: "sess-active",
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
	}
	if err := store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	input := &Input{
		SessionID: "sess-active",
		CWD:       projectPath,
		Prompt:    "yes, use JWT for auth",
	}

	out, err := HandleUserPrompt(input)
	if err != nil {
		t.Fatalf("HandleUserPrompt() error: %v", err)
	}

	if out.AdditionalContext == "" {
		t.Fatal("expected note reminder in AdditionalContext, got empty string")
	}
	if !strings.Contains(out.AdditionalContext, "add a note") {
		t.Errorf("AdditionalContext should mention adding a note, got: %s", out.AdditionalContext)
	}
}

func TestHandleUserPromptNoActiveTicket(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-no-ticket",
		CWD:       projectPath,
		Prompt:    "hello",
	}

	out, err := HandleUserPrompt(input)
	if err != nil {
		t.Fatalf("HandleUserPrompt() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no AdditionalContext without active ticket, got: %s", out.AdditionalContext)
	}
}

func TestHandleUserPromptNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
		Prompt:    "test prompt",
	}

	if _, err := HandleUserPrompt(input); err != nil {
		t.Fatalf("HandleUserPrompt() should not error on missing config, got: %v", err)
	}
}
