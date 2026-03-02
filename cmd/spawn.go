package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/spawn"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn <ticket-id>",
	Short: "Launch an AI agent worker in an isolated git worktree",
	Long: `Spawn launches a non-interactive AI agent (claude -p) in an isolated git
worktree to work on a ticket. The worker commits to its own branch and
coordinates through the ticket system.

The spawned worker runs in the background. Use 'st list' to check status.

Worktree: .worktrees/<ticket-id>
Branch:   st/<ticket-id>`,
	Args: cobra.ExactArgs(1),
	RunE: runSpawn,
}

var (
	spawnTimeout time.Duration
	spawnBackend string
	spawnDryRun  bool
)

func init() {
	spawnCmd.Flags().DurationVar(&spawnTimeout, "timeout", 45*time.Minute, "worker timeout (e.g. 45m, 1h)")
	spawnCmd.Flags().StringVar(&spawnBackend, "backend", "claude", "AI backend to use")
	spawnCmd.Flags().BoolVar(&spawnDryRun, "dry-run", false, "print the prompt without launching")
	rootCmd.AddCommand(spawnCmd)
}

func runSpawn(_ *cobra.Command, args []string) error {
	ticketID := args[0]

	// Load ticket to validate it exists and show info
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	tk, err := store.Get(ticketID)
	if err != nil {
		return fmt.Errorf("get ticket: %w", err)
	}

	if spawnDryRun {
		prompt := spawn.BuildPrompt(tk, "<run-id>")
		fmt.Println("--- Dry Run: Prompt ---")
		fmt.Println(prompt)
		fmt.Println("--- End Prompt ---")
		fmt.Printf("\nWorktree: %s\n", spawn.WorktreePath(".", tk.ID))
		fmt.Printf("Branch:   %s\n", spawn.BranchName(tk.ID))
		fmt.Printf("Backend:  %s\n", spawnBackend)
		fmt.Printf("Timeout:  %s\n", spawnTimeout)
		return nil
	}

	// Verify we're in a git repo
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	repoRoot, err := spawn.WorktreeRepoRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Check if worktree already exists
	worktreePath := spawn.WorktreePath(repoRoot, tk.ID)
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree already exists at %s — remove it first or the worker may still be running", worktreePath)
	}

	result, err := spawn.Run(spawn.Options{
		TicketID: tk.ID,
		Backend:  spawnBackend,
		Timeout:  spawnTimeout,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Spawned worker for %s: %s\n", tk.ID, tk.Title)
	fmt.Printf("  PID:      %d\n", result.PID)
	fmt.Printf("  Worktree: %s\n", result.WorktreePath)
	fmt.Printf("  Branch:   %s\n", result.Branch)
	fmt.Printf("  Run ID:   %s\n", result.RunID)
	fmt.Printf("  Log:      %s\n", result.LogPath)
	fmt.Printf("  Timeout:  %s\n", spawnTimeout)
	if result.TmuxWindow != "" {
		fmt.Printf("  Tmux:     window %q (in current session)\n", result.TmuxWindow)
	}
	fmt.Printf("\nWaiting for worker to complete...\n")

	if err := result.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Worker exited with error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Check log: %s\n", result.LogPath)
		return err
	}

	fmt.Printf("Worker completed successfully.\n")
	return nil
}
