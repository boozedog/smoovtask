package hook

import (
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
)

func TestHandleStop(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-stop",
		CWD:       projectPath,
	}

	if err := HandleStop(input); err != nil {
		t.Fatalf("HandleStop() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookStop, "sess-stop", "test-project")
}

func TestHandleStopNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	if err := HandleStop(input); err != nil {
		t.Fatalf("HandleStop() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookStop, "sess-no-proj", "")
}

func TestHandleStopNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
	}

	if err := HandleStop(input); err != nil {
		t.Fatalf("HandleStop() should not error on missing config, got: %v", err)
	}
}
