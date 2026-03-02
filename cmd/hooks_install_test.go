package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHooksInstall_FreshInstall(t *testing.T) {
	// Use a temp dir for HOME so we don't touch real settings
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := newTestEnv(t)
	_ = env

	out, err := env.runCmd(t, "hooks", "install")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Installed") {
		t.Errorf("output = %q, want substring %q", out, "Installed")
	}

	// Verify settings file was created
	settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks key missing or not a map")
	}

	// Check that expected hook events were installed
	expectedEvents := []string{
		"SessionStart", "PreToolUse", "PostToolUse",
		"SubagentStart", "SubagentStop", "Stop",
		"TaskCompleted", "TeammateIdle",
		"PermissionRequest", "SessionEnd",
	}
	for _, eventName := range expectedEvents {
		if _, ok := hooks[eventName]; !ok {
			t.Errorf("missing hook event %q", eventName)
		}
	}

	// Verify at least one hook has an "st hook" command
	foundSB := false
	for _, eventName := range expectedEvents {
		groups, ok := hooks[eventName].([]any)
		if !ok {
			continue
		}
		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				continue
			}
			hookList, ok := group["hooks"].([]any)
			if !ok {
				continue
			}
			for _, h := range hookList {
				entry, ok := h.(map[string]any)
				if !ok {
					continue
				}
				cmd, _ := entry["command"].(string)
				if strings.HasPrefix(cmd, "st hook") {
					foundSB = true
				}
			}
		}
	}
	if !foundSB {
		t.Error("no 'st hook' command found in installed hooks")
	}
}

func TestHooksInstall_Idempotent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := newTestEnv(t)
	_ = env

	// Install once
	_, err := env.runCmd(t, "hooks", "install")
	if err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Install again
	out, err := env.runCmd(t, "hooks", "install")
	if err != nil {
		t.Fatalf("second install: %v", err)
	}

	if !strings.Contains(out, "All smoovtask Claude hooks already installed") {
		t.Errorf("output = %q, want substring %q", out, "All smoovtask Claude hooks already installed")
	}

	// Verify settings file hasn't duplicated hooks
	settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	hooks, _ := settings["hooks"].(map[string]any)
	// Each hook event should have exactly 1 group (no duplicates)
	for eventName, groups := range hooks {
		groupList, ok := groups.([]any)
		if !ok {
			continue
		}
		if len(groupList) != 1 {
			t.Errorf("event %q has %d groups, want 1 (no duplicates)", eventName, len(groupList))
		}
	}
}

func TestHooksInstall_PreservesExistingHooks(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create existing settings with a non-st hook
	claudeDir := filepath.Join(tmpHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("create claude dir: %v", err)
	}

	existingSettings := map[string]any{
		"some_other_key": "preserved",
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "my-custom-hook",
						},
					},
				},
			},
		},
	}
	data, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		t.Fatalf("marshal existing settings: %v", err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write existing settings: %v", err)
	}

	env := newTestEnv(t)
	_ = env

	_, err = env.runCmd(t, "hooks", "install")
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	// Verify existing settings are preserved
	updatedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var updated map[string]any
	if err := json.Unmarshal(updatedData, &updated); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	// Check that other keys are preserved
	if updated["some_other_key"] != "preserved" {
		t.Errorf("some_other_key = %v, want %q", updated["some_other_key"], "preserved")
	}

	// Check that the custom hook in PreToolUse is preserved
	hooks, _ := updated["hooks"].(map[string]any)
	preToolGroups, ok := hooks["PreToolUse"].([]any)
	if !ok {
		t.Fatal("PreToolUse hooks missing")
	}

	// Should have 2 groups: the existing custom hook + the st hook
	if len(preToolGroups) != 2 {
		t.Fatalf("PreToolUse has %d groups, want 2 (custom + st)", len(preToolGroups))
	}

	// Verify the custom hook is still there
	firstGroup, ok := preToolGroups[0].(map[string]any)
	if !ok {
		t.Fatal("first PreToolUse group is not a map")
	}
	hookList, ok := firstGroup["hooks"].([]any)
	if !ok {
		t.Fatal("first group hooks not found")
	}
	firstHook, ok := hookList[0].(map[string]any)
	if !ok {
		t.Fatal("first hook entry is not a map")
	}
	if firstHook["command"] != "my-custom-hook" {
		t.Errorf("first hook command = %v, want %q", firstHook["command"], "my-custom-hook")
	}
}

