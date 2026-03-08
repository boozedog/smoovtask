package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectMeta holds per-project metadata stored in project.md frontmatter.
type ProjectMeta struct {
	Path string `yaml:"path,omitempty"`
	Repo string `yaml:"repo,omitempty"`
}

// LoadMeta reads project metadata from <vault>/projects/<name>/project.md.
// Returns nil (not an error) if the file does not exist.
func LoadMeta(vaultPath, projectName string) (*ProjectMeta, error) {
	p := filepath.Join(vaultPath, "projects", projectName, "project.md")
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read project.md: %w", err)
	}

	fm, _, err := splitProjectFrontmatter(string(data))
	if err != nil {
		return nil, err
	}
	if fm == "" {
		return &ProjectMeta{}, nil
	}

	var meta ProjectMeta
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("parse project.md frontmatter: %w", err)
	}

	return &meta, nil
}

// SaveMeta writes project metadata to <vault>/projects/<name>/project.md.
// Creates the directory if needed.
func SaveMeta(vaultPath, projectName string, meta *ProjectMeta) error {
	dir := filepath.Join(vaultPath, "projects", projectName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	fm, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal project metadata: %w", err)
	}

	content := "---\n" + string(fm) + "---\n"

	p := filepath.Join(dir, "project.md")
	return os.WriteFile(p, []byte(content), 0o644)
}

// ListProjects returns the names of all project subdirectories in the vault.
func ListProjects(vaultPath string) ([]string, error) {
	dir := filepath.Join(vaultPath, "projects")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read projects dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ListProjectsMeta loads metadata for all projects in the vault.
func ListProjectsMeta(vaultPath string) (map[string]*ProjectMeta, error) {
	names, err := ListProjects(vaultPath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*ProjectMeta, len(names))
	for _, name := range names {
		meta, err := LoadMeta(vaultPath, name)
		if err != nil {
			continue
		}
		if meta == nil {
			meta = &ProjectMeta{}
		}
		result[name] = meta
	}
	return result, nil
}

// splitProjectFrontmatter splits a markdown file into YAML frontmatter and body.
func splitProjectFrontmatter(content string) (frontmatter, body string, err error) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content, nil
	}

	rest := content[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", content, nil
	}

	frontmatter = rest[:end]
	body = strings.TrimPrefix(rest[end+4:], "\n")
	return frontmatter, body, nil
}
