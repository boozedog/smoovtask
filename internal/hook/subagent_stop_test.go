package hook

import (
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
)

func TestHandleSubagentStop(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-subagent-stop",
		CWD:       projectPath,
	}

	if err := HandleSubagentStop(input); err != nil {
		t.Fatalf("HandleSubagentStop() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookSubagentStop, "sess-subagent-stop", "test-project")
}

func TestHandleSubagentStopNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	if err := HandleSubagentStop(input); err != nil {
		t.Fatalf("HandleSubagentStop() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookSubagentStop, "sess-no-proj", "")
}

func TestHandleSubagentStopNoConfig(t *testing.T) {
	// Point HOME at a dir with no config — handler should return nil (not error).
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
	}

	if err := HandleSubagentStop(input); err != nil {
		t.Fatalf("HandleSubagentStop() should not error on missing config, got: %v", err)
	}
}
