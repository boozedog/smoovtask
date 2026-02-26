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

var assignCmd = &cobra.Command{
	Use:   "assign <ticket-id> <agent-id>",
	Short: "Manually assign a ticket to an agent",
	Args:  cobra.ExactArgs(2),
	RunE:  runAssign,
}

func init() {
	rootCmd.AddCommand(assignCmd)
}

func runAssign(_ *cobra.Command, args []string) error {
	id := args[0]
	agentID := args[1]

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
	tk.Assignee = agentID
	tk.Updated = now

	actor := identity.Actor()
	sessionID := identity.SessionID()
	ticket.AppendSection(tk, "Assigned", actor, sessionID, "", map[string]string{
		"assignee": agentID,
	}, now)

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
		Event:   event.TicketAssigned,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		Session: sessionID,
		Data:    map[string]any{"assignee": agentID},
	})

	fmt.Printf("Assigned %s to %s\n", tk.ID, agentID)
	return nil
}
