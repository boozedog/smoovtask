package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/identity"
	"github.com/boozedog/smoovbrain/internal/project"
	"github.com/boozedog/smoovbrain/internal/ticket"
	"github.com/boozedog/smoovbrain/internal/workflow"
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:   "pick [ticket-id]",
	Short: "Pick up a ticket and start working on it",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPick,
}

func init() {
	rootCmd.AddCommand(pickCmd)
}

func runPick(_ *cobra.Command, args []string) error {
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

	var tk *ticket.Ticket
	if len(args) == 1 {
		tk, err = store.Get(args[0])
		if err != nil {
			return fmt.Errorf("get ticket: %w", err)
		}
	} else {
		// Find an open ticket for the current project
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		proj := project.Detect(cfg, cwd)
		if proj == "" {
			return fmt.Errorf("not in a registered project — run `sb init` first")
		}

		tickets, err := store.List(ticket.ListFilter{Project: proj, Status: ticket.StatusOpen})
		if err != nil {
			return fmt.Errorf("list tickets: %w", err)
		}
		if len(tickets) == 0 {
			return fmt.Errorf("no open tickets for project %q", proj)
		}
		tk = tickets[0]
	}

	// Validate transition
	if err := workflow.ValidateTransition(tk.Status, ticket.StatusInProgress); err != nil {
		return err
	}

	now := time.Now().UTC()
	tk.Status = ticket.StatusInProgress
	tk.Assignee = sessionID
	if tk.Assignee == "" {
		tk.Assignee = actor
	}
	tk.Updated = now

	ticket.AppendSection(tk, "In Progress", actor, sessionID, "", map[string]string{
		"assignee": tk.Assignee,
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
		Event:   event.StatusInProgress,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		Session: sessionID,
		Data:    map[string]any{"assignee": tk.Assignee},
	})

	fmt.Printf("Picked up %s: %s\n", tk.ID, tk.Title)
	return nil
}
