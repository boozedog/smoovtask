package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Claude Code hook integration",
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install smoovbrain hooks into Claude Code settings",
	Long:  `Installs smoovbrain hook commands into ~/.claude/settings.json. Existing hooks and settings are preserved.`,
	RunE:  runHooksInstall,
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	rootCmd.AddCommand(hooksCmd)
}

// hookEntry represents a single hook command entry.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Async   bool   `json:"async,omitempty"`
}

// hookGroup represents a group of hooks with an optional matcher.
type hookGroup struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []hookEntry `json:"hooks"`
}

// smoovbrainHooks returns the hook config that sb needs installed.
func smoovbrainHooks() map[string][]hookGroup {
	return map[string][]hookGroup{
		"SessionStart": {
			{
				Matcher: "startup",
				Hooks:   []hookEntry{{Type: "command", Command: "sb hook session-start"}},
			},
		},
		"PreToolUse": {
			{
				Matcher: "*",
				Hooks:   []hookEntry{{Type: "command", Command: "sb hook pre-tool", Async: true}},
			},
		},
		"PostToolUse": {
			{
				Matcher: "*",
				Hooks:   []hookEntry{{Type: "command", Command: "sb hook post-tool", Async: true}},
			},
		},
		"SubagentStart": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook subagent-start"}},
			},
		},
		"SubagentStop": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook subagent-stop", Async: true}},
			},
		},
		"TaskCompleted": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook task-completed", Async: true}},
			},
		},
		"TeammateIdle": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook teammate-idle", Async: true}},
			},
		},
		"Stop": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook stop", Async: true}},
			},
		},
		"PermissionRequest": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook permission-request"}},
			},
		},
		"SessionEnd": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "sb hook session-end", Async: true}},
			},
		},
	}
}

func runHooksInstall(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// Read existing settings (or start fresh)
	settings := make(map[string]any)
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", settingsPath, err)
	}

	// Get or create the hooks map
	existingHooks, _ := settings["hooks"].(map[string]any)
	if existingHooks == nil {
		existingHooks = make(map[string]any)
	}

	wanted := smoovbrainHooks()
	var installed, skipped []string

	for eventName, groups := range wanted {
		if hasSmoovbrainHook(existingHooks, eventName) {
			skipped = append(skipped, eventName)
			continue
		}

		// Convert groups to the right type for JSON
		var groupSlice []any
		for _, g := range groups {
			groupSlice = append(groupSlice, marshalHookGroup(g))
		}

		// Merge with any existing hooks for this event
		existing, _ := existingHooks[eventName].([]any)
		existingHooks[eventName] = append(existing, groupSlice...)
		installed = append(installed, eventName)
	}

	settings["hooks"] = existingHooks

	// Write settings back
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", settingsPath, err)
	}

	// Print summary
	if len(installed) > 0 {
		fmt.Printf("Installed %d hook(s):\n", len(installed))
		for _, name := range installed {
			fmt.Printf("  + %s\n", name)
		}
	}
	if len(skipped) > 0 {
		fmt.Printf("Already installed (%d hook(s)):\n", len(skipped))
		for _, name := range skipped {
			fmt.Printf("  = %s\n", name)
		}
	}
	if len(installed) == 0 {
		fmt.Println("All smoovbrain hooks already installed.")
	}

	fmt.Printf("\nSettings: %s\n", settingsPath)
	return nil
}

// hasSmoovbrainHook checks if an sb hook command already exists for the given event.
func hasSmoovbrainHook(hooks map[string]any, eventName string) bool {
	groups, ok := hooks[eventName].([]any)
	if !ok {
		return false
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
			if len(cmd) >= 7 && cmd[:7] == "sb hook" {
				return true
			}
		}
	}
	return false
}

// marshalHookGroup converts a hookGroup to a map[string]any for JSON merging.
func marshalHookGroup(g hookGroup) map[string]any {
	m := make(map[string]any)
	if g.Matcher != "" {
		m["matcher"] = g.Matcher
	}
	var hooks []any
	for _, h := range g.Hooks {
		entry := map[string]any{
			"type":    h.Type,
			"command": h.Command,
		}
		if h.Async {
			entry["async"] = true
		}
		hooks = append(hooks, entry)
	}
	m["hooks"] = hooks
	return m
}
