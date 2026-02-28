package project

import (
	"path/filepath"
	"strings"

	"github.com/boozedog/smoovtask/internal/config"
)

// Detect determines the project name from the given directory path
// by matching against registered projects in the config.
// First tries path-based matching (longest prefix wins), then falls
// back to git remote URL comparison if no path match is found.
// Returns empty string if no match is found.
func Detect(cfg *config.Config, dir string) string {
	dir = filepath.Clean(dir)

	var bestName string
	var bestLen int

	for name, proj := range cfg.Projects {
		projPath := filepath.Clean(proj.Path)
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

	for name, proj := range cfg.Projects {
		if proj.Repo != "" && proj.Repo == remoteURL {
			return name
		}
	}

	return ""
}
