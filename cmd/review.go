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

var reviewCmd = &cobra.Command{
	Use:   "review [ticket-id]",
	Short: "Claim a ticket for review (eligibility check enforced)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReview,
}

var reviewTicket string

func init() {
	reviewCmd.Flags().StringVar(&reviewTicket, "ticket", "", "ticket ID to review")
	rootCmd.AddCommand(reviewCmd)
}

func runReview(_ *cobra.Command, args []string) error {
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

	ticketID := reviewTicket
	if ticketID == "" && len(args) == 1 {
		ticketID = args[0]
	}

	tk, err := resolveReviewTicket(store, ticketID)
	if err != nil {
		return err
	}

	if tk.Status != ticket.StatusReview {
		return fmt.Errorf("ticket %s is %s, not REVIEW", tk.ID, tk.Status)
	}

	// Check review eligibility
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}

	if runID != "" {
		eligible, err := workflow.CanReview(eventsDir, tk.ID, runID)
		if err != nil {
			return fmt.Errorf("check review eligibility: %w", err)
		}
		if !eligible {
			return fmt.Errorf("review denied — run %q has previously touched ticket %s", runID, tk.ID)
		}
	}

	now := time.Now().UTC()
	tk.Assignee = runID
	if tk.Assignee == "" {
		tk.Assignee = actor
	}
	tk.Updated = now

	ticket.AppendSection(tk, "Review Claimed", actor, runID, "", map[string]string{
		"reviewer": tk.Assignee,
	}, now)

	if err := store.Save(tk); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	// Log event
	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   "ticket.review-claimed",
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data:    map[string]any{"reviewer": tk.Assignee},
	})

	fmt.Printf("Claimed %s for review: %s\n\n", tk.ID, tk.Title)

	// Print review context
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

	fmt.Printf("--- Review Checklist ---\n")
	fmt.Printf("- [ ] Read the ticket description and acceptance criteria\n")
	fmt.Printf("- [ ] Verify the implementation matches the requirements\n")
	fmt.Printf("- [ ] Check for edge cases and error handling\n")
	fmt.Printf("- [ ] Review code quality and test coverage\n")
	fmt.Printf("- [ ] If the fix cannot be fully verified through code review alone (e.g., UI behavior,\n")
	fmt.Printf("      runtime issues), ask the user to confirm the fix works before approving\n")
	fmt.Printf("- [ ] Document findings with `st note \"<findings>\"`\n")
	fmt.Printf("\nReminder: `st note` is required before approving (`st status done`) or rejecting (`st status rework`).\n")
	fmt.Print(guidance.LoggingReview)
	return nil
}
