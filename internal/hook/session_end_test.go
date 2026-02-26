package hook

import (
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
)

func TestHandleSessionEnd(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-end",
		CWD:       projectPath,
	}

	if err := HandleSessionEnd(input); err != nil {
		t.Fatalf("HandleSessionEnd() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookSessionEnd, "sess-end", "test-project")
}

func TestHandleSessionEndNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	if err := HandleSessionEnd(input); err != nil {
		t.Fatalf("HandleSessionEnd() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookSessionEnd, "sess-no-proj", "")
}

func TestHandleSessionEndNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
	}

	if err := HandleSessionEnd(input); err != nil {
		t.Fatalf("HandleSessionEnd() should not error on missing config, got: %v", err)
	}
}
