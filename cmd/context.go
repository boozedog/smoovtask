package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/identity"
	"github.com/boozedog/smoovbrain/internal/project"
	"github.com/boozedog/smoovbrain/internal/ticket"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Print current session context as JSON",
	Args:  cobra.NoArgs,
	RunE:  runContext,
}

func init() {
	rootCmd.AddCommand(contextCmd)
}

type contextOutput struct {
	SessionID    string  `json:"session_id"`
	Project      string  `json:"project"`
	ActiveTicket *string `json:"active_ticket"`
	CWD          string  `json:"cwd"`
}

func runContext(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	sessionID := identity.SessionID()
	proj := project.Detect(cfg, cwd)

	out := contextOutput{
		SessionID: sessionID,
		Project:   proj,
		CWD:       cwd,
	}

	// Find active ticket for this session
	if sessionID != "" {
		ticketsDir, err := cfg.TicketsDir()
		if err == nil {
			store := ticket.NewStore(ticketsDir)
			tickets, err := store.List(ticket.ListFilter{Project: proj})
			if err == nil {
				for _, tk := range tickets {
					if tk.Assignee == sessionID &&
						(tk.Status == ticket.StatusInProgress || tk.Status == ticket.StatusRework) {
						out.ActiveTicket = &tk.ID
						break
					}
				}
			}
		}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
