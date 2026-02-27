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

var noteCmd = &cobra.Command{
	Use:   "note [ticket-id] <message>",
	Short: "Append a note to the current ticket",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runNote,
}

var noteTicket string

func init() {
	noteCmd.Flags().StringVar(&noteTicket, "ticket", "", "ticket ID (default: current ticket)")
	rootCmd.AddCommand(noteCmd)
}

func runNote(_ *cobra.Command, args []string) error {
	// Support both: st note <message> and st note <ticket-id> <message>
	var message string
	ticketFlag := noteTicket
	if len(args) == 2 {
		if ticketFlag == "" {
			ticketFlag = args[0]
		}
		message = args[1]
	} else {
		// Single arg — if --ticket is not set and arg looks like a ticket ID, error helpfully
		if ticketFlag == "" && ticket.LooksLikeTicketID(args[0]) {
			return fmt.Errorf("argument %q looks like a ticket ID — did you mean: st note %s \"<message>\"?", args[0], args[0])
		}
		message = args[0]
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
	runID := identity.RunID()
	actor := identity.Actor()

	tk, err := resolveCurrentTicket(store, cfg, runID, ticketFlag)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	ticket.AppendSection(tk, "Note", actor, runID, message, nil, now)

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
		Event:   event.TicketNote,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data:    map[string]any{"message": message},
	})

	fmt.Printf("Note added to %s\n", tk.ID)
	return nil
}
