package rules

import (
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
// First deny -> immediate deny. First allow -> proceed (run bash pipeline if Bash).
// No match -> ask (passthrough).
func evaluate(rulesets []*Ruleset, bash *BashPipeline, event, toolName string, toolInput map[string]any) *EvalResult {
	command, filePath, url := extractFields(toolInput)

	// Extract notification_type from toolInput if present (for notification events)
	var notificationType string
	if toolInput != nil {
		if v, ok := toolInput["notification_type"].(string); ok {
			notificationType = v
		}
	}

	for _, rs := range rulesets {
		// Only evaluate rulesets matching the event type
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

			switch rule.Action {
			case ActionDeny:
				return &EvalResult{
					Decision: ActionDeny,
					Reason:   rule.Message,
					Ruleset:  rs.Name,
					Rule:     rule.Name,
				}
			case ActionAllow:
				// For Bash commands, additionally run structural analysis
				if toolName == "Bash" && bash != nil {
					if deny, reason := bash.Check(command); deny {
						return &EvalResult{
							Decision: ActionDeny,
							Reason:   reason,
							Ruleset:  "bash-pipeline",
							Rule:     "structural-analysis",
						}
					}
				}
				return &EvalResult{
					Decision: ActionAllow,
					Reason:   rule.Message,
					Ruleset:  rs.Name,
					Rule:     rule.Name,
				}
			case ActionAsk:
				return &EvalResult{
					Decision: ActionAsk,
					Reason:   rule.Message,
					Ruleset:  rs.Name,
					Rule:     rule.Name,
				}
			}
		}
	}

	// No match — passthrough to Claude's permission system
	return &EvalResult{
		Decision: ActionAsk,
		Reason:   "no matching rule",
	}
}
