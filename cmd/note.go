package cmd

import (
	"fmt"
	"time"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/identity"
	"github.com/boozedog/smoovbrain/internal/ticket"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note <message>",
	Short: "Append a note to the current ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runNote,
}

var noteTicket string

func init() {
	noteCmd.Flags().StringVar(&noteTicket, "ticket", "", "ticket ID (default: current ticket)")
	rootCmd.AddCommand(noteCmd)
}

func runNote(_ *cobra.Command, args []string) error {
	message := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	sessionID := identity.SessionID()
	actor := identity.Actor()

	// Use the shared resolver, temporarily setting the package var
	oldTicket := statusTicket
	statusTicket = noteTicket
	tk, err := resolveCurrentTicket(store, cfg, sessionID)
	statusTicket = oldTicket
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	ticket.AppendSection(tk, "Note", actor, sessionID, message, nil, now)

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
		Event:   "ticket.note",
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		Session: sessionID,
		Data:    map[string]any{"message": message},
	})

	fmt.Printf("Note added to %s\n", tk.ID)
	return nil
}
