package rules

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Package-level compiled regexes to avoid per-invocation recompilation.
var (
	reSudo     = regexp.MustCompile(`\bsudo\b`)
	reRedirect = regexp.MustCompile(`>{1,2}\s*(/etc/|~/.ssh/|~/.bashrc|/root/)`)
	reCmdSub   = regexp.MustCompile("(`|\\$\\()")
	reGhAPI    = regexp.MustCompile(`\bgh\s+api\b`)
	reSplit    = regexp.MustCompile(`\s*(?:&&|\|\||;)\s*`)
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
func (bp *BashPipeline) Check(command string) (bool, string) {
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
		if deny, reason := checkDangerousOperators(part); deny {
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
func checkDangerousOperators(part string) (bool, string) {
	// Check for sudo
	if reSudo.MatchString(part) {
		return true, "sudo is not allowed"
	}

	// Check for dangerous redirects to sensitive paths
	if reRedirect.MatchString(part) {
		return true, "redirect to sensitive path is not allowed"
	}

	// Check for command substitution
	if reCmdSub.MatchString(part) {
		return true, "command substitution is not allowed"
	}

	return false, ""
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
