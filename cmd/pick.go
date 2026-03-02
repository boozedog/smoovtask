package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/guidance"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/workflow"
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:   "pick <ticket-id>",
	Short: "Pick up a ticket and start working on it",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPick,
}

var pickTicket string

func init() {
	pickCmd.Flags().StringVar(&pickTicket, "ticket", "", "ticket ID to pick up")
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
	runID := identity.RunID()
	actor := identity.Actor()

	// Resolve ticket ID: --ticket flag takes precedence, then positional arg.
	ticketID := pickTicket
	if ticketID == "" && len(args) == 1 {
		ticketID = args[0]
	}
	if ticketID == "" {
		return fmt.Errorf("ticket ID required — use `st list` then `st pick <id>` (or `--ticket <id>`)")
	}

	tk, err := store.Get(ticketID)
	if err != nil {
		return fmt.Errorf("get ticket: %w", err)
	}

	if runID != "" {
		tickets, err := store.List(ticket.ListFilter{})
		if err != nil {
			return fmt.Errorf("list tickets: %w", err)
		}

		var active []string
		for _, t := range tickets {
			if t.Assignee == runID &&
				(t.Status == ticket.StatusInProgress || t.Status == ticket.StatusRework) &&
				t.ID != tk.ID {
				active = append(active, t.ID)
			}
		}

		if len(active) > 0 {
			return fmt.Errorf("run %q already has active ticket(s): %s — hand off or submit before picking another", runID, strings.Join(active, ", "))
		}
	}

	// Validate transition
	if err := workflow.ValidateTransition(tk.Status, ticket.StatusInProgress); err != nil {
		return err
	}

	now := time.Now().UTC()
	tk.Status = ticket.StatusInProgress
	tk.Assignee = runID
	if tk.Assignee == "" {
		tk.Assignee = actor
	}
	tk.Updated = now

	ticket.AppendSection(tk, "In Progress", actor, runID, "", map[string]string{
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
		RunID:   runID,
		Data:    map[string]any{"assignee": tk.Assignee},
	})

	fmt.Printf("Picked up %s: %s\n\n", tk.ID, tk.Title)

	// Print ticket context
	fmt.Printf("--- Ticket Metadata ---\n")
	fmt.Printf("ID:       %s\n", tk.ID)
	fmt.Printf("Priority: %s\n", tk.Priority)
	fmt.Printf("Project:  %s\n", tk.Project)
	if len(tk.Tags) > 0 {
		fmt.Printf("Tags:     %s\n", strings.Join(tk.Tags, ", "))
	}
	fmt.Println()

	if tk.Body != "" {
		fmt.Printf("--- Ticket Body ---\n")
		fmt.Println(tk.Body)
	}

	fmt.Printf("--- Before You Start ---\n")
	fmt.Printf("Read the ticket description carefully. If ANYTHING is unclear or ambiguous:\n")
	fmt.Printf("- Ask the user to clarify requirements before writing any code\n")
	fmt.Printf("- Confirm acceptance criteria if they are missing or vague\n")
	fmt.Printf("- Verify scope — ask what is in and out of scope if uncertain\n")
	fmt.Printf("- Resolve ambiguity — don't guess at intent, ask\n")
	fmt.Printf("\nOnly begin implementation once you fully understand what is expected.\n")
	fmt.Print(guidance.LoggingImplementation)
	return nil
}
