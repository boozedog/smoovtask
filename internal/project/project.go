// Package project detects and resolves smoovtask project context from the working directory.
package project

import (
	"path/filepath"
	"strings"
)

// Detect determines the project name from the given directory path
// by scanning project.md files in the vault for path matches (longest prefix wins),
// then falling back to git remote URL comparison.
// Returns empty string if no match is found.
func Detect(vaultPath, dir string) string {
	if vaultPath == "" {
		return ""
	}

	dir = filepath.Clean(dir)

	meta, err := ListProjectsMeta(vaultPath)
	if err != nil {
		return ""
	}

	var bestName string
	var bestLen int

	for name, pm := range meta {
		if pm.Path == "" {
			continue
		}
		projPath := filepath.Clean(pm.Path)
		if dir == projPath || strings.HasPrefix(dir, projPath+string(filepath.Separator)) {
			if len(projPath) > bestLen {
				bestName = name
				bestLen = len(projPath)
			}
		}
	}

	if bestName != "" {
		return bestName
	}

	// Fall back to git remote URL comparison.
	remoteURL := GitRemoteURL(dir)
	if remoteURL == "" {
		return ""
	}

	for name, pm := range meta {
		if pm.Repo != "" && pm.Repo == remoteURL {
			return name
		}
	}

	return ""
}
