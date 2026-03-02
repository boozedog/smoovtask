package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
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

func TestHandlePermissionRequestDenyRule(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a rules dir with a deny rule
	rulesDir := filepath.Join(env.ConfigDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rule := `name: test-deny
priority: 100
event: PreToolUse
rules:
  - name: deny-push
    match:
      tool: Bash
      command: "git push"
    action: deny
    message: "git push is blocked"
`
	if err := os.WriteFile(filepath.Join(rulesDir, "01-test.yaml"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	input := &Input{
		SessionID:     "sess-deny",
		CWD:           projectPath,
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     map[string]any{"command": "git push origin main"},
	}

	out, err := HandlePermissionRequest(input)
	if err != nil {
		t.Fatalf("HandlePermissionRequest() error: %v", err)
	}

	if out.Decision == nil {
		t.Fatal("expected a deny decision, got nil")
	}
	if out.Decision.Behavior != "deny" {
		t.Errorf("expected behavior=deny, got %q", out.Decision.Behavior)
	}
	if out.Decision.Reason != "git push is blocked" {
		t.Errorf("expected reason 'git push is blocked', got %q", out.Decision.Reason)
	}
}

func TestHandlePermissionRequestAllowRule(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a rules dir with an allow rule
	rulesDir := filepath.Join(env.ConfigDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rule := `name: test-allow
priority: 50
event: PreToolUse
rules:
  - name: allow-git
    match:
      tool: Bash
      command: "^git\\s+"
    action: allow
    message: "git commands allowed"
`
	if err := os.WriteFile(filepath.Join(rulesDir, "01-test.yaml"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	input := &Input{
		SessionID:     "sess-allow",
		CWD:           projectPath,
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     map[string]any{"command": "git status"},
	}

	out, err := HandlePermissionRequest(input)
	if err != nil {
		t.Fatalf("HandlePermissionRequest() error: %v", err)
	}

	if out.Decision == nil {
		t.Fatal("expected an allow decision, got nil")
	}
	if out.Decision.Behavior != "allow" {
		t.Errorf("expected behavior=allow, got %q", out.Decision.Behavior)
	}
}

func TestHandlePermissionRequestNoRulesPassthrough(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	// No rules dir created — should passthrough
	input := &Input{
		SessionID:     "sess-norules",
		CWD:           projectPath,
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     map[string]any{"command": "ls -la"},
	}

	out, err := HandlePermissionRequest(input)
	if err != nil {
		t.Fatalf("HandlePermissionRequest() error: %v", err)
	}

	if out.Decision != nil {
		t.Errorf("expected nil decision (passthrough), got %+v", out.Decision)
	}
}