func TestHooksInstall_PIExtension(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	env := newTestEnv(t)
	_ = env

	out, err := env.runCmd(t, "hooks", "install", "--agents", "pi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Installed pi extension") {
		t.Errorf("output = %q, want substring %q", out, "Installed pi extension")
	}

	extensionPath := filepath.Join(tmpHome, ".pi", "agent", "extensions", "smoovtask-hooks.ts")
	data, err := os.ReadFile(extensionPath)
	if err != nil {
		t.Fatalf("read extension: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "st', ['hook', 'pi-event']") {
		t.Errorf("extension content missing pi hook bridge")
	}
	if !strings.Contains(content, "basename(path, '.jsonl')") {
		t.Errorf("extension content missing session id normalization")
	}
	if !strings.Contains(content, "uuidPattern.test(suffix)") {
		t.Errorf("extension content missing uuid suffix extraction")
	}
}

func TestExpandAgents_BothIncludesPI(t *testing.T) {
	got := expandAgents([]string{"both"})
	want := []string{"claude", "opencode", "pi"}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHookBridgesDoNotWrapContextTwice(t *testing.T) {
	if strings.Contains(opencodePluginCode, "<smoovtask>") {
		t.Fatal("opencode plugin should not add wrapper tags; hook context is already wrapped")
	}
	if strings.Contains(piExtensionCode, "<smoovtask>") {
		t.Fatal("pi extension should not add wrapper tags; hook context is already wrapped")
	}
	if !strings.Contains(opencodePluginCode, "output.system.push(cachedContext)") {
		t.Fatal("opencode plugin should push cachedContext directly")
	}
	if !strings.Contains(piExtensionCode, "event.systemPrompt + '\\n\\n' + cachedContext") {
		t.Fatal("pi extension should append cachedContext directly")
	}
	if !strings.Contains(opencodePluginCode, "result.hookSpecificOutput.behavior === 'deny'") {
		t.Fatal("opencode plugin should block denied pre-tool decisions")
	}
	if !strings.Contains(opencodePluginCode, "return { block: true, reason: result.hookSpecificOutput.reason || 'Blocked by smoovtask' }") {
		t.Fatal("opencode plugin should return block payload on denied decision")
	}
	if !strings.Contains(opencodePluginCode, "tool.execute.before', 'tool.execute.after") {
		t.Fatal("opencode plugin should include tool event fallback in handled event list")
	}
	if !strings.Contains(opencodePluginCode, "const looksLikeTool = eventType.includes('tool')") {
		t.Fatal("opencode plugin should detect tool-like events from generic event bus")
	}
	if !strings.Contains(opencodePluginCode, "payload.type = looksBefore ? 'tool.execute.before' : 'tool.execute.after'") {
		t.Fatal("opencode plugin should normalize tool fallback events to canonical hook types")
	}
	if !strings.Contains(opencodePluginCode, "payload.tool_name = toolName") {
		t.Fatal("opencode plugin should include normalized tool name for fallback events")
	}
	if !strings.Contains(opencodePluginCode, "if (eventType === 'session.status')") {
		t.Fatal("opencode plugin should synthesize tool activity from session.status events")
	}
	if !strings.Contains(opencodePluginCode, "props.status && typeof props.status === 'object'") {
		t.Fatal("opencode plugin should parse object-form session.status payloads")
	}
	if !strings.Contains(opencodePluginCode, "statusType === 'running' || statusType === 'working' || statusType === 'busy'") {
		t.Fatal("opencode plugin should treat active session statuses as pre-tool")
	}
	if !strings.Contains(opencodePluginCode, "statusType === 'idle' || statusType === 'waiting'") {
		t.Fatal("opencode plugin should treat idle session statuses as post-tool")
	}
	if !strings.Contains(opencodePluginCode, "if (eventType === 'message.part.updated')") {
		t.Fatal("opencode plugin should bridge message part updates into tool activity")
	}
	if !strings.Contains(opencodePluginCode, "if (partType === 'tool')") {
		t.Fatal("opencode plugin should only synthesize tool hooks for tool message parts")
	}
	if !strings.Contains(opencodePluginCode, "partStatus === 'pending' || partStatus === 'running'") {
		t.Fatal("opencode plugin should map pending/running tool parts to pre-tool")
	}
	if !strings.Contains(opencodePluginCode, "partStatus === 'completed' || partStatus === 'done' || partStatus === 'success' || partStatus === 'failed' || partStatus === 'error' || partStatus === 'cancelled'") {
		t.Fatal("opencode plugin should map terminal tool part statuses to post-tool")
	}
	if !strings.Contains(piExtensionCode, "result.hookSpecificOutput.behavior === 'deny'") {
		t.Fatal("pi extension should block denied pre-tool decisions")
	}
}
