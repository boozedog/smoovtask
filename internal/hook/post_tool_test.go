package hook

import (
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
)

func TestHandlePostTool(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-post-tool",
		CWD:       projectPath,
		ToolName:  "Write",
	}

	if err := HandlePostTool(input); err != nil {
		t.Fatalf("HandlePostTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPostTool, "sess-post-tool", "test-project")

	// Verify tool name is in event data.
	if ev.Data == nil {
		t.Fatal("event data is nil, expected tool field")
	}
	tool, ok := ev.Data["tool"]
	if !ok {
		t.Fatal("event data missing 'tool' key")
	}
	if tool != "Write" {
		t.Errorf("event data tool = %q, want %q", tool, "Write")
	}
}

func TestHandlePostToolNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
		ToolName:  "Grep",
	}

	if err := HandlePostTool(input); err != nil {
		t.Fatalf("HandlePostTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPostTool, "sess-no-proj", "")
}

func TestHandlePostToolNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
		ToolName:  "Bash",
	}

	if err := HandlePostTool(input); err != nil {
		t.Fatalf("HandlePostTool() should not error on missing config, got: %v", err)
	}
}
