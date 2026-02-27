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
	"github.com/boozedog/smoovtask/internal/workflow"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <status>",
	Short: "Transition current ticket to a new status",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

var statusTicket string

func init() {
	statusCmd.Flags().StringVar(&statusTicket, "ticket", "", "ticket ID (default: current ticket)")
	rootCmd.AddCommand(statusCmd)

	// Aliases
	submitCmd := &cobra.Command{
		Use:    "submit",
		Short:  "Submit current ticket for review (alias for `st status review`)",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd, []string{"review"})
		},
	}
	rootCmd.AddCommand(submitCmd)
}

func runStatus(_ *cobra.Command, args []string) error {
	targetStatus, err := workflow.StatusFromAlias(strings.ToLower(args[0]))
	if err != nil {
		return err
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

	tk, err := resolveCurrentTicket(store, cfg, runID, statusTicket)
	if err != nil {
		return err
	}

	// Validate transition
	if err := workflow.ValidateTransition(tk.Status, targetStatus); err != nil {
		return err
	}

	// Check rules
	if workflow.RequiresAssignee(targetStatus) && tk.Assignee == "" {
		return fmt.Errorf("cannot move to %s — ticket has no assignee. Run `st pick %s` first", targetStatus, tk.ID)
	}

	if workflow.RequiresNote(tk.Status, targetStatus) {
		evDir, evErr := cfg.EventsDir()
		if evErr != nil {
			return fmt.Errorf("get events dir: %w", evErr)
		}
		hasNote, noteErr := workflow.HasNoteSince(evDir, tk.ID, tk.Updated)
		if noteErr != nil {
			return fmt.Errorf("check note requirement: %w", noteErr)
		}
		if !hasNote {
			noteCmd := fmt.Sprintf("st note --ticket %s --run-id %s \"<message>\"", tk.ID, runID)
			msg := "cannot move to %s — a very detailed note is required before review. Run `%s` first"
			if tk.Status == ticket.StatusReview {
				noteCmd = fmt.Sprintf("st note --ticket %s --run-id %s \"<findings>\"", tk.ID, runID)
				msg = "cannot move to %s — a very detailed review note is required. Document your findings with `%s` first"
			}
			return fmt.Errorf(msg, targetStatus, noteCmd)
		}
	}

	now := time.Now().UTC()
	oldStatus := tk.Status
	tk.Status = targetStatus
	tk.Updated = now

	// Clear assignee when submitting for review — the reviewer will claim it via `st review`.
	// Clear assignee when moving to backlog — ticket is being deprioritized.
	if targetStatus == ticket.StatusReview || targetStatus == ticket.StatusBacklog {
		tk.Assignee = ""
	}

	heading := statusHeading(targetStatus)

	var sectionFields map[string]string
	if oldStatus == ticket.StatusReview && (targetStatus == ticket.StatusDone || targetStatus == ticket.StatusRework) {
		sectionFields = map[string]string{"reviewed-by": tk.Assignee}
	}

	ticket.AppendSection(tk, heading, actor, runID, "", sectionFields, now)

	if err := store.Save(tk); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	// Log event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	evType := "status." + strings.ToLower(string(targetStatus))
	_ = el.Append(event.Event{
		TS:      now,
		Event:   evType,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data:    map[string]any{"from": string(oldStatus)},
	})

	fmt.Printf("%s: %s → %s\n", tk.ID, oldStatus, targetStatus)

	// Auto-unblock dependents when a ticket moves to DONE
	if targetStatus == ticket.StatusDone {
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
				RunID:   runID,
				Data:    map[string]any{"from": string(ticket.StatusBlocked), "reason": "auto-unblock"},
			})
			fmt.Printf("Auto-unblocked: %s → %s\n", ut.ID, ut.Status)
		}
	}

	return nil
}

// statusHeading converts a status to a human-readable section heading.
func statusHeading(s ticket.Status) string {
	headings := map[ticket.Status]string{
		ticket.StatusBacklog:    "Backlog",
		ticket.StatusOpen:       "Open",
		ticket.StatusInProgress: "In Progress",
		ticket.StatusReview:     "Review Requested",
		ticket.StatusRework:     "Rework",
		ticket.StatusDone:       "Done",
		ticket.StatusBlocked:    "Blocked",
	}
	if h, ok := headings[s]; ok {
		return h
	}
	return string(s)
}
