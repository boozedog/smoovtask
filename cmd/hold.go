package cmd

import (
	"fmt"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var holdCmd = &cobra.Command{
	Use:   "hold <ticket-id> <reason>",
	Short: "Block a ticket with a human hold",
	Args:  cobra.ExactArgs(2),
	RunE:  runHold,
}

func init() {
	rootCmd.AddCommand(holdCmd)
}

func runHold(_ *cobra.Command, args []string) error {
	id := args[0]
	reason := args[1]

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

	if tk.Status == ticket.StatusBlocked {
		return fmt.Errorf("ticket %s is already BLOCKED", tk.ID)
	}

	now := time.Now().UTC()
	oldStatus := tk.Status
	tk.PriorStatus = &oldStatus
	tk.Status = ticket.StatusBlocked
	tk.Updated = now

	actor := identity.Actor()
	runID := identity.RunID()
	ticket.AppendSection(tk, "Blocked (Hold)", actor, runID, reason, nil, now)

	if err := store.Save(tk); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	// Log event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   event.StatusBlocked,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data: map[string]any{
			"reason":       "hold",
			"message":      reason,
			"prior_status": string(oldStatus),
		},
	})

	fmt.Printf("Held %s: %s\n", tk.ID, reason)
	return nil
}
