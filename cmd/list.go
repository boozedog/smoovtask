package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/spawn"
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

	projectsDir, err := cfg.ProjectsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	// Auto-detect project from PWD if not specified
	filterProject := listProject
	if filterProject == "" {
		cwd, err := os.Getwd()
		if err == nil {
			vaultPath, vErr := cfg.VaultPath()
			if vErr == nil {
				filterProject = project.Detect(vaultPath, cwd)
			}
		}
	}

	filter := ticket.ListFilter{
		Project: filterProject,
		Status:  ticket.Status(strings.ToUpper(listStatus)),
	}
	if listStatus == "" {
		filter.Status = ""
	}

	// Default: hide DONE and CANCELLED unless --all or --status is explicit
	if !listAll && listStatus == "" {
		filter.Excludes = []ticket.Status{ticket.StatusDone, ticket.StatusCancelled}
	}

	store := ticket.NewStore(projectsDir)
	tickets, err := store.ListMeta(filter)
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

	// Look up worker info in a single batch (best-effort)
	eventsDir, _ := cfg.EventsDir()
	var workerStates map[string]*spawn.WorkerInfo
	if eventsDir != "" {
		workerStates, _ = spawn.BatchGetWorkerInfo(eventsDir)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, tk := range tickets {
		worker := ""
		if info, ok := workerStates[tk.ID]; ok {
			worker = workerAnnotation(info)
		}
		status := string(tk.Status)
		if worker != "" {
			status += " " + worker
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			tk.ID, truncate(tk.Title, 40), status, tk.Priority, tk.Project); err != nil {
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
	case ticket.StatusHumanReview:
		return 1
	case ticket.StatusRework:
		return 2
	case ticket.StatusInProgress:
		return 3
	case ticket.StatusOpen:
		return 4
	case ticket.StatusBlocked:
		return 5
	case ticket.StatusBacklog:
		return 6
	case ticket.StatusDone:
		return 7
	case ticket.StatusCancelled:
		return 8
	default:
		return 9
	}
}

func workerAnnotation(info *spawn.WorkerInfo) string {
	switch info.State {
	case spawn.WorkerRunning:
		return fmt.Sprintf("[worker: running %s]", info.Elapsed.Truncate(time.Second))
	case spawn.WorkerCompleted:
		return "[worker: done]"
	case spawn.WorkerFailed:
		return "[worker: failed]"
	case spawn.WorkerTimeout:
		return "[worker: timeout]"
	case spawn.WorkerStale:
		return "[worker: stale]"
	default:
		return ""
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
