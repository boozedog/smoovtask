package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSeedDefaultsCreatesFiles(t *testing.T) {
	dir := t.TempDir()

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	// Now seeded as .md files with YAML frontmatter.
	expected := []string{
		"bash-allowlist.md",
		"bash-pipeline.md",
		"file-protection.md",
		"git-safety.md",
		"go-tools.md",
		"readonly-tools.md",
	}
	for _, name := range expected {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected %s to be non-empty", name)
		}
	}
}

func TestSeedDefaultsMergesNewRules(t *testing.T) {
	dir := t.TempDir()

	// Write an existing allowlist as .md with YAML frontmatter.
	existing := "---\nname: bash-allowlist\ndescription: Common safe commands\npriority: 50\nevent: PreToolUse\nrules:\n  - name: allow-my-custom\n    match:\n      tool: Bash\n      command: ^mycmd\\b\n    action: allow\n    message: \"my custom command\"\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "bash-allowlist.md"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	// Read the merged file.
	data, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.md"))
	if err != nil {
		t.Fatal(err)
	}

	yamlData := extractFrontmatter(data)
	if yamlData == nil {
		t.Fatal("no frontmatter in merged file")
	}

	var rs Ruleset
	if err := yaml.Unmarshal(yamlData, &rs); err != nil {
		t.Fatalf("parse merged file: %v", err)
	}

	// Custom rule should still be there.
	names := make(map[string]bool, len(rs.Rules))
	for _, r := range rs.Rules {
		names[r.Name] = true
	}

	if !names["allow-my-custom"] {
		t.Error("custom rule allow-my-custom was removed")
	}
	if !names["allow-st"] {
		t.Error("default rule allow-st was not added")
	}
	if !names["allow-git-commit"] {
		t.Error("default rule allow-git-commit was not added")
	}
}

func TestSeedDefaultsMergesFromLegacyYAML(t *testing.T) {
	dir := t.TempDir()

	// Write an existing allowlist as legacy .yaml (no frontmatter).
	existing := "name: bash-allowlist\ndescription: Custom\npriority: 50\nevent: PreToolUse\nrules:\n  - name: allow-my-custom\n    match:\n      tool: Bash\n      command: ^mycmd\\b\n    action: allow\n    message: \"my custom command\"\n"
	if err := os.WriteFile(filepath.Join(dir, "bash-allowlist.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	// Should have been converted to .md.
	mdPath := filepath.Join(dir, "bash-allowlist.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("expected .md file: %v", err)
	}

	yamlData := extractFrontmatter(data)
	if yamlData == nil {
		t.Fatal("no frontmatter in converted file")
	}

	var rs Ruleset
	if err := yaml.Unmarshal(yamlData, &rs); err != nil {
		t.Fatalf("parse: %v", err)
	}

	names := make(map[string]bool, len(rs.Rules))
	for _, r := range rs.Rules {
		names[r.Name] = true
	}

	if !names["allow-my-custom"] {
		t.Error("custom rule was lost during migration")
	}
	if !names["allow-st"] {
		t.Error("default rule allow-st was not added")
	}

	// Old .yaml should have been removed.
	if _, err := os.Stat(filepath.Join(dir, "bash-allowlist.yaml")); err == nil {
		t.Error("legacy .yaml file should have been removed after migration to .md")
	}
}

func TestSeedDefaultsDoesNotOverwriteExistingRules(t *testing.T) {
	dir := t.TempDir()

	// Write an existing allowlist that has allow-st with a custom pattern.
	existing := "---\nname: bash-allowlist\ndescription: Custom\npriority: 50\nevent: PreToolUse\nrules:\n  - name: allow-st\n    match:\n      tool: Bash\n      command: ^st\\b\n    action: allow\n    message: \"my custom st rule\"\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "bash-allowlist.md"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.md"))
	if err != nil {
		t.Fatal(err)
	}

	// The user's custom allow-st message should be preserved, not overwritten.
	if !strings.Contains(string(data), "my custom st rule") {
		t.Error("existing allow-st rule was overwritten by default")
	}
}

