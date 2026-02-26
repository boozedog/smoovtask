package hook

import (
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
)

func TestHandlePreTool(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-pre-tool",
		CWD:       projectPath,
		ToolName:  "Read",
	}

	if err := HandlePreTool(input); err != nil {
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

	if err := HandlePreTool(input); err != nil {
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

	if err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() should not error on missing config, got: %v", err)
	}
}
