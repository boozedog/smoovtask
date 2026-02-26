package cmd

import (
	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/project"
)

// findProjectFromCwd detects the project from the current working directory.
func findProjectFromCwd(cfg *config.Config, cwd string) string {
	return project.Detect(cfg, cwd)
}
