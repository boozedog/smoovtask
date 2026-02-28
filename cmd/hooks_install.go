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
	Short: "Install smoovtask hooks into agent settings",
	Long:  `Installs smoovtask hook commands into agent settings (Claude Code, opencode, or pi). Existing hooks and settings are preserved.`,
	RunE:  runHooksInstall,
}

var agents []string

func init() {
	hooksInstallCmd.Flags().StringSliceVar(&agents, "agents", []string{"claude"}, "Agents to install hooks for: claude, opencode, pi, both")
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

// opencodePluginCode is the TypeScript code for the opencode plugin.
const opencodePluginCode = `import { spawnSync } from 'child_process';
import { writeFileSync } from 'fs';

function log(msg) {
  writeFileSync('/tmp/opencode-plugin.log', new Date().toISOString() + ': ' + msg + '\\n', { flag: 'a' });
}

function runHookSync(eventJson) {
  const result = spawnSync('st', ['hook', 'opencode-event'], {
    input: eventJson,
    env: { ...process.env, OPENCODE_HOOK: '1' },
    timeout: 5000,
  });
  if (result.stderr && result.stderr.length > 0) {
    log('stderr: ' + result.stderr.toString());
  }
  if (result.status !== 0) {
    log('Hook failed code ' + result.status);
    return null;
  }
  const trimmed = (result.stdout || '').toString().trim();
  if (!trimmed) return null;
  try {
    return JSON.parse(trimmed);
  } catch (e) {
    log('Parse error: ' + e);
    return null;
  }
}

// Cache session-start context so we don't shell out on every LLM turn.
let cachedContext = null;

export default async ({ project, client, $, directory, worktree }) => {
  return {
    "experimental.chat.system.transform": async (input, output) => {
      if (!cachedContext) {
        log('Running session-start hook for system prompt injection');
        const event = { type: 'session.created', properties: { info: { id: input.sessionID || 'unknown', directory: directory } } };
        const result = runHookSync(JSON.stringify(event));
        if (result && result.additionalContext) {
          cachedContext = result.additionalContext;
          log('Cached context: ' + cachedContext.substring(0, 200));
        }
      }
      if (cachedContext) {
        output.system.push('<smoovtask>\\n' + cachedContext + '\\n</smoovtask>');
      }
    },
    event: async ({ event }) => {
      // Only process events that st cares about (not message updates etc.)
      const handled = ['session.created', 'tool.execute.before', 'tool.execute.after',
        'session.idle', 'permission.asked', 'session.deleted'];
      if (!handled.includes(event.type)) return;

      log('Event: ' + event.type);
      const eventJson = JSON.stringify(event);
      const result = runHookSync(eventJson);
      if (!result) return;
      log('Result: ' + JSON.stringify(result).substring(0, 500));

      // Refresh cached context on session.created
      if (event.type === 'session.created' && result.additionalContext) {
        cachedContext = result.additionalContext;
        log('Updated cached context from session.created');
      }

      // Handle pre-tool additionalContext by injecting into the session
      if (result.additionalContext && event.type === 'tool.execute.before') {
        const props = event.properties || {};
        const sessionId = props.sessionID;
        if (sessionId) {
          try {
            await client.session.prompt({
              path: { id: sessionId },
              body: {
                noReply: true,
                parts: [{ type: 'text', text: result.additionalContext, synthetic: true }]
              }
            });
            log('Injected pre-tool context into session ' + sessionId);
          } catch (e) {
            log('Prompt inject error: ' + e);
          }
        }
      }
    }
  };
};`

// piExtensionCode is the TypeScript code for the pi extension bridge.
const piExtensionCode = `import { spawnSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { basename } from 'node:path';

function log(msg) {
  writeFileSync('/tmp/pi-extension.log', new Date().toISOString() + ': ' + msg + '\n', { flag: 'a' });
}

function runHookSync(payload) {
  const result = spawnSync('st', ['hook', 'pi-event'], {
    input: JSON.stringify(payload),
    env: { ...process.env, PI_HOOK: '1' },
    timeout: 5000,
  });

  if (result.stderr && result.stderr.length > 0) {
    log('stderr: ' + result.stderr.toString());
  }
  if (result.status !== 0) {
    log('Hook failed code ' + result.status);
    return null;
  }

  const trimmed = (result.stdout || '').toString().trim();
  if (!trimmed) return null;

  try {
    return JSON.parse(trimmed);
  } catch (e) {
    log('Parse error: ' + e);
    return null;
  }
}

function getSessionID(ctx) {
  const path = ctx?.sessionManager?.getSessionFile?.();
  if (typeof path === 'string' && path.length > 0) {
    const candidate = basename(path, '.jsonl');
    const parts = candidate.split('_');
    const suffix = parts[parts.length - 1];
    const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
    if (uuidPattern.test(suffix)) {
      return suffix;
    }
    return candidate;
  }
  return 'pi-session';
}

export default function (pi) {
  let cachedContext = null;

  pi.on('session_start', async (_event, ctx) => {
    const payload = {
      type: 'session_start',
      session_id: getSessionID(ctx),
      cwd: ctx.cwd,
    };
    const result = runHookSync(payload);
    if (result && result.additionalContext) {
      cachedContext = result.additionalContext;
    }
  });

  pi.on('before_agent_start', async (event, ctx) => {
    if (!cachedContext) {
      const payload = {
        type: 'session_start',
        session_id: getSessionID(ctx),
        cwd: ctx.cwd,
      };
      const result = runHookSync(payload);
      if (result && result.additionalContext) {
        cachedContext = result.additionalContext;
      }
    }

    if (!cachedContext) return;

    return {
      systemPrompt: event.systemPrompt + '\n\n<smoovtask>\n' + cachedContext + '\n</smoovtask>',
    };
  });

  pi.on('tool_call', async (event, ctx) => {
    const payload = {
      type: 'tool_call',
      session_id: getSessionID(ctx),
      cwd: ctx.cwd,
      tool_name: event.toolName,
    };
    const result = runHookSync(payload);
    if (result && result.additionalContext) {
      ctx.ui.notify(result.additionalContext, 'warning');
    }
    if (result && result.hookSpecificOutput && result.hookSpecificOutput.behavior === 'deny') {
      return { block: true, reason: result.hookSpecificOutput.reason || 'Blocked by smoovtask' };
    }
  });

  pi.on('tool_result', async (event, ctx) => {
    runHookSync({
      type: 'tool_result',
      session_id: getSessionID(ctx),
      cwd: ctx.cwd,
      tool_name: event.toolName,
    });
  });

  pi.on('agent_end', async (_event, ctx) => {
    runHookSync({
      type: 'agent_end',
      session_id: getSessionID(ctx),
      cwd: ctx.cwd,
    });
  });

  pi.on('session_shutdown', async (_event, ctx) => {
    runHookSync({
      type: 'session_shutdown',
      session_id: getSessionID(ctx),
      cwd: ctx.cwd,
    });
  });
}
`

// smoovtaskHooks returns the hook config that st needs installed.
func smoovtaskHooks() map[string][]hookGroup {
	return map[string][]hookGroup{
		"SessionStart": {
			{
				Matcher: "startup|resume|clear|compact",
				Hooks:   []hookEntry{{Type: "command", Command: "st hook session-start"}},
			},
		},
		"PreToolUse": {
			{
				Matcher: "*",
				Hooks:   []hookEntry{{Type: "command", Command: "st hook pre-tool"}},
			},
		},
		"PostToolUse": {
			{
				Matcher: "*",
				Hooks:   []hookEntry{{Type: "command", Command: "st hook post-tool", Async: true}},
			},
		},
		"SubagentStart": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook subagent-start"}},
			},
		},
		"SubagentStop": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook subagent-stop", Async: true}},
			},
		},
		"TaskCompleted": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook task-completed", Async: true}},
			},
		},
		"TeammateIdle": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook teammate-idle", Async: true}},
			},
		},
		"Stop": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook stop", Async: true}},
			},
		},
		"PermissionRequest": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook permission-request"}},
			},
		},
		"SessionEnd": {
			{
				Hooks: []hookEntry{{Type: "command", Command: "st hook session-end", Async: true}},
			},
		},
	}
}

