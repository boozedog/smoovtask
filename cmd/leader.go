package cmd

import "github.com/spf13/cobra"

var leaderCLI string

var leaderCmd = &cobra.Command{
	Use:   "leader",
	Short: "Start a leader session in tmux",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return launchSession(roleLeader, "", leaderCLI)
	},
}

func init() {
	leaderCmd.Flags().StringVar(&leaderCLI, "cli", "", "CLI backend override (claude, opencode, pi)")
	rootCmd.AddCommand(leaderCmd)
}
