package rules

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/*.yaml
var defaultRules embed.FS

// SeedDefaults writes embedded default rule files into dir.
// For new files, writes the full default. For existing files,
// merges in any default rules whose names don't already exist
// (additive only — never removes or overwrites existing rules).
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

		dst := filepath.Join(dir, entry.Name())

		existingData, readErr := os.ReadFile(dst)
		if readErr != nil {
			// File doesn't exist — write the full default.
			if err := os.WriteFile(dst, defaultData, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", dst, err)
			}
			continue
		}

		// File exists — merge in missing rules.
		merged, err := mergeRules(existingData, defaultData)
		if err != nil {
			// Can't parse — leave the file alone.
			continue
		}
		if merged == nil {
			// Nothing new to add.
			continue
		}

		if err := os.WriteFile(dst, merged, 0o644); err != nil {
			return fmt.Errorf("write merged %s: %w", dst, err)
		}
	}

	return nil
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
