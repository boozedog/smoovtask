package hook

import (
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
)

func TestHandleTaskCompleted(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-task-done",
		CWD:       projectPath,
	}

	if err := HandleTaskCompleted(input); err != nil {
		t.Fatalf("HandleTaskCompleted() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookTaskCompleted, "sess-task-done", "test-project")
}

func TestHandleTaskCompletedNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	if err := HandleTaskCompleted(input); err != nil {
		t.Fatalf("HandleTaskCompleted() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookTaskCompleted, "sess-no-proj", "")
}

func TestHandleTaskCompletedNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
	}

	if err := HandleTaskCompleted(input); err != nil {
		t.Fatalf("HandleTaskCompleted() should not error on missing config, got: %v", err)
	}
}
