package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHandlePreTool(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-pre-tool",
		CWD:       projectPath,
		ToolName:  "Read",
	}

	if _, err := HandlePreTool(input); err != nil {
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

	if _, err := HandlePreTool(input); err != nil {
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

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() should not error on missing config, got: %v", err)
	}
}

func TestHandlePreToolBlocksWriteWithoutTicket(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-no-ticket",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if !strings.Contains(out.AdditionalContext, "BLOCKED") {
		t.Error("expected block guidance when editing without active ticket")
	}
	if !strings.Contains(out.AdditionalContext, "st pick") {
		t.Error("block guidance should mention st pick")
	}
	if !strings.Contains(out.AdditionalContext, "--run-id sess-no-ticket") {
		t.Error("block guidance should include the run id")
	}
	if out.Decision == nil {
		t.Fatal("expected deny decision when editing without active ticket")
	}
	if out.Decision.Behavior != "deny" {
		t.Errorf("decision behavior = %q, want %q", out.Decision.Behavior, "deny")
	}
	if !strings.Contains(out.Decision.Reason, "st pick") {
		t.Error("decision reason should include remediation")
	}
}

func TestHandlePreToolNoWarningWithActiveTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create an in-progress ticket assigned to our session
	store := ticket.NewStore(env.projectsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_active",
		Title:    "Active ticket",
		Project:  "test-project",
		Status:   ticket.StatusInProgress,
		Assignee: "sess-with-ticket",
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		SessionID: "sess-with-ticket",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no warning with active ticket, got: %q", out.AdditionalContext)
	}
	if out.Decision != nil {
		t.Errorf("expected no deny decision with active ticket, got: %#v", out.Decision)
	}
}

func TestHandlePreToolNoWarningWithReworkTicket(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a REWORK ticket assigned to our session
	store := ticket.NewStore(env.projectsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_rework",
		Title:    "Rework ticket",
		Project:  "test-project",
		Status:   ticket.StatusRework,
		Assignee: "sess-with-rework",
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		SessionID: "sess-with-rework",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no warning with REWORK ticket, got: %q", out.AdditionalContext)
	}
	if out.Decision != nil {
		t.Errorf("expected no deny decision with REWORK ticket, got: %#v", out.Decision)
	}
}

func TestHandlePreToolNoWarningForReadTools(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-no-ticket",
		CWD:       projectPath,
		ToolName:  "Read",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.AdditionalContext != "" {
		t.Errorf("expected no warning for Read tool, got: %q", out.AdditionalContext)
	}
	if out.Decision != nil {
		t.Errorf("expected no deny decision for Read tool, got: %#v", out.Decision)
	}
}

func TestHandlePreToolBashLogsCommand(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-bash",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolInput: map[string]any{
			"command": "echo hello",
		},
	}

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookPreTool, "sess-bash", "test-project")

	cmd, ok := ev.Data["command"]
	if !ok {
		t.Fatal("event data missing 'command' key for Bash tool")
	}
	if cmd != "echo hello" {
		t.Errorf("event data command = %q, want %q", cmd, "echo hello")
	}
}

func TestHandlePreToolNonBashNoCommand(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-read",
		CWD:       projectPath,
		ToolName:  "Read",
		ToolInput: map[string]any{
			"file_path": "/tmp/foo.go",
		},
	}

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	if _, ok := ev.Data["command"]; ok {
		t.Error("non-Bash tool should not have 'command' in event data")
	}
}

func TestHandlePreToolBlocksWriteWithoutRunID(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "",
		CWD:       projectPath,
		ToolName:  "Write",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if !strings.Contains(out.AdditionalContext, "BLOCKED") {
		t.Fatal("expected blocked guidance when run id is missing")
	}
	if !strings.Contains(out.AdditionalContext, "st list") {
		t.Fatal("expected fallback guidance with st list")
	}
	if out.Decision == nil || out.Decision.Behavior != "deny" {
		t.Fatalf("expected deny decision, got %#v", out.Decision)
	}
}

func TestHandlePreToolDenyRule(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

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
		SessionID: "sess-deny",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "git push origin main"},
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
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

func TestHandlePreToolAllowRule(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

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
		SessionID: "sess-allow",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "git status"},
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.Decision == nil {
		t.Fatal("expected an allow decision, got nil")
	}
	if out.Decision.Behavior != "allow" {
		t.Errorf("expected behavior=allow, got %q", out.Decision.Behavior)
	}
}

func TestHandlePreToolNoRulesPassthrough(t *testing.T) {
	projectPath := t.TempDir()
	setupTestEnv(t, projectPath)

	input := &Input{
		SessionID: "sess-norules",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "ls -la"},
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.Decision != nil {
		t.Errorf("expected nil decision (passthrough), got %+v", out.Decision)
	}
}

func TestHandlePreToolAllowRuleLogsDecision(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	rulesDir := filepath.Join(env.ConfigDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rule := `name: test-allow
priority: 50
event: PreToolUse
rules:
  - name: allow-st
    match:
      tool: Bash
      command: "^st\\s+"
    action: allow
    message: "st commands allowed"
`
	if err := os.WriteFile(filepath.Join(rulesDir, "01-test.yaml"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	input := &Input{
		SessionID: "sess-decision-log",
		CWD:       projectPath,
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "st status review"},
	}

	if _, err := HandlePreTool(input); err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	events := readTodayEvents(t, env.EventsDir)
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (pre-tool + decision), got %d", len(events))
	}

	decision := events[1]
	if decision.Event != event.HookRuleDecision {
		t.Errorf("event type = %q, want %q", decision.Event, event.HookRuleDecision)
	}
	if decision.Data["decision"] != "allow" {
		t.Errorf("decision = %v, want allow", decision.Data["decision"])
	}
	if decision.Data["ruleset"] != "test-allow" {
		t.Errorf("ruleset = %v, want test-allow", decision.Data["ruleset"])
	}
	if decision.Data["rule"] != "allow-st" {
		t.Errorf("rule = %v, want allow-st", decision.Data["rule"])
	}
}

func TestHandlePreToolTicketDenyTakesPriorityOverRuleAllow(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a rule that would allow Edit
	rulesDir := filepath.Join(env.ConfigDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rule := `name: test-allow-edit
priority: 50
event: PreToolUse
rules:
  - name: allow-edit
    match:
      tool: Edit
    action: allow
    message: "edit allowed"
`
	if err := os.WriteFile(filepath.Join(rulesDir, "01-test.yaml"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	// No ticket — should still be blocked despite allow rule
	input := &Input{
		SessionID: "sess-no-ticket",
		CWD:       projectPath,
		ToolName:  "Edit",
	}

	out, err := HandlePreTool(input)
	if err != nil {
		t.Fatalf("HandlePreTool() error: %v", err)
	}

	if out.Decision == nil || out.Decision.Behavior != "deny" {
		t.Fatalf("expected deny (ticket check), got %#v", out.Decision)
	}
	if !strings.Contains(out.AdditionalContext, "BLOCKED") {
		t.Error("expected BLOCKED message from ticket check")
	}
}
