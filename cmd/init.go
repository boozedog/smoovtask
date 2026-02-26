package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Register current directory as a smoovbrain project",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	name := filepath.Base(cwd)

	if existing, ok := cfg.Projects[name]; ok {
		if existing.Path == cwd {
			fmt.Printf("Project %q already registered at %s\n", name, cwd)
			return nil
		}
	}

	cfg.Projects[name] = config.ProjectConfig{Path: cwd}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Registered project %q at %s\n", name, cwd)
	return nil
}
