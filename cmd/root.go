package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "sb",
	Short:         "smoovbrain — AI agent workflow and ticketing system",
	Long:          `An opinionated workflow/ticketing system for Claude Code agents. Enforces process, captures everything in an Obsidian vault, and provides full visibility into agent work.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
