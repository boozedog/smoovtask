package cmd

import (
	"fmt"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/workflow"
	"github.com/spf13/cobra"
)

var handoffCmd = &cobra.Command{
	Use:   "handoff [ticket-id]",
	Short: "Hand off a ticket back to the pool (clear assignee, reset to OPEN)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHandoff,
}

var handoffTicket string

func init() {
	handoffCmd.Flags().StringVar(&handoffTicket, "ticket", "", "ticket ID (default: current ticket)")
	rootCmd.AddCommand(handoffCmd)
}

func runHandoff(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	runID := identity.RunID()
	actor := identity.Actor()

	// Resolve ticket: --ticket flag > positional arg > auto-detect
	ticketID := handoffTicket
	if ticketID == "" && len(args) == 1 {
		ticketID = args[0]
	}

	var tk *ticket.Ticket
	if ticketID != "" {
		tk, err = store.Get(ticketID)
		if err != nil {
			return fmt.Errorf("get ticket: %w", err)
		}
	} else {
		tk, err = resolveCurrentTicket(store, cfg, runID, "")
		if err != nil {
			return err
		}
	}

	// Validate transition
	if err := workflow.ValidateTransition(tk.Status, ticket.StatusOpen); err != nil {
		return err
	}

	// Guard: ticket must have an assignee
	if tk.Assignee == "" {
		return fmt.Errorf("cannot hand off %s — ticket has no assignee", tk.ID)
	}

	now := time.Now().UTC()
	oldStatus := tk.Status
	previousAssignee := tk.Assignee

	tk.Status = ticket.StatusOpen
	tk.Assignee = ""
	tk.Updated = now

	ticket.AppendSection(tk, "Handed Off", actor, runID, "", map[string]string{
		"previous-assignee": previousAssignee,
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
		Event:   event.TicketHandoff,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data: map[string]any{
			"from":              string(oldStatus),
			"previous_assignee": previousAssignee,
		},
	})

	fmt.Printf("Handed off %s: %s (%s → OPEN)\n", tk.ID, tk.Title, oldStatus)

	return nil
}
