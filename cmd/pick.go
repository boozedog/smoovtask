package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/guidance"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/spawn"
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

	projectsDir, err := cfg.ProjectsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(projectsDir)
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

	worktreePath, createdWorktree, err := ensureTicketWorktree(tk.ID)
	if err != nil {
		return err
	}

	fmt.Printf("Picked up %s: %s\n\n", tk.ID, tk.Title)
	if worktreePath == "" {
		fmt.Printf("Worktree convention: use `.worktrees/%s` (reuse existing tree when resuming).\n\n", tk.ID)
	} else {
		if createdWorktree {
			fmt.Printf("Created ticket worktree: %s\n", worktreePath)
		} else {
			fmt.Printf("Reusing ticket worktree: %s\n", worktreePath)
		}
		fmt.Printf("Switch now: cd \"%s\"\n\n", worktreePath)
	}

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
	fmt.Printf("- Do implementation work in the ticket worktree: `.worktrees/%s` (create if missing, reuse if present)\n", tk.ID)
	fmt.Printf("\nOnly begin implementation once you fully understand what is expected.\n")
	fmt.Print(guidance.LoggingImplementation())
	return nil
}

func ensureTicketWorktree(ticketID string) (string, bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("get working directory: %w", err)
	}

	repoRoot, err := spawn.WorktreeRepoRoot(cwd)
	if err != nil {
		return "", false, nil
	}

	baseRef, err := spawn.CurrentCommit(cwd)
	if err != nil {
		return "", false, fmt.Errorf("resolve current commit: %w", err)
	}

	worktreePath, _, created, err := spawn.EnsureWorktree(repoRoot, ticketID, baseRef)
	if err != nil {
		return "", false, fmt.Errorf("ensure ticket worktree: %w", err)
	}

	return worktreePath, created, nil
}
