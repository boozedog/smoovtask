package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/event"
	"github.com/boozedog/smoovbrain/internal/project"
	"github.com/boozedog/smoovbrain/internal/ticket"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new ticket for the current project",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

var (
	newPriority  string
	newTags      string
	newDependsOn string
)

func init() {
	newCmd.Flags().StringVar(&newPriority, "priority", "P3", "ticket priority (P0-P5)")
	newCmd.Flags().StringVar(&newTags, "tags", "", "comma-separated tags")
	newCmd.Flags().StringVar(&newDependsOn, "depends-on", "", "comma-separated ticket IDs this ticket depends on")
	rootCmd.AddCommand(newCmd)
}

func runNew(_ *cobra.Command, args []string) error {
	title := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	proj := project.Detect(cfg, cwd)
	if proj == "" {
		return fmt.Errorf("not in a registered project — run `sb init` first")
	}

	priority := ticket.Priority(newPriority)
	if !ticket.ValidPriorities[priority] {
		return fmt.Errorf("invalid priority %q (use P0-P5)", newPriority)
	}

	var tags []string
	if newTags != "" {
		for _, t := range strings.Split(newTags, ",") {
			tags = append(tags, strings.TrimSpace(t))
		}
	}

	var dependsOn []string
	if newDependsOn != "" {
		for _, d := range strings.Split(newDependsOn, ",") {
			dependsOn = append(dependsOn, strings.TrimSpace(d))
		}
	}

	now := time.Now().UTC()
	tk := &ticket.Ticket{
		Title:     title,
		Project:   proj,
		Status:    ticket.StatusOpen,
		Priority:  priority,
		DependsOn: dependsOn,
		Created:   now,
		Updated:   now,
		Tags:      tags,
	}
	if tk.Tags == nil {
		tk.Tags = []string{}
	}
	if tk.DependsOn == nil {
		tk.DependsOn = []string{}
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	ticket.AppendSection(tk, "Created", "human", "", title, nil, now)

	if err := store.Create(tk); err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}

	// Log event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   event.TicketCreated,
		Ticket:  tk.ID,
		Project: proj,
		Actor:   "human",
		Data:    map[string]any{"title": title, "priority": string(priority)},
	})

	fmt.Printf("Created %s: %s\n", tk.ID, title)

	// Auto-block if any dependencies are not DONE
	if len(dependsOn) > 0 {
		unresolved, checkErr := ticket.CheckDependencies(store, tk)
		if checkErr != nil {
			fmt.Fprintf(os.Stderr, "warning: dependency check failed: %v\n", checkErr)
		} else if len(unresolved) > 0 {
			openStatus := ticket.StatusOpen
			tk.PriorStatus = &openStatus
			tk.Status = ticket.StatusBlocked
			tk.Updated = now

			ticket.AppendSection(tk, "Blocked (Dependencies)", "sb", "", fmt.Sprintf("Unresolved dependencies: %s", strings.Join(unresolved, ", ")), nil, now)

			if err := store.Save(tk); err != nil {
				return fmt.Errorf("save ticket after auto-block: %w", err)
			}

			_ = el.Append(event.Event{
				TS:      now,
				Event:   event.StatusBlocked,
				Ticket:  tk.ID,
				Project: proj,
				Actor:   "sb",
				Data: map[string]any{
					"reason":       "depends-on",
					"refs":         dependsOn,
					"prior_status": "OPEN",
				},
			})

			fmt.Printf("Auto-blocked %s: waiting on %s\n", tk.ID, strings.Join(unresolved, ", "))
		}
	}

	return nil
}
