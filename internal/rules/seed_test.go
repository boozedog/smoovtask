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

	expected := []string{
		"bash-allowlist.yaml",
		"bash-pipeline.yaml",
		"file-protection.yaml",
		"git-safety.yaml",
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

	// Write an existing allowlist with only one custom rule.
	existing := `name: bash-allowlist
description: Common safe commands
priority: 50
event: PreToolUse
rules:
  - name: allow-my-custom
    match:
      tool: Bash
      command: ^mycmd\b
    action: allow
    message: "my custom command"
`
	if err := os.WriteFile(filepath.Join(dir, "bash-allowlist.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	// Read the merged file.
	data, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	var rs Ruleset
	if err := yaml.Unmarshal(data, &rs); err != nil {
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
	if !names["allow-go-test"] {
		t.Error("default rule allow-go-test was not added")
	}
}

func TestSeedDefaultsDoesNotOverwriteExistingRules(t *testing.T) {
	dir := t.TempDir()

	// Write an existing allowlist that has allow-st with a custom pattern.
	existing := `name: bash-allowlist
description: Custom
priority: 50
event: PreToolUse
rules:
  - name: allow-st
    match:
      tool: Bash
      command: ^st\b
    action: allow
    message: "my custom st rule"
`
	if err := os.WriteFile(filepath.Join(dir, "bash-allowlist.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.yaml"))
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
	first, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	// Seed again — should be a no-op.
	if err := SeedDefaults(dir); err != nil {
		t.Fatalf("second SeedDefaults() error = %v", err)
	}

	second, err := os.ReadFile(filepath.Join(dir, "bash-allowlist.yaml"))
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

	// Write a file that can't be parsed as YAML ruleset.
	garbage := filepath.Join(dir, "bash-allowlist.yaml")
	if err := os.WriteFile(garbage, []byte("not: valid: yaml: [[["), 0o644); err != nil {
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
	if string(data) != "not: valid: yaml: [[[" {
		t.Error("unparseable file was modified")
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