func TestSeedDefaultsNoOpWhenUpToDate(t *testing.T) {
	dir := t.TempDir()

	// Seed once.
	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("first SeedDefaults() error = %v", err)
	}

	// Read file content after first seed.
	first, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Seed again — should be a no-op.
	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("second SeedDefaults() error = %v", err)
	}

	second, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.md"))
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Error("second seed modified file when nothing was new")
	}
}

func TestSeedDefaultsCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "rules")

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected seeded files in new dir")
	}
}

func TestSeedDefaultsSkipsUnparseableFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a file that can't be parsed as YAML frontmatter.
	garbage := filepath.Join(dir, "bash-allowlist.md")
	if err := os.WriteFile(garbage, []byte("---\nnot: valid: yaml: [[[\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should not error — just skip the unparseable file.
	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() should not error on unparseable file: %v", err)
	}

	// Verify file was left alone.
	data, err := os.ReadFile(garbage)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "---\nnot: valid: yaml: [[[\n---\n" {
		t.Error("unparseable file was modified")
	}
}

func TestLoadRulesetsFromMarkdown(t *testing.T) {
	dir := t.TempDir()

	md := "---\nname: test-rules\npriority: 10\nevent: PreToolUse\nrules:\n  - name: allow-test\n    match:\n      tool: Bash\n      command: ^test\\b\n    action: allow\n    message: \"test allowed\"\n---\n# Test Rules\n\nDocumentation here.\n"
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	rulesets, _, err := LoadRulesets(dir)
	if err != nil {
		t.Fatalf("LoadRulesets() error = %v", err)
	}
	if len(rulesets) != 1 {
		t.Fatalf("expected 1 ruleset, got %d", len(rulesets))
	}
	if rulesets[0].Name != "test-rules" {
		t.Errorf("expected name 'test-rules', got %q", rulesets[0].Name)
	}
	if len(rulesets[0].Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rulesets[0].Rules))
	}
	if rulesets[0].Rules[0].Name != "allow-test" {
		t.Errorf("expected rule 'allow-test', got %q", rulesets[0].Rules[0].Name)
	}
}

func TestMergeRules(t *testing.T) {
	existing := []byte(`name: test
rules:
  - name: rule-a
    match:
      tool: Bash
      command: ^a\b
    action: allow
  - name: rule-b
    match:
      tool: Bash
      command: ^b\b
    action: allow
`)

	defaults := []byte(`name: test
rules:
  - name: rule-b
    match:
      tool: Bash
      command: ^b-default\b
    action: allow
    message: "should not overwrite"
  - name: rule-c
    match:
      tool: Bash
      command: ^c\b
    action: allow
    message: "new rule"
`)

	merged, err := mergeRules(existing, defaults)
	if err != nil {
		t.Fatalf("mergeRules() error = %v", err)
	}
	if merged == nil {
		t.Fatal("expected merged output, got nil")
	}

	var rs Ruleset
	if err := yaml.Unmarshal(merged, &rs); err != nil {
		t.Fatalf("parse merged: %v", err)
	}

	if len(rs.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rs.Rules))
	}

	// rule-b should keep existing pattern, not default.
	for _, r := range rs.Rules {
		if r.Name == "rule-b" && r.Match.Command == "^b-default\\b" {
			t.Error("rule-b was overwritten by default")
		}
	}

	// rule-c should be added.
	found := false
	for _, r := range rs.Rules {
		if r.Name == "rule-c" {
			found = true
		}
	}
	if !found {
		t.Error("rule-c was not added")
	}
}

func TestMergeRulesNothingNew(t *testing.T) {
	data := []byte(`name: test
rules:
  - name: rule-a
    match:
      tool: Bash
    action: allow
`)

	merged, err := mergeRules(data, data)
	if err != nil {
		t.Fatalf("mergeRules() error = %v", err)
	}
	if merged != nil {
		t.Error("expected nil when nothing new to add")
	}
}

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		nil_ bool
	}{
		{"valid", "---\nfoo: bar\n---\nbody\n", "foo: bar", false},
		{"no frontmatter", "just markdown", "", true},
		{"no closing", "---\nfoo: bar\n", "", true},
		{"empty frontmatter", "---\n---\nbody\n", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFrontmatter([]byte(tt.in))
			if tt.nil_ {
				if got != nil {
					t.Errorf("expected nil, got %q", string(got))
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil")
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}
