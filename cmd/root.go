package cmd

import (
	"fmt"
	"os"

	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/spf13/cobra"
)

var (
	runIDFlag string
	humanFlag bool
)

var rootCmd = &cobra.Command{
	Use:           "st",
	Short:         "smoovtask — AI agent workflow and ticketing system",
	Long:          `An opinionated workflow/ticketing system for Claude Code agents. Enforces process, captures everything in an Obsidian vault, and provides full visibility into agent work.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		identity.SetRunID(runIDFlag)
		identity.SetHuman(humanFlag)

		if isIdentityExempt(cmd) {
			return nil
		}

		if humanFlag && runIDFlag != "" {
			return fmt.Errorf("choose one identity mode: pass --run-id for agent activity or --human for manual activity")
		}

		if !humanFlag && runIDFlag == "" {
			return fmt.Errorf("run ID required for agent commands — pass --run-id (or use --human for manual commands)")
		}

		return nil
	},
}

func isIdentityExempt(cmd *cobra.Command) bool {
	if cmd.Parent() == nil {
		return true
	}

	if cmd.Name() == "hook" || cmd.Name() == "help" || cmd.Name() == "assign" || cmd.Name() == "init" || cmd.Name() == "show" || cmd.Name() == "web" {
		return true
	}

	if cmd.Name() == "install" && cmd.Parent() != nil && cmd.Parent().Name() == "hooks" {
		return true
	}

	return false
}

func init() {
	rootCmd.PersistentFlags().StringVar(&runIDFlag, "run-id", "", "run ID for this agent session")
	rootCmd.PersistentFlags().BoolVar(&humanFlag, "human", false, "mark command as human/manual activity")
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
