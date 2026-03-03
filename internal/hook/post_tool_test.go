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

func TestHandlePostToolBashLogsExitCode(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-bash-post",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolResponse: map[string]any{
			"exit_code": float64(0),
		},
	}

	if err := HandlePostTool(input); err != nil {
		t.Fatalf("HandlePostTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPostTool, "sess-bash-post", "test-project")

	code, ok := ev.Data["exit_code"]
	if !ok {
		t.Fatal("event data missing 'exit_code' key for Bash tool")
	}
	if code != float64(0) {
		t.Errorf("event data exit_code = %v, want 0", code)
	}
}

func TestHandlePostToolBashNonZeroExitCode(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-bash-fail",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolResponse: map[string]any{
			"exit_code": float64(1),
		},
	}

	if err := HandlePostTool(input); err != nil {
		t.Fatalf("HandlePostTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)

	code, ok := ev.Data["exit_code"]
	if !ok {
		t.Fatal("event data missing 'exit_code' key for failed Bash tool")
	}
	if code != float64(1) {
		t.Errorf("event data exit_code = %v, want 1", code)
	}
}

func TestHandlePostToolNonBashNoExitCode(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-write-post",
		CWD:       projectPath,
		ToolName:  "Write",
	}

	if err := HandlePostTool(input); err != nil {
		t.Fatalf("HandlePostTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	if _, ok := ev.Data["exit_code"]; ok {
		t.Error("non-Bash tool should not have 'exit_code' in event data")
	}
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
