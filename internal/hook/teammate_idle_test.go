package hook

import (
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
)

func TestHandleTeammateIdle(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-idle",
		CWD:       projectPath,
	}

	if err := HandleTeammateIdle(input); err != nil {
		t.Fatalf("HandleTeammateIdle() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookTeammateIdle, "sess-idle", "test-project")
}

func TestHandleTeammateIdleNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	if err := HandleTeammateIdle(input); err != nil {
		t.Fatalf("HandleTeammateIdle() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookTeammateIdle, "sess-no-proj", "")
}

func TestHandleTeammateIdleNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
	}

	if err := HandleTeammateIdle(input); err != nil {
		t.Fatalf("HandleTeammateIdle() should not error on missing config, got: %v", err)
	}
}
