package skills

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed defaults/*/SKILL.md
var defaultSkills embed.FS

// Install writes embedded skill files to the global Claude Code skills
// directory (~/.claude/skills/<name>/SKILL.md). Existing skills are
// overwritten — st-managed skills are authoritative.
func Install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	skillsDir := filepath.Join(home, ".claude", "skills")

	entries, err := defaultSkills.ReadDir("defaults")
	if err != nil {
		return fmt.Errorf("read embedded skills: %w", err)
	}

	var installed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		data, err := defaultSkills.ReadFile("defaults/" + name + "/SKILL.md")
		if err != nil {
			return fmt.Errorf("read embedded %s/SKILL.md: %w", name, err)
		}

		destDir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("create skill dir %s: %w", name, err)
		}

		destFile := filepath.Join(destDir, "SKILL.md")
		if err := os.WriteFile(destFile, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", destFile, err)
		}

		installed = append(installed, name)
	}

	if len(installed) > 0 {
		fmt.Printf("Installed %d skill(s):\n", len(installed))
		for _, name := range installed {
			fmt.Printf("  + %s\n", name)
		}
	}

	return nil
}

// Uninstall removes st-managed skills from the global Claude Code skills
// directory. Only removes skills that match embedded defaults by name.
func Uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	skillsDir := filepath.Join(home, ".claude", "skills")

	entries, err := defaultSkills.ReadDir("defaults")
	if err != nil {
		return fmt.Errorf("read embedded skills: %w", err)
	}

	var removed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		destDir := filepath.Join(skillsDir, name)

		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			continue
		}

		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("remove skill %s: %w", name, err)
		}

		removed = append(removed, name)
	}

	if len(removed) > 0 {
		fmt.Printf("Removed %d skill(s):\n", len(removed))
		for _, name := range removed {
			fmt.Printf("  - %s\n", name)
		}
	} else {
		fmt.Println("No st-managed skills found to remove.")
	}

	return nil
}
