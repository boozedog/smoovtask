package hook

import (
	"testing"

	"github.com/boozedog/smoovbrain/internal/event"
)

func TestHandlePermissionRequest(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-perm",
		CWD:       projectPath,
		ToolName:  "Bash",
	}

	out, err := HandlePermissionRequest(input)
	if err != nil {
		t.Fatalf("HandlePermissionRequest() error: %v", err)
	}

	// Should return empty Output (pass-through — no decision).
	if out.Decision != nil {
		t.Errorf("Decision should be nil (pass-through), got: %+v", out.Decision)
	}
	if out.AdditionalContext != "" {
		t.Errorf("AdditionalContext should be empty, got: %q", out.AdditionalContext)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPermissionReq, "sess-perm", "test-project")
}

func TestHandlePermissionRequestNoProject(t *testing.T) {
	env := setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	out, err := HandlePermissionRequest(input)
	if err != nil {
		t.Fatalf("HandlePermissionRequest() error: %v", err)
	}

	if out.Decision != nil {
		t.Errorf("Decision should be nil, got: %+v", out.Decision)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPermissionReq, "sess-no-proj", "")
}

func TestHandlePermissionRequestNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	input := &Input{
		SessionID: "sess-no-config",
		CWD:       "/tmp",
	}

	out, err := HandlePermissionRequest(input)
	if err != nil {
		t.Fatalf("HandlePermissionRequest() should not error on missing config, got: %v", err)
	}

	if out.Decision != nil {
		t.Errorf("Decision should be nil, got: %+v", out.Decision)
	}
}
