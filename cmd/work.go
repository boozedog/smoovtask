package cmd

import "github.com/spf13/cobra"

var workCLI string

var workCmd = &cobra.Command{
	Use:   "work",
	Short: "Start an implementer session in tmux",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return launchSession(roleImplementer, "", workCLI)
	},
}

func init() {
	workCmd.Flags().StringVar(&workCLI, "cli", "", "CLI backend override (claude, opencode, pi)")
	rootCmd.AddCommand(workCmd)
}
