package rules

import (
	"fmt"
	"log/slog"
	"strings"
)

// normalizeEvent converts an event name to a canonical form for comparison.
// Handles PascalCase ("PreToolUse") and snake_case ("pre_tool_use") equivalently.
func normalizeEvent(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}

// extractFields pulls matching fields from the tool input map.
func extractFields(toolInput map[string]any) (command, filePath, url string) {
	if toolInput == nil {
		return command, filePath, url
	}
	if v, ok := toolInput["command"].(string); ok {
		command = v
	}
	if v, ok := toolInput["file_path"].(string); ok {
		filePath = v
	}
	if v, ok := toolInput["url"].(string); ok {
		url = v
	}
	return command, filePath, url
}

// Evaluate loads rulesets from dir and evaluates the request against them.
// Returns nil when no rules directory exists or no rules are configured
// (distinct from an "ask" result).
func Evaluate(dir, event, toolName string, toolInput map[string]any) *EvalResult {
	rulesets, bash, err := LoadRulesets(dir)
	if err != nil {
		slog.Warn("failed to load rulesets", "error", err)
		return nil
	}
	if len(rulesets) == 0 && bash == nil {
		return nil
	}

	return evaluate(rulesets, bash, event, toolName, toolInput)
}

// evaluate runs a request through all rulesets in priority order.
// For Bash commands with compound operators (&&, ||, ;), splits and evaluates
// each sub-command independently. First deny -> immediate deny. First allow ->
// proceed (run bash pipeline if Bash). No match -> ask (passthrough).
func evaluate(rulesets []*Ruleset, bash *BashPipeline, event, toolName string, toolInput map[string]any) *EvalResult {
	command, filePath, url := extractFields(toolInput)

	// Extract notification_type from toolInput if present (for notification events)
	var notificationType string
	if toolInput != nil {
		if v, ok := toolInput["notification_type"].(string); ok {
			notificationType = v
		}
	}

	// For Bash commands, split compound commands and evaluate each sub-command.
	if toolName == "Bash" && command != "" {
		parts := splitChainedCommands(command)
		if len(parts) > 1 {
			return evaluateCompound(rulesets, bash, event, parts, filePath, url, notificationType, command)
		}
	}

	result := matchCommand(rulesets, event, toolName, command, filePath, url, notificationType)
	if result.Decision == ActionAllow && toolName == "Bash" && bash != nil {
		if deny, reason := bash.Check(command, rulesets); deny {
			return &EvalResult{
				Decision: ActionDeny,
				Reason:   reason,
				Ruleset:  "bash-pipeline",
				Rule:     "structural-analysis",
			}
		}
	}
	return result
}

// evaluateCompound handles compound Bash commands (joined by &&, ||, ;).
// Each sub-command is matched independently against rules. If any sub-command
// is denied, the whole command is denied. If all are allowed, the full command
// is run through the bash pipeline for structural analysis.
func evaluateCompound(rulesets []*Ruleset, bash *BashPipeline, event string, parts []string, filePath, url, notificationType, fullCommand string) *EvalResult {
	var lastAllow *EvalResult
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result := matchCommand(rulesets, event, "Bash", part, filePath, url, notificationType)
		switch result.Decision {
		case ActionDeny:
			return result
		case ActionAllow:
			lastAllow = result
		default:
			return &EvalResult{
				Decision: ActionAsk,
				Reason:   fmt.Sprintf("sub-command not allowed: %s", part),
			}
		}
	}
	if lastAllow == nil {
		return &EvalResult{Decision: ActionAsk, Reason: "no matching rule"}
	}
	// Run pipeline on the full command for structural analysis.
	if bash != nil {
		if deny, reason := bash.Check(fullCommand, rulesets); deny {
			return &EvalResult{
				Decision: ActionDeny,
				Reason:   reason,
				Ruleset:  "bash-pipeline",
				Rule:     "structural-analysis",
			}
		}
	}
	return lastAllow
}

// matchCommand finds the first matching rule for a single command/tool invocation.
func matchCommand(rulesets []*Ruleset, event, toolName, command, filePath, url, notificationType string) *EvalResult {
	for _, rs := range rulesets {
		if rs.Event != "" && normalizeEvent(rs.Event) != normalizeEvent(event) {
			continue
		}
		for _, rule := range rs.Rules {
			matched, err := matchRule(&rule, toolName, command, filePath, url, notificationType)
			if err != nil {
				slog.Warn("match error", "ruleset", rs.Name, "rule", rule.Name, "match", rule.Match, "error", err)
				continue
			}
			if !matched {
				continue
			}
			return &EvalResult{
				Decision: rule.Action,
				Reason:   rule.Message,
				Ruleset:  rs.Name,
				Rule:     rule.Name,
			}
		}
	}
	return &EvalResult{
		Decision: ActionAsk,
		Reason:   "no matching rule",
	}
}
