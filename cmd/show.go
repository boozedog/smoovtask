package cmd

import (
	"fmt"
	"os"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [ticket-id]",
	Short: "Show full ticket detail",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runShow,
}

var showTicket string

func init() {
	showCmd.Flags().StringVar(&showTicket, "ticket", "", "ticket ID to show")
	rootCmd.AddCommand(showCmd)
}

func runShow(_ *cobra.Command, args []string) error {
	// --ticket flag takes precedence, then positional arg
	id := showTicket
	if id == "" && len(args) == 1 {
		id = args[0]
	}
	if id == "" {
		return fmt.Errorf("ticket ID required — pass --ticket or provide as argument")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	tk, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("get ticket: %w", err)
	}

	data, err := ticket.Render(tk)
	if err != nil {
		return fmt.Errorf("render ticket: %w", err)
	}

	_, err = os.Stdout.Write(data)
	return err
}
