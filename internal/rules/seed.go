package rules

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/*.yaml
var defaultRules embed.FS

// SeedDefaults writes embedded default rule files into dir as markdown files
// with YAML frontmatter. For new files, writes the full default. For existing
// files (either .md or .yaml), merges in any default rules whose names don't
// already exist (additive only — never removes or overwrites existing rules).
func SeedDefaults(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create rules dir: %w", err)
	}

	entries, err := defaultRules.ReadDir("defaults")
	if err != nil {
		return fmt.Errorf("read embedded defaults: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		defaultData, err := defaultRules.ReadFile("defaults/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), err)
		}

		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		mdPath := filepath.Join(dir, baseName+".md")
		yamlPath := filepath.Join(dir, entry.Name())

		// Check for existing file — prefer .md, fall back to .yaml.
		existingPath := ""
		if _, err := os.Stat(mdPath); err == nil {
			existingPath = mdPath
		} else if _, err := os.Stat(yamlPath); err == nil {
			existingPath = yamlPath
		}

		if existingPath == "" {
			// No existing file — write as markdown with frontmatter.
			md := wrapAsFrontmatter(defaultData)
			if err := os.WriteFile(mdPath, md, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", mdPath, err)
			}
			continue
		}

		// File exists — read and merge.
		existingData, err := os.ReadFile(existingPath)
		if err != nil {
			continue
		}

		// Extract YAML content from existing file (handles both .md and .yaml).
		existingYAML := existingData
		if filepath.Ext(existingPath) == ".md" {
			if fm := extractFrontmatter(existingData); fm != nil {
				existingYAML = fm
			} else {
				continue // Can't parse, leave alone.
			}
		}

		merged, err := mergeRules(existingYAML, defaultData)
		if err != nil || merged == nil {
			continue
		}

		// Write back as markdown.
		md := wrapAsFrontmatter(merged)
		if err := os.WriteFile(mdPath, md, 0o644); err != nil {
			return fmt.Errorf("write merged %s: %w", mdPath, err)
		}

		// If we upgraded from .yaml to .md, remove the old .yaml file.
		if existingPath == yamlPath && mdPath != yamlPath {
			_ = os.Remove(yamlPath)
		}
	}

	return nil
}

// wrapAsFrontmatter wraps YAML content as markdown frontmatter.
func wrapAsFrontmatter(yamlData []byte) []byte {
	return []byte("---\n" + string(yamlData) + "---\n")
}

// mergeRules adds any rules from defaultData that are missing (by name)
// in existingData. Returns nil if no new rules were added.
func mergeRules(existingData, defaultData []byte) ([]byte, error) {
	var existing Ruleset
	if err := yaml.Unmarshal(existingData, &existing); err != nil {
		return nil, fmt.Errorf("parse existing: %w", err)
	}

	var defaults Ruleset
	if err := yaml.Unmarshal(defaultData, &defaults); err != nil {
		return nil, fmt.Errorf("parse defaults: %w", err)
	}

	// Build set of existing rule names.
	have := make(map[string]bool, len(existing.Rules))
	for _, r := range existing.Rules {
		have[r.Name] = true
	}

	// Collect rules that are new.
	var added bool
	for _, r := range defaults.Rules {
		if !have[r.Name] {
			existing.Rules = append(existing.Rules, r)
			added = true
		}
	}

	if !added {
		return nil, nil
	}

	return yaml.Marshal(&existing)
}
