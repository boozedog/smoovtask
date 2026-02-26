package cmd

import (
	"fmt"
	"os"

	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/spf13/cobra"
)

var sessionFlag string

var rootCmd = &cobra.Command{
	Use:           "st",
	Short:         "smoovtask — AI agent workflow and ticketing system",
	Long:          `An opinionated workflow/ticketing system for Claude Code agents. Enforces process, captures everything in an Obsidian vault, and provides full visibility into agent work.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if sessionFlag != "" {
			identity.SetSessionID(sessionFlag)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&sessionFlag, "session", "", "session ID (overrides CLAUDE_SESSION_ID)")
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
