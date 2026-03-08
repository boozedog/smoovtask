package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Register current directory as a smoovtask project",
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

	vaultPath, err := cfg.VaultPath()
	if err != nil {
		return fmt.Errorf("get vault path: %w", err)
	}

	// Use git repo name if available, otherwise fall back to directory name.
	remoteURL := project.GitRemoteURL(cwd)
	name := project.RepoName(remoteURL)
	if name == "" {
		name = filepath.Base(cwd)
	}

	// Check if already registered with same path.
	existing, _ := project.LoadMeta(vaultPath, name)
	if existing != nil && existing.Path == cwd {
		fmt.Printf("Project %q already registered at %s\n", name, cwd)
		return nil
	}

	meta := &project.ProjectMeta{Path: cwd}
	if remoteURL != "" {
		meta.Repo = remoteURL
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	if err := project.SaveMeta(vaultPath, name, meta); err != nil {
		return fmt.Errorf("save project metadata: %w", err)
	}

	if remoteURL != "" {
		fmt.Printf("Registered project %q at %s (git: %s)\n", name, cwd, remoteURL)
	} else {
		fmt.Printf("Registered project %q at %s\n", name, cwd)
	}
	return nil
}
