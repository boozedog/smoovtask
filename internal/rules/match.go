package rules

import (
	"regexp"
	"strings"
	"sync"
)

// regexCache caches compiled regexes to avoid recompilation.
var regexCache sync.Map

func getRegex(pattern string) (*regexp.Regexp, error) {
	if v, ok := regexCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.Store(pattern, re)
	return re, nil
}

// matchRule checks if a request matches a single rule's criteria.
// All specified fields must match (AND logic).
func matchRule(rule *Rule, toolName, command, filePath, url, notificationType string) (bool, error) {
	// Match tool name
	if len(rule.Match.Tool) > 0 {
		matched := false
		for _, t := range rule.Match.Tool {
			if strings.EqualFold(t, toolName) {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}

	// Match command (regex)
	if rule.Match.Command != "" {
		re, err := getRegex(rule.Match.Command)
		if err != nil {
			return false, err
		}
		if !re.MatchString(command) {
			return false, nil
		}
	}

	// Match file_path (regex)
	if rule.Match.FilePath != "" {
		re, err := getRegex(rule.Match.FilePath)
		if err != nil {
			return false, err
		}
		if !re.MatchString(filePath) {
			return false, nil
		}
	}

	// Match URL (regex)
	if rule.Match.URL != "" {
		re, err := getRegex(rule.Match.URL)
		if err != nil {
			return false, err
		}
		if !re.MatchString(url) {
			return false, nil
		}
	}

	// Match notification_type
	if rule.Match.NotificationType != "" {
		re, err := getRegex(rule.Match.NotificationType)
		if err != nil {
			return false, err
		}
		if !re.MatchString(notificationType) {
			return false, nil
		}
	}

	return true, nil
}
