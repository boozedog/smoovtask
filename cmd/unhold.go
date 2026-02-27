package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var unholdCmd = &cobra.Command{
	Use:   "unhold <ticket-id>",
	Short: "Release a human hold on a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runUnhold,
}

func init() {
	rootCmd.AddCommand(unholdCmd)
}

func runUnhold(_ *cobra.Command, args []string) error {
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

	if tk.Status != ticket.StatusBlocked {
		return fmt.Errorf("ticket %s is %s, not BLOCKED", tk.ID, tk.Status)
	}

	if tk.PriorStatus == nil {
		return fmt.Errorf("ticket %s has no prior status to snap back to", tk.ID)
	}

	now := time.Now().UTC()
	snapBack := *tk.PriorStatus
	tk.Status = snapBack
	tk.PriorStatus = nil
	tk.Updated = now

	actor := identity.Actor()
	runID := identity.RunID()
	ticket.AppendSection(tk, "Unhold", actor, runID, "", nil, now)

	if err := store.Save(tk); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	// Log event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	evType := "status." + strings.ToLower(string(snapBack))
	_ = el.Append(event.Event{
		TS:      now,
		Event:   evType,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data: map[string]any{
			"from":   string(ticket.StatusBlocked),
			"reason": "unhold",
		},
	})

	fmt.Printf("Released hold on %s: now %s\n", tk.ID, snapBack)
	return nil
}
