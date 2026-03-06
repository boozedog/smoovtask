package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/spawn"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var prepCmd = &cobra.Command{
	Use:   "prep [ticket-id]",
	Short: "Create a PR worktree with squashed changes ready for signed commits",
	Long: `Create a clean worktree off the base branch and squash-merge changes
from the ticket's work branch into it. The result is a single staged
changeset ready for the human to review and commit with GPG signing.

Does NOT push or create a PR — it stages the work and prints next steps.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPrep,
}

var (
	prepTicket string
	prepBase   string
)

func init() {
	prepCmd.Flags().StringVar(&prepTicket, "ticket", "", "ticket ID (default: current ticket)")
	prepCmd.Flags().StringVar(&prepBase, "base", "", "base branch (default: auto-detect main/master)")
	rootCmd.AddCommand(prepCmd)
}

func runPrep(_ *cobra.Command, args []string) error {
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

	// Resolve ticket
	ticketID := prepTicket
	if ticketID == "" && len(args) == 1 {
		ticketID = args[0]
	}

	var tk *ticket.Ticket
	if ticketID != "" {
		tk, err = store.Get(ticketID)
		if err != nil {
			return fmt.Errorf("get ticket: %w", err)
		}
	} else {
		tk, err = resolveCurrentTicket(store, cfg, runID, "")
		if err != nil {
			return err
		}
	}

	// Find repo root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	repoRoot, err := spawn.WorktreeRepoRoot(cwd)
	if err != nil {
		return fmt.Errorf("cannot determine repo root: %w", err)
	}

	// Determine work branch
	workBranch := spawn.BranchName(tk.ID)
	exists, err := gitBranchExists(repoRoot, workBranch)
	if err != nil {
		return fmt.Errorf("check work branch: %w", err)
	}
	if !exists {
		return fmt.Errorf("work branch %q not found — has this ticket been picked?", workBranch)
	}

	// Determine base branch
	baseBranch := prepBase
	if baseBranch == "" {
		baseBranch, err = detectBaseBranch(repoRoot)
		if err != nil {
			return fmt.Errorf("detect base branch: %w", err)
		}
	}

	// Check for existing PR worktree
	prWorktreeID := "pr-" + tk.ID
	prPath := spawn.WorktreePath(repoRoot, prWorktreeID)
	if _, err := os.Stat(prPath); err == nil {
		return fmt.Errorf("PR worktree already exists at %s — remove it first if you want to start over:\n  git worktree remove %s", prPath, prPath)
	}

	// Show what we're about to do
	diffStat, err := gitDiffStat(repoRoot, baseBranch, workBranch)
	if err != nil {
		return fmt.Errorf("diff stat: %w", err)
	}

	fmt.Printf("Ticket:      %s — %s\n", tk.ID, tk.Title)
	fmt.Printf("Work branch: %s\n", workBranch)
	fmt.Printf("Base branch: %s\n", baseBranch)
	fmt.Printf("PR worktree: %s\n", prPath)
	fmt.Println()
	fmt.Println("--- Changes ---")
	fmt.Println(diffStat)

	// Check for conflict-risk files (modified on base since diverge point)
	conflictFiles, err := detectConflictRisk(repoRoot, baseBranch, workBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not check conflict risk: %v\n", err)
	} else if len(conflictFiles) > 0 {
		fmt.Println("--- Conflict Risk ---")
		fmt.Println("These files were modified on both branches:")
		for _, f := range conflictFiles {
			fmt.Printf("  %s\n", f)
		}
		fmt.Println()
	}

	// Create PR worktree off base branch
	prBranch := "pr/" + tk.ID
	worktreesDir := spawn.WorktreePath(repoRoot, "")
	if err := os.MkdirAll(worktreesDir, 0o755); err != nil {
		return fmt.Errorf("create .worktrees dir: %w", err)
	}

	cmd := exec.Command("git", "worktree", "add", "-b", prBranch, prPath, baseBranch)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create PR worktree: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Squash-merge work branch into PR worktree
	mergeCmd := exec.Command("git", "merge", "--squash", workBranch)
	mergeCmd.Dir = prPath
	mergeOut, mergeErr := mergeCmd.CombinedOutput()
	mergeOutput := strings.TrimSpace(string(mergeOut))

	// Check for conflicts
	hasConflicts := false
	if mergeErr != nil {
		// git merge --squash exits non-zero on conflicts
		if strings.Contains(mergeOutput, "CONFLICT") {
			hasConflicts = true
		} else {
			return fmt.Errorf("merge --squash failed: %s: %w", mergeOutput, mergeErr)
		}
	}

	fmt.Println()
	if hasConflicts {
		fmt.Println("--- Merge Conflicts ---")
		fmt.Println("Conflicts detected. Resolve them in the PR worktree:")
		fmt.Printf("  cd %q\n", prPath)
		fmt.Println()

		// List conflicted files
		conflictCmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
		conflictCmd.Dir = prPath
		conflictOut, _ := conflictCmd.Output()
		if len(conflictOut) > 0 {
			fmt.Println("Conflicted files:")
			for _, f := range strings.Split(strings.TrimSpace(string(conflictOut)), "\n") {
				if f != "" {
					fmt.Printf("  %s\n", f)
				}
			}
			fmt.Println()
		}

		fmt.Println("After resolving, stage the fixes and commit:")
		fmt.Printf("  git add -A && git commit\n")
	} else {
		fmt.Println("--- Ready to Commit ---")
		fmt.Println("All changes are staged in the PR worktree. No conflicts.")
		fmt.Println()
		fmt.Println("Suggested commit message:")
		fmt.Println()
		commitMsg := suggestCommitMessage(tk)
		// Print indented
		for _, line := range strings.Split(commitMsg, "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
		fmt.Println("To commit with GPG signing:")
		fmt.Printf("  cd %q\n", prPath)
		fmt.Printf("  git commit -m %q\n", commitMsg)
		fmt.Println()
		fmt.Println("Then push and create PR:")
		fmt.Printf("  git push -u origin %s\n", prBranch)
		fmt.Printf("  gh pr create --title %q --base %s\n", tk.Title, baseBranch)
	}

	return nil
}

// detectBaseBranch determines the default branch (main or master).
func detectBaseBranch(repoRoot string) (string, error) {
	for _, candidate := range []string{"main", "master"} {
		cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+candidate)
		cmd.Dir = repoRoot
		if err := cmd.Run(); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no main or master branch found — use --base to specify")
}

// gitBranchExists checks if a local branch exists.
func gitBranchExists(repoRoot, branch string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoRoot
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check branch %s: %w", branch, err)
}

// gitDiffStat returns `git diff --stat` between two refs.
func gitDiffStat(repoRoot, base, head string) (string, error) {
	cmd := exec.Command("git", "diff", "--stat", base+"..."+head)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff --stat: %w", err)
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "(no changes)", nil
	}
	return result, nil
}

// detectConflictRisk finds files modified on both branches since they diverged.
func detectConflictRisk(repoRoot, base, work string) ([]string, error) {
	// Find merge base
	mergeBaseCmd := exec.Command("git", "merge-base", base, work)
	mergeBaseCmd.Dir = repoRoot
	mbOut, err := mergeBaseCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git merge-base: %w", err)
	}
	mergeBase := strings.TrimSpace(string(mbOut))

	// Files changed on base since diverge
	baseFiles, err := gitChangedFiles(repoRoot, mergeBase, base)
	if err != nil {
		return nil, err
	}

	// Files changed on work since diverge
	workFiles, err := gitChangedFiles(repoRoot, mergeBase, work)
	if err != nil {
		return nil, err
	}

	// Intersection
	baseSet := make(map[string]bool, len(baseFiles))
	for _, f := range baseFiles {
		baseSet[f] = true
	}

	var conflicts []string
	for _, f := range workFiles {
		if baseSet[f] {
			conflicts = append(conflicts, f)
		}
	}

	return conflicts, nil
}

// gitChangedFiles returns files changed between two refs.
func gitChangedFiles(repoRoot, from, to string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", from+".."+to)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only %s..%s: %w", from, to, err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// suggestCommitMessage builds a commit message from the ticket.
func suggestCommitMessage(tk *ticket.Ticket) string {
	// Extract first paragraph from body as description
	desc := extractDescription(tk.Body)
	if desc != "" {
		return tk.Title + "\n\n" + desc
	}
	return tk.Title
}

// extractDescription pulls the first meaningful paragraph from the ticket body.
// Skips section headers (##) and metadata lines.
func extractDescription(body string) string {
	lines := strings.Split(body, "\n")
	var descLines []string
	inContent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip headers and metadata
		if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "**actor:") || strings.HasPrefix(trimmed, "**assignee:") {
			if inContent {
				break // stop at next section
			}
			continue
		}

		if trimmed == "" {
			if inContent {
				break // end of paragraph
			}
			continue
		}

		inContent = true
		descLines = append(descLines, trimmed)
	}

	return strings.Join(descLines, "\n")
}
