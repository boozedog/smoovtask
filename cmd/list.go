package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets with optional filters",
	RunE:  runList,
}

var (
	listProject string
	listStatus  string
	listAll     bool
)

func init() {
	listCmd.Flags().StringVar(&listProject, "project", "", "filter by project name")
	listCmd.Flags().StringVar(&listStatus, "status", "", "filter by status")
	listCmd.Flags().BoolVar(&listAll, "all", false, "show all tickets including DONE")
	rootCmd.AddCommand(listCmd)
}

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	// Auto-detect project from PWD if not specified
	filterProject := listProject
	if filterProject == "" {
		cwd, err := os.Getwd()
		if err == nil {
			filterProject = project.Detect(cfg, cwd)
		}
	}

	filter := ticket.ListFilter{
		Project: filterProject,
		Status:  ticket.Status(strings.ToUpper(listStatus)),
	}
	if listStatus == "" {
		filter.Status = ""
	}

	// Default: hide DONE unless --all or --status is explicit
	if !listAll && listStatus == "" {
		filter.Excludes = []ticket.Status{ticket.StatusDone}
	}

	store := ticket.NewStore(ticketsDir)
	tickets, err := store.List(filter)
	if err != nil {
		return fmt.Errorf("list tickets: %w", err)
	}

	if len(tickets) == 0 {
		fmt.Println("No tickets found.")
		return nil
	}

	// Default sort: status weight (REVIEW first, then active, then backlog) then priority
	sort.Slice(tickets, func(i, j int) bool {
		wi, wj := listStatusWeight(tickets[i].Status), listStatusWeight(tickets[j].Status)
		if wi != wj {
			return wi < wj
		}
		// Lower priority number = higher priority (P0 > P1 > ...)
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].Updated.After(tickets[j].Updated)
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, tk := range tickets {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			tk.ID, truncate(tk.Title, 40), tk.Status, tk.Priority, tk.Project); err != nil {
			return err
		}
	}

	return w.Flush()
}

// listStatusWeight returns sort weight: lower = higher in list.
func listStatusWeight(s ticket.Status) int {
	switch s {
	case ticket.StatusReview:
		return 0
	case ticket.StatusRework:
		return 1
	case ticket.StatusInProgress:
		return 2
	case ticket.StatusOpen:
		return 3
	case ticket.StatusBlocked:
		return 4
	case ticket.StatusBacklog:
		return 5
	case ticket.StatusDone:
		return 6
	default:
		return 7
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
