package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close <ticket-id>",
	Short: "Mark a ticket as done (human shortcut)",
	Args:  cobra.ExactArgs(1),
	RunE:  runClose,
}

func init() {
	rootCmd.AddCommand(closeCmd)
}

func runClose(_ *cobra.Command, args []string) error {
	id := args[0]

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

	now := time.Now().UTC()
	oldStatus := tk.Status
	tk.Status = ticket.StatusDone
	tk.PriorStatus = nil
	tk.Updated = now

	actor := identity.Actor()
	runID := identity.RunID()
	ticket.AppendSection(tk, "Closed", actor, runID, "", nil, now)

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
		Event:   event.StatusDone,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data: map[string]any{
			"from":   string(oldStatus),
			"reason": "close",
		},
	})

	fmt.Printf("Closed %s\n", tk.ID)

	// Auto-unblock dependents
	unblocked, unblockedErr := ticket.AutoUnblock(store, tk.ID)
	if unblockedErr != nil {
		fmt.Fprintf(os.Stderr, "warning: auto-unblock check failed: %v\n", unblockedErr)
	}
	for _, ut := range unblocked {
		snapStatus := "status." + strings.ToLower(string(ut.Status))
		_ = el.Append(event.Event{
			TS:      now,
			Event:   snapStatus,
			Ticket:  ut.ID,
			Project: ut.Project,
			Actor:   "st",
			Data:    map[string]any{"from": string(ticket.StatusBlocked), "reason": "auto-unblock"},
		})
		fmt.Printf("Auto-unblocked: %s → %s\n", ut.ID, ut.Status)
	}

	return nil
}
