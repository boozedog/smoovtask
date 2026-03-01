package rules

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestLoadRulesets(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules")
	rulesets, bp, err := LoadRulesets(dir)
	if err != nil {
		t.Fatalf("LoadRulesets() error = %v", err)
	}

	// Should have 4 regular rulesets (bash-pipeline is separate)
	if len(rulesets) != 4 {
		t.Fatalf("expected 4 rulesets, got %d", len(rulesets))
	}

	// Should have a bash pipeline
	if bp == nil {
		t.Fatal("expected bash pipeline to be loaded")
	}

	// Rulesets should be sorted by priority descending
	for i := 1; i < len(rulesets); i++ {
		if rulesets[i].Priority > rulesets[i-1].Priority {
			t.Errorf("rulesets not sorted by priority: %s(%d) > %s(%d)",
				rulesets[i].Name, rulesets[i].Priority,
				rulesets[i-1].Name, rulesets[i-1].Priority)
		}
	}

	// Verify highest priority is first
	if rulesets[0].Name != "security" {
		t.Errorf("expected first ruleset to be 'security' (priority 100), got %q (priority %d)",
			rulesets[0].Name, rulesets[0].Priority)
	}
}

func TestLoadRulesetsRuleContent(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules")
	rulesets, _, err := LoadRulesets(dir)
	if err != nil {
		t.Fatalf("LoadRulesets() error = %v", err)
	}

	// Find the security ruleset
	var security *Ruleset
	for _, rs := range rulesets {
		if rs.Name == "security" {
			security = rs
			break
		}
	}
	if security == nil {
		t.Fatal("security ruleset not found")
	}

	if len(security.Rules) != 2 {
		t.Fatalf("expected 2 rules in security, got %d", len(security.Rules))
	}

	if security.Rules[0].Name != "deny-rm-rf" {
		t.Errorf("expected first rule to be deny-rm-rf, got %q", security.Rules[0].Name)
	}
	if security.Rules[0].Action != ActionDeny {
		t.Errorf("expected deny action, got %s", security.Rules[0].Action)
	}
}

func TestLoadRulesetsMultiTool(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules")
	rulesets, _, err := LoadRulesets(dir)
	if err != nil {
		t.Fatalf("LoadRulesets() error = %v", err)
	}

	// Find multi-tool ruleset
	var mt *Ruleset
	for _, rs := range rulesets {
		if rs.Name == "multi-tool" {
			mt = rs
			break
		}
	}
	if mt == nil {
		t.Fatal("multi-tool ruleset not found")
	}

	if len(mt.Rules[0].Match.Tool) != 3 {
		t.Errorf("expected 3 tools in multi-tool rule, got %d", len(mt.Rules[0].Match.Tool))
	}
}

func TestLoadRulesetsBashPipelineConfig(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules")
	_, bp, err := LoadRulesets(dir)
	if err != nil {
		t.Fatalf("LoadRulesets() error = %v", err)
	}

	if bp == nil {
		t.Fatal("expected bash pipeline")
	}

	// Verify safe sinks were loaded
	if !bp.safeSinks["head"] {
		t.Error("expected 'head' in safe sinks")
	}
	if !bp.safeSinks["jq"] {
		t.Error("expected 'jq' in safe sinks")
	}

	// Verify blocked flags were loaded
	blockedSet := make(map[string]bool)
	for _, f := range bp.ghAPIBlockedFlags {
		blockedSet[f] = true
	}
	if !blockedSet["-X"] {
		t.Error("expected '-X' in blocked flags")
	}
	if !blockedSet["--method"] {
		t.Error("expected '--method' in blocked flags")
	}
}

func TestLoadRulesetsEmptyDir(t *testing.T) {
	rulesets, bp, err := LoadRulesets("")
	if err != nil {
		t.Fatalf("LoadRulesets('') error = %v", err)
	}
	if rulesets != nil {
		t.Error("expected nil rulesets for empty dir")
	}
	if bp != nil {
		t.Error("expected nil bash pipeline for empty dir")
	}
}

func TestLoadRulesetsNonexistentDir(t *testing.T) {
	rulesets, bp, err := LoadRulesets("/nonexistent/path/to/rules")
	if err != nil {
		t.Fatalf("LoadRulesets nonexistent dir should not error, got: %v", err)
	}
	if rulesets != nil {
		t.Error("expected nil rulesets for nonexistent dir")
	}
	if bp != nil {
		t.Error("expected nil bash pipeline for nonexistent dir")
	}
}

func TestLoadRulesetsValidationNoName(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules_invalid_noname")
	_, _, err := LoadRulesets(dir)
	if err == nil {
		t.Fatal("expected error for rule with no name")
	}
	if !strings.Contains(err.Error(), "has no name") {
		t.Errorf("expected 'has no name' error, got: %v", err)
	}
}

func TestLoadRulesetsValidationBadAction(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules_invalid_action")
	_, _, err := LoadRulesets(dir)
	if err == nil {
		t.Fatal("expected error for rule with invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("expected 'invalid action' error, got: %v", err)
	}
}

func TestLoadRulesetsSkipsNonYAML(t *testing.T) {
	dir := filepath.Join(testdataDir(), "rules")
	rulesets, _, err := LoadRulesets(dir)
	if err != nil {
		t.Fatalf("LoadRulesets() error = %v", err)
	}
	if len(rulesets) == 0 {
		t.Error("expected rulesets to be loaded")
	}
}

func TestCheckRegexComplexity(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
		errMsg  string
	}{
		// Valid patterns
		{name: "simple literal", pattern: "foo", wantErr: false},
		{name: "anchored wildcard", pattern: "^foo.*bar$", wantErr: false},
		{name: "character class", pattern: `^[a-z]+$`, wantErr: false},
		{name: "alternation", pattern: `git (push|pull|fetch)`, wantErr: false},
		{name: "single quantifier", pattern: `\d+\.\d+`, wantErr: false},
		{name: "optional group", pattern: `https?://`, wantErr: false},

		// Nested quantifiers (ReDoS)
		{name: "nested plus", pattern: `(a+)+`, wantErr: true, errMsg: "nested quantifiers"},
		{name: "nested star", pattern: `(a*)*`, wantErr: true, errMsg: "nested quantifiers"},
		{name: "star inside plus", pattern: `(a*)+`, wantErr: true, errMsg: "nested quantifiers"},
		{name: "nested repeat", pattern: `(a{1,10})+`, wantErr: true, errMsg: "nested quantifiers"},

		// Length cap
		{name: "pattern too long", pattern: strings.Repeat("a", 1025), wantErr: true, errMsg: "pattern too long"},
		{name: "pattern at max length", pattern: strings.Repeat("a", 1024), wantErr: false},

		// Invalid regex syntax
		{name: "invalid regex", pattern: `(unclosed`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRegexComplexity(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
