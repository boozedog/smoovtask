package rules

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Package-level compiled regexes to avoid per-invocation recompilation.
var (
	reSudo        = regexp.MustCompile(`\bsudo\b`)
	reRedirect    = regexp.MustCompile(`>{1,2}\s*(/etc/|~/.ssh/|~/.bashrc|/root/)`)
	reBacktick    = regexp.MustCompile("`")
	reDollarParen = regexp.MustCompile(`\$\(`)
	reGhAPI       = regexp.MustCompile(`\bgh\s+api\b`)
	reSplit       = regexp.MustCompile(`\s*(?:&&|\|\||;)\s*`)
)

// BashPipeline performs structural analysis on bash commands.
// It runs after declarative rules say "allow" for a Bash command,
// catching edge cases in piped/chained commands.
type BashPipeline struct {
	safeSinks         map[string]bool
	ghAPIBlockedFlags []string
}

// NewBashPipeline creates a BashPipeline from config.
func NewBashPipeline(cfg *BashPipelineConfig) *BashPipeline {
	if cfg == nil {
		return nil
	}
	bp := &BashPipeline{
		safeSinks: make(map[string]bool),
	}
	for _, s := range cfg.SafeSinks {
		bp.safeSinks[s] = true
	}
	bp.ghAPIBlockedFlags = make([]string, len(cfg.GhAPIBlockedFlags))
	copy(bp.ghAPIBlockedFlags, cfg.GhAPIBlockedFlags)
	sort.Strings(bp.ghAPIBlockedFlags)
	return bp
}

// Check performs structural analysis on a bash command.
// Returns (deny, reason) — if deny is false, the command is allowed.
// Optional rulesets enable smart $() handling — inner commands are checked
// against allowlist rules before blocking.
func (bp *BashPipeline) Check(command string, rulesets ...[]*Ruleset) (bool, string) {
	if bp == nil {
		return false, ""
	}

	// Split on &&, ||, and ; to get individual commands
	parts := splitChainedCommands(command)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for dangerous operators
		if deny, reason := checkDangerousOperators(part, rulesets...); deny {
			return true, reason
		}

		// Check pipe chains — last command must be safe sink
		if deny, reason := bp.checkPipeChain(part); deny {
			return true, reason
		}

		// Check gh api for mutations
		if deny, reason := bp.checkGhAPI(part); deny {
			return true, reason
		}
	}

	return false, ""
}

// splitChainedCommands splits on &&, ||, and ; using a simple regex.
// NOTE: This does not handle shell quoting — quoted delimiters will be split incorrectly.
func splitChainedCommands(cmd string) []string {
	return reSplit.Split(cmd, -1)
}

// splitUnquoted splits s on unquoted occurrences of sep.
// It respects single quotes, double quotes, and backslash escapes.
func splitUnquoted(s string, sep byte) []string {
	var parts []string
	var buf strings.Builder
	inSingle, inDouble := false, false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '\\' && !inSingle && i+1 < len(s) {
			buf.WriteByte(c)
			i++
			buf.WriteByte(s[i])
			continue
		}

		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			buf.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			buf.WriteByte(c)
		case c == sep && !inSingle && !inDouble:
			parts = append(parts, buf.String())
			buf.Reset()
		default:
			buf.WriteByte(c)
		}
	}
	parts = append(parts, buf.String())
	return parts
}

// checkDangerousOperators looks for redirects to sensitive paths, sudo, command substitution, etc.
// When rulesets are provided, $() command substitutions are allowed if all inner
// commands match allowlist rules. Backticks are always blocked.
func checkDangerousOperators(part string, rulesets ...[]*Ruleset) (bool, string) {
	// Check for sudo
	if reSudo.MatchString(part) {
		return true, "sudo is not allowed"
	}

	// Check for dangerous redirects to sensitive paths
	if reRedirect.MatchString(part) {
		return true, "redirect to sensitive path is not allowed"
	}

	// Always block backticks — nudge toward $() syntax.
	if reBacktick.MatchString(part) {
		return true, "backtick command substitution is not allowed (use $() instead)"
	}

	// Check for $() command substitution.
	if reDollarParen.MatchString(part) {
		// If rulesets are provided, try smart matching of inner commands.
		if len(rulesets) > 0 && rulesets[0] != nil {
			inner := extractCommandSubstitutions(part)
			if len(inner) > 0 {
				allAllowed := true
				for _, cmd := range inner {
					cmd = strings.TrimSpace(cmd)
					if cmd == "" {
						continue
					}
					result := matchCommand(rulesets[0], "pretooluse", "Bash", cmd, "", "", "")
					if result.Decision != ActionAllow {
						allAllowed = false
						break
					}
				}
				if allAllowed {
					return false, ""
				}
			}
		}
		return true, "command substitution is not allowed"
	}

	return false, ""
}

// extractCommandSubstitutions extracts the inner commands from $() groups,
// handling nesting up to a depth limit of 3.
func extractCommandSubstitutions(s string) []string {
	var results []string
	extractCmdSubsRecursive(s, 0, &results)
	return results
}

// extractCmdSubsRecursive finds $() groups and collects inner commands.
func extractCmdSubsRecursive(s string, depth int, results *[]string) {
	if depth > 3 {
		return
	}

	for i := 0; i < len(s)-1; i++ {
		if s[i] == '$' && s[i+1] == '(' {
			// Find the matching closing paren, respecting nesting.
			start := i + 2
			parenDepth := 1
			inSingle, inDouble := false, false
			j := start
			for j < len(s) && parenDepth > 0 {
				c := s[j]
				if c == '\\' && !inSingle && j+1 < len(s) {
					j += 2
					continue
				}
				switch {
				case c == '\'' && !inDouble:
					inSingle = !inSingle
				case c == '"' && !inSingle:
					inDouble = !inDouble
				case c == '(' && !inSingle && !inDouble:
					parenDepth++
				case c == ')' && !inSingle && !inDouble:
					parenDepth--
				}
				if parenDepth > 0 {
					j++
				}
			}
			if parenDepth == 0 {
				inner := s[start:j]
				*results = append(*results, inner)
				// Recurse into the inner command for nested $().
				extractCmdSubsRecursive(inner, depth+1, results)
				i = j // skip past this substitution
			}
		}
	}
}

// checkPipeChain validates that piped commands end with a safe sink.
func (bp *BashPipeline) checkPipeChain(part string) (bool, string) {
	segments := splitUnquoted(part, '|')
	if len(segments) < 2 {
		return false, ""
	}

	lastCmd := strings.TrimSpace(segments[len(segments)-1])
	firstWord := strings.Fields(lastCmd)
	if len(firstWord) == 0 {
		return false, ""
	}

	// If the last command in the pipe is a known safe sink, allow it
	if bp.safeSinks[firstWord[0]] {
		return false, ""
	}

	// Not a known safe sink — deny it
	return true, fmt.Sprintf("pipe to unknown command: %s", firstWord[0])
}

// checkGhAPI checks for mutation flags on gh api calls.
func (bp *BashPipeline) checkGhAPI(part string) (bool, string) {
	if !reGhAPI.MatchString(part) {
		return false, ""
	}

	fields := strings.Fields(part)
	for _, f := range fields {
		for _, blocked := range bp.ghAPIBlockedFlags {
			if f == blocked {
				return true, "gh api mutation flag blocked: " + f
			}
			if strings.HasPrefix(f, blocked) && len(f) > len(blocked) {
				return true, "gh api mutation flag blocked: " + blocked
			}
		}
	}

	return false, ""
}
