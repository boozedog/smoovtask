package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new ticket for the current project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNew,
}

var (
	newPriority    string
	newTags        string
	newDependsOn   string
	newDescription string
	newProject     string
	newTitle       string
)

func init() {
	newCmd.Flags().StringVarP(&newPriority, "priority", "p", "P3", "ticket priority (P0-P5)")
	newCmd.Flags().StringVarP(&newDescription, "description", "d", "", "ticket description/acceptance criteria")
	newCmd.Flags().StringVar(&newTags, "tags", "", "comma-separated tags")
	newCmd.Flags().StringVar(&newDependsOn, "depends-on", "", "comma-separated ticket IDs this ticket depends on")
	newCmd.Flags().StringVar(&newProject, "project", "", "project name (defaults to auto-detect from current directory)")
	newCmd.Flags().StringVarP(&newTitle, "title", "t", "", "ticket title (alternative to positional argument)")
	rootCmd.AddCommand(newCmd)
}

func runNew(_ *cobra.Command, args []string) error {
	var title string
	switch {
	case newTitle != "":
		title = newTitle
	case len(args) > 0:
		title = args[0]
	default:
		return fmt.Errorf("title is required — use --title or pass as argument")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var proj string
	if newProject != "" {
		if _, ok := cfg.Projects[newProject]; !ok {
			return fmt.Errorf("unknown project %q — check `st init` or config", newProject)
		}
		proj = newProject
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		proj = project.Detect(cfg, cwd)
		if proj == "" {
			return fmt.Errorf("not in a registered project — run `st init` or use --project")
		}
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

	actor := identity.Actor()
	runID := identity.RunID()
	sectionContent := title
	if newDescription != "" {
		sectionContent = newDescription
	}
	ticket.AppendSection(tk, "Created", actor, runID, sectionContent, nil, now)

	if err := store.Create(tk); err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}

	// Log event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	evData := map[string]any{"title": title, "priority": string(priority)}
	if newDescription != "" {
		evData["description"] = newDescription
	}
	_ = el.Append(event.Event{
		TS:      now,
		Event:   event.TicketCreated,
		Ticket:  tk.ID,
		Project: proj,
		Actor:   actor,
		RunID:   runID,
		Data:    evData,
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

			ticket.AppendSection(tk, "Blocked (Dependencies)", "st", "", fmt.Sprintf("Unresolved dependencies: %s", strings.Join(unresolved, ", ")), nil, now)

			if err := store.Save(tk); err != nil {
				return fmt.Errorf("save ticket after auto-block: %w", err)
			}

			_ = el.Append(event.Event{
				TS:      now,
				Event:   event.StatusBlocked,
				Ticket:  tk.ID,
				Project: proj,
				Actor:   "st",
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