func runHooksInstall(_ *cobra.Command, _ []string) error {
	expandedAgents := expandAgents(agents)

	for _, agent := range expandedAgents {
		switch agent {
		case "claude":
			if err := installClaudeHooks(); err != nil {
				return fmt.Errorf("install claude hooks: %w", err)
			}
		case "opencode":
			if err := installOpencodePlugin(); err != nil {
				return fmt.Errorf("install opencode plugin: %w", err)
			}
		case "pi":
			if err := installPiExtension(); err != nil {
				return fmt.Errorf("install pi extension: %w", err)
			}
		}
	}
	return nil
}

func expandAgents(agents []string) []string {
	for _, a := range agents {
		if a == "both" {
			return []string{"claude", "opencode", "pi"}
		}
	}
	return agents
}

func installClaudeHooks() error {
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

	wanted := smoovtaskHooks()
	var installed, skipped []string

	for eventName, groups := range wanted {
		if hasSmoovtaskHook(existingHooks, eventName) {
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
		fmt.Printf("Installed %d Claude hook(s):\n", len(installed))
		for _, name := range installed {
			fmt.Printf("  + %s\n", name)
		}
	}
	if len(skipped) > 0 {
		fmt.Printf("Already installed (%d Claude hook(s)):\n", len(skipped))
		for _, name := range skipped {
			fmt.Printf("  = %s\n", name)
		}
	}
	if len(installed) == 0 {
		fmt.Println("All smoovtask Claude hooks already installed.")
	}

	fmt.Printf("\nSettings: %s\n", settingsPath)
	return nil
}

// hasSmoovtaskHook checks if an st hook command already exists for the given event.
func hasSmoovtaskHook(hooks map[string]any, eventName string) bool {
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
			if len(cmd) >= 7 && cmd[:7] == "st hook" {
				return true
			}
		}
	}
	return false
}

// installOpencodePlugin installs the smoovtask plugin for opencode.
// Plugins in ~/.config/opencode/plugins/ are auto-loaded at startup,
// so we just write the .ts file — no config entry needed.
func installOpencodePlugin() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	pluginsDir := filepath.Join(home, ".config", "opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("create plugins dir: %w", err)
	}

	pluginFile := filepath.Join(pluginsDir, "smoovtask-hooks.ts")
	if err := os.WriteFile(pluginFile, []byte(opencodePluginCode), 0o644); err != nil {
		return fmt.Errorf("write plugin file: %w", err)
	}

	fmt.Printf("Installed opencode plugin: %s\n", pluginFile)
	return nil
}

// installPiExtension installs the smoovtask extension for pi.
// Extensions in ~/.pi/agent/extensions/ are auto-discovered by pi.
func installPiExtension() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	extensionsDir := filepath.Join(home, ".pi", "agent", "extensions")
	if err := os.MkdirAll(extensionsDir, 0o755); err != nil {
		return fmt.Errorf("create extensions dir: %w", err)
	}

	extensionFile := filepath.Join(extensionsDir, "smoovtask-hooks.ts")
	if err := os.WriteFile(extensionFile, []byte(piExtensionCode), 0o644); err != nil {
		return fmt.Errorf("write extension file: %w", err)
	}

	fmt.Printf("Installed pi extension: %s\n", extensionFile)
	return nil
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
