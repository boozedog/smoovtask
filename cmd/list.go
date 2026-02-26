package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/project"
	"github.com/boozedog/smoovbrain/internal/ticket"
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
)

func init() {
	listCmd.Flags().StringVar(&listProject, "project", "", "filter by project name")
	listCmd.Flags().StringVar(&listStatus, "status", "", "filter by status")
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

	store := ticket.NewStore(ticketsDir)
	tickets, err := store.List(filter)
	if err != nil {
		return fmt.Errorf("list tickets: %w", err)
	}

	if len(tickets) == 0 {
		fmt.Println("No tickets found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, tk := range tickets {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			tk.ID, truncate(tk.Title, 40), tk.Status, tk.Priority, tk.Project); err != nil {
			return err
		}
	}

	return w.Flush()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
