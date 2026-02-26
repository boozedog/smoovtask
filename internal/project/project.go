package project

import (
	"path/filepath"
	"strings"

	"github.com/boozedog/smoovbrain/internal/config"
)

// Detect determines the project name from the given directory path
// by matching against registered projects in the config.
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

	return bestName
}
