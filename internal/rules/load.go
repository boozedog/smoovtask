package rules

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp/syntax"
	"sort"

	"gopkg.in/yaml.v3"
)

// checkRegexComplexity rejects patterns that could cause catastrophic backtracking.
func checkRegexComplexity(pattern string) error {
	const maxPatternLen = 1024
	if len(pattern) > maxPatternLen {
		return fmt.Errorf("pattern too long (%d chars, max %d)", len(pattern), maxPatternLen)
	}
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return err
	}
	if hasNestedQuantifiers(re) {
		return fmt.Errorf("nested quantifiers detected (potential ReDoS)")
	}
	return nil
}

// hasNestedQuantifiers checks if a regex AST contains quantifiers nested inside other quantifiers.
func hasNestedQuantifiers(re *syntax.Regexp) bool {
	return checkNested(re, false)
}

func checkNested(re *syntax.Regexp, insideRepeat bool) bool {
	isRepeat := re.Op == syntax.OpStar || re.Op == syntax.OpPlus || re.Op == syntax.OpRepeat
	if isRepeat && insideRepeat {
		return true
	}
	nowInside := insideRepeat || isRepeat
	for _, sub := range re.Sub {
		if checkNested(sub, nowInside) {
			return true
		}
	}
	return false
}

// LoadRulesets loads all YAML rule files from a directory, sorted by priority.
func LoadRulesets(dir string) ([]*Ruleset, *BashPipeline, error) {
	if dir == "" {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read rules dir: %w", err)
	}

	var rulesets []*Ruleset
	var bashPipeline *BashPipeline

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", path, err)
		}

		var rs Ruleset
		if err := yaml.Unmarshal(data, &rs); err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", path, err)
		}

		// Validate rules
		for i, rule := range rs.Rules {
			if rule.Name == "" {
				return nil, nil, fmt.Errorf("%s: rule %d has no name", path, i)
			}
			if rule.Action != ActionAllow && rule.Action != ActionDeny && rule.Action != ActionAsk {
				return nil, nil, fmt.Errorf("%s: rule %q has invalid action %q", path, rule.Name, rule.Action)
			}

			// Validate regex patterns (includes ReDoS protection)
			for field, pattern := range map[string]string{
				"command":           rule.Match.Command,
				"file_path":         rule.Match.FilePath,
				"url":               rule.Match.URL,
				"notification_type": rule.Match.NotificationType,
			} {
				if pattern != "" {
					if err := checkRegexComplexity(pattern); err != nil {
						return nil, nil, fmt.Errorf("%s: rule %q has invalid %s regex %q: %w", path, rule.Name, field, pattern, err)
					}
				}
			}

			// Warn if a rule has completely empty MatchConfig (matches everything)
			if len(rule.Match.Tool) == 0 && rule.Match.Command == "" && rule.Match.FilePath == "" && rule.Match.URL == "" && rule.Match.NotificationType == "" {
				slog.Warn("rule has empty match config — it will match all requests", "file", path, "rule", rule.Name)
			}
		}

		if rs.Type == "bash-pipeline" {
			if bashPipeline != nil {
				return nil, nil, fmt.Errorf("%s: multiple bash-pipeline files are not allowed", path)
			}
			if len(rs.Rules) > 0 {
				slog.Warn("bash-pipeline file contains rules — they will be ignored", "file", path)
			}
			if rs.Config != nil {
				bashPipeline = NewBashPipeline(rs.Config)
			}
			continue
		}

		rulesets = append(rulesets, &rs)
	}

	// Sort by priority descending
	sort.Slice(rulesets, func(i, j int) bool {
		return rulesets[i].Priority > rulesets[j].Priority
	})

	return rulesets, bashPipeline, nil
}
