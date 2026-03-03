package hook

import (
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
)

func TestHandleUserPrompt(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-user-prompt",
		CWD:       projectPath,
		Prompt:    "fix the login bug",
	}

	if err := HandleUserPrompt(input); err != nil {
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

	if err := HandleUserPrompt(input); err != nil {
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

func TestHandleUserPromptNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
		Prompt:    "test prompt",
	}

	if err := HandleUserPrompt(input); err != nil {
		t.Fatalf("HandleUserPrompt() should not error on missing config, got: %v", err)
	}
}
