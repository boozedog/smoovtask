package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/web/templates"
	"gopkg.in/yaml.v3"
)

// Rules renders the rules page.
func (h *Handler) Rules(w http.ResponseWriter, r *http.Request) {
	data := h.buildRulesData(r)
	_ = templates.RulesPage(data).Render(r.Context(), w)
}

// PartialRules renders the rules partial for htmx swaps.
func (h *Handler) PartialRules(w http.ResponseWriter, r *http.Request) {
	data := h.buildRulesData(r)
	_ = templates.RulesPartial(data).Render(r.Context(), w)
}

// AllowRule handles POST requests to add a command to the allowlist.
func (h *Handler) AllowRule(w http.ResponseWriter, r *http.Request) {
	tool := r.FormValue("tool")
	command := r.FormValue("command")
	if tool == "" || command == "" {
		http.Error(w, "missing tool or command", http.StatusBadRequest)
		return
	}

	rulesDir, err := h.cfg.RulesDir()
	if err != nil {
		http.Error(w, "rules dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := addAllowRule(rulesDir, tool, command); err != nil {
		http.Error(w, "add rule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated rules partial.
	data := h.buildRulesData(r)
	_ = templates.RulesPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildRulesData(r *http.Request) templates.RulesData {
	filterProject := r.URL.Query().Get("project")

	// Query rule-decision events from the last 7 days.
	after := time.Now().UTC().Add(-7 * 24 * time.Hour)
	q := event.Query{
		After:   after,
		Project: filterProject,
	}
	events, _ := event.QueryEvents(h.eventsDir, q)

	// Filter to rule-decision events only.
	type groupKey struct {
		Tool    string
		Command string
	}

	type groupState struct {
		Key        groupKey
		Decision   string
		Count      int
		AllowCount int
		DenyCount  int
		AskCount   int
		LastSeen   time.Time
		Ruleset    string
		Rule       string
	}

	groups := make(map[groupKey]*groupState)

	for _, ev := range events {
		if ev.Event != event.HookRuleDecision {
			continue
		}
		if ev.Data == nil {
			continue
		}

		tool, _ := ev.Data["tool"].(string)
		decision, _ := ev.Data["decision"].(string)
		ruleset, _ := ev.Data["ruleset"].(string)
		rule, _ := ev.Data["rule"].(string)

		// Build the command key from whichever field is present.
		var cmd string
		switch {
		case ev.Data["command"] != nil:
			cmd, _ = ev.Data["command"].(string)
		case ev.Data["file_path"] != nil:
			cmd, _ = ev.Data["file_path"].(string)
		case ev.Data["pattern"] != nil:
			cmd, _ = ev.Data["pattern"].(string)
		}

		// For old events without command data, group by rule name instead
		// so they don't all collapse into one blob.
		if cmd == "" {
			cmd = rule
		}

		key := groupKey{Tool: tool, Command: cmd}
		g, ok := groups[key]
		if !ok {
			g = &groupState{Key: key, Decision: decision, Ruleset: ruleset, Rule: rule}
			groups[key] = g
		}
		g.Count++
		switch decision {
		case "allow":
			g.AllowCount++
		case "deny":
			g.DenyCount++
		default:
			g.AskCount++
		}
		if ev.TS.After(g.LastSeen) {
			g.LastSeen = ev.TS
			g.Decision = decision
			g.Ruleset = ruleset
			g.Rule = rule
		}
	}

	// Convert to template data.
	var result []templates.RuleGroup
	for _, g := range groups {
		// Determine the dominant decision for this group.
		dominantDecision := "allow"
		if g.DenyCount > 0 {
			dominantDecision = "deny"
		} else if g.AskCount > 0 {
			dominantDecision = "ask"
		}

		result = append(result, templates.RuleGroup{
			Tool:             g.Key.Tool,
			Command:          g.Key.Command,
			DominantDecision: dominantDecision,
			Count:            g.Count,
			AllowCount:       g.AllowCount,
			DenyCount:        g.DenyCount,
			AskCount:         g.AskCount,
			LastSeen:         g.LastSeen,
			Ruleset:          g.Ruleset,
			Rule:             g.Rule,
		})
	}

	// Sort: unmatched (ask) first, then denied, then allow. Within same decision, by count descending.
	decisionOrder := map[string]int{"ask": 0, "deny": 1, "allow": 2}
	sort.Slice(result, func(i, j int) bool {
		di, dj := decisionOrder[result[i].DominantDecision], decisionOrder[result[j].DominantDecision]
		if di != dj {
			return di < dj
		}
		return result[i].Count > result[j].Count
	})

	return templates.RulesData{
		Groups:         result,
		CurrentProject: filterProject,
		Projects:       h.allProjects(),
	}
}

// addAllowRule appends a new allow rule to user-allowlist.md in the rules dir.
func addAllowRule(rulesDir, tool, command string) error {
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return fmt.Errorf("create rules dir: %w", err)
	}

	path := filepath.Join(rulesDir, "user-allowlist.md")

	type matchConfig struct {
		Tool    string `yaml:"tool"`
		Command string `yaml:"command,omitempty"`
	}
	type rule struct {
		Name    string      `yaml:"name"`
		Match   matchConfig `yaml:"match"`
		Action  string      `yaml:"action"`
		Message string      `yaml:"message"`
	}
	type ruleset struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Priority    int    `yaml:"priority"`
		Event       string `yaml:"event"`
		Rules       []rule `yaml:"rules"`
	}

	rs := ruleset{
		Name:        "user-allowlist",
		Description: "User-added allowlist rules from the web UI",
		Priority:    60,
		Event:       "PreToolUse",
	}

	// Load existing file if present (markdown with YAML frontmatter).
	data, err := os.ReadFile(path)
	if err == nil {
		yamlData := extractFrontmatter(data)
		if yamlData != nil {
			_ = yaml.Unmarshal(yamlData, &rs)
		}
	}

	// Build a safe rule name from the command.
	safeName := "allow-" + tool
	if command != "" {
		// Take first word of command for the name.
		parts := strings.Fields(command)
		if len(parts) > 0 {
			safeName += "-" + sanitizeRuleName(parts[0])
		}
	}

	// Check if a rule with this exact match already exists.
	for _, r := range rs.Rules {
		if r.Match.Tool == tool && r.Match.Command == command {
			return nil // Already exists.
		}
	}

	newRule := rule{
		Name:    safeName,
		Match:   matchConfig{Tool: tool},
		Action:  "allow",
		Message: "allowed from web UI",
	}

	// Build a regex pattern from the command.
	if command != "" {
		newRule.Match.Command = buildCommandPattern(command)
	}

	rs.Rules = append(rs.Rules, newRule)

	out, err := yaml.Marshal(&rs)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// Write as markdown with YAML frontmatter.
	md := []byte("---\n" + string(out) + "---\n")
	return os.WriteFile(path, md, 0o644)
}

// sanitizeRuleName makes a string safe for use as a YAML rule name.
func sanitizeRuleName(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	return strings.ToLower(re.ReplaceAllString(s, "-"))
}

// extractFrontmatter extracts YAML frontmatter from markdown content.
func extractFrontmatter(data []byte) []byte {
	sep := []byte("---")
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, sep) {
		return nil
	}
	rest := trimmed[len(sep):]
	end := bytes.Index(rest, sep)
	if end < 0 {
		return nil
	}
	return bytes.TrimSpace(rest[:end])
}

// buildCommandPattern creates a regex pattern from a concrete command.
// Takes the first two words and anchors them.
func buildCommandPattern(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return "^" + regexp.QuoteMeta(parts[0]) + `\b`
	}
	// Use first two words for the pattern.
	return "^" + regexp.QuoteMeta(parts[0]) + `\s+` + regexp.QuoteMeta(parts[1]) + `\b`
}
