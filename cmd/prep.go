package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/guidance"
	"github.com/boozedog/smoovtask/internal/spawn"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var prepCmd = &cobra.Command{
	Use:   "prep [ticket-id]",
	Short: "Create a PR worktree and print merge guidance",
	Long: `Analyze ticket work branches, create a clean PR worktree off the
base branch, and print step-by-step merge commands.

Creates the worktree but does NOT merge or commit — it prints commands
for the agent or human to execute inside the worktree.

Batch mode (no args): finds all mergeable tickets (REVIEW, HUMAN-REVIEW,
or DONE with an st/<id> branch that has commits beyond base), analyzes
them in dependency-aware order, creates a single PR worktree, and prints
merge commands.

Single-ticket mode (with ticket ID): analyzes one ticket's work branch,
creates a PR worktree, and prints the merge command.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPrep,
}

var (
	prepTicket string
	prepBase   string
)

func init() {
	prepCmd.Flags().StringVar(&prepTicket, "ticket", "", "ticket ID (default: batch mode)")
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

	// Find repo root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	repoRoot, err := spawn.WorktreeRepoRoot(cwd)
	if err != nil {
		return fmt.Errorf("cannot determine repo root: %w", err)
	}

	// Determine base branch
	baseBranch := prepBase
	if baseBranch == "" {
		baseBranch, err = detectBaseBranch(repoRoot)
		if err != nil {
			return fmt.Errorf("detect base branch: %w", err)
		}
	}

	// Dispatch: single-ticket or batch
	ticketID := prepTicket
	if ticketID == "" && len(args) == 1 {
		ticketID = args[0]
	}

	if ticketID != "" {
		return runPrepSingle(store, repoRoot, baseBranch, ticketID)
	}
	return runPrepBatch(cfg, store, repoRoot, baseBranch, cwd)
}

// runPrepSingle handles `st prep <ticket-id>` — single-ticket analysis.
func runPrepSingle(store *ticket.Store, repoRoot, baseBranch, ticketID string) error {
	tk, err := store.Get(ticketID)
	if err != nil {
		return fmt.Errorf("get ticket: %w", err)
	}

	workBranch := spawn.BranchName(tk.ID)
	exists, err := gitBranchExists(repoRoot, workBranch)
	if err != nil {
		return fmt.Errorf("check work branch: %w", err)
	}
	if !exists {
		return fmt.Errorf("work branch %q not found — has this ticket been picked?", workBranch)
	}

	commits, err := gitCommitCount(repoRoot, baseBranch, workBranch)
	if err != nil {
		return fmt.Errorf("count commits: %w", err)
	}
	if commits == 0 {
		fmt.Printf("Ticket %s (%s) has no commits beyond %s — nothing to merge.\n", tk.ID, tk.Title, baseBranch)
		return nil
	}

	diffStat, err := gitDiffStat(repoRoot, baseBranch, workBranch)
	if err != nil {
		return fmt.Errorf("diff stat: %w", err)
	}

	fmt.Println("=== Single Ticket ===")
	fmt.Printf("Ticket:      %s — %s\n", tk.ID, tk.Title)
	fmt.Printf("Work branch: %s (%d commit(s))\n", workBranch, commits)
	fmt.Printf("Base branch: %s\n", baseBranch)
	fmt.Println()
	fmt.Println(diffStat)

	conflictFiles, err := detectConflictRisk(repoRoot, baseBranch, workBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not check conflict risk: %v\n", err)
	} else if len(conflictFiles) > 0 {
		fmt.Println()
		fmt.Println("--- Conflict Risk ---")
		fmt.Println("These files were modified on both the base and work branch:")
		for _, f := range conflictFiles {
			fmt.Printf("  %s\n", f)
		}
	}

	// Create PR worktree
	prWorktreeID := "pr-" + tk.ID
	prPath, err := createPRWorktree(repoRoot, prWorktreeID, "pr/"+tk.ID, baseBranch)
	if err != nil {
		return err
	}

	commitMsg := suggestCommitMessage(tk)

	fmt.Println()
	fmt.Println("=== PR Worktree Created ===")
	fmt.Printf("Path: %s\n", prPath)
	fmt.Println()
	fmt.Println("⚠️  " + guidance.PRCommitRules())
	fmt.Println()
	fmt.Println("Enter the worktree and run:")
	fmt.Printf("  cd %q\n", prPath)
	fmt.Printf("  git merge --squash %s && git commit -m %q\n", workBranch, commitMsg)
	fmt.Println()
	fmt.Println("Then push and create PR:")
	fmt.Printf("  git push -u origin pr/%s\n", tk.ID)
	fmt.Printf("  gh pr create --base %s\n", baseBranch)

	return nil
}

// runPrepBatch handles `st prep` (no args) — batch analysis.
// Finds all mergeable tickets, analyzes them, and prints merge guidance.
func runPrepBatch(cfg *config.Config, store *ticket.Store, repoRoot, baseBranch, cwd string) error {
	proj := findProjectFromCwd(cfg, cwd)

	mergeable, err := mergeableTickets(store, proj, repoRoot)
	if err != nil {
		return fmt.Errorf("find mergeable tickets: %w", err)
	}
	if len(mergeable) == 0 {
		fmt.Println("No mergeable tickets found.")
		fmt.Println("Mergeable = status REVIEW, HUMAN-REVIEW, or DONE with an st/<id> branch.")
		return nil
	}

	// Sort by dependency order
	tickets := sortByDependencyOrder(mergeable)

	// Filter to tickets with actual commits and collect per-ticket info
	type ticketInfo struct {
		tk         *ticket.Ticket
		workBranch string
		commits    int
		diffStat   string
		files      []string // files changed by this ticket
	}

	var included []ticketInfo
	var skipped []string

	for _, tk := range tickets {
		workBranch := spawn.BranchName(tk.ID)
		commits, err := gitCommitCount(repoRoot, baseBranch, workBranch)
		if err != nil {
			return fmt.Errorf("count commits for %s: %w", tk.ID, err)
		}
		if commits == 0 {
			skipped = append(skipped, fmt.Sprintf("%s (%s)", tk.ID, tk.Title))
			continue
		}

		diffStat, err := gitDiffStat(repoRoot, baseBranch, workBranch)
		if err != nil {
			return fmt.Errorf("diff stat for %s: %w", tk.ID, err)
		}

		files, err := gitChangedFiles(repoRoot, baseBranch, workBranch)
		if err != nil {
			return fmt.Errorf("changed files for %s: %w", tk.ID, err)
		}

		included = append(included, ticketInfo{
			tk:         tk,
			workBranch: workBranch,
			commits:    commits,
			diffStat:   diffStat,
			files:      files,
		})
	}

	if len(included) == 0 {
		fmt.Println("No tickets have commits beyond the base branch — nothing to merge.")
		return nil
	}

	// Header
	fmt.Printf("=== PR Preparation — %d ticket(s) ===\n", len(included))
	fmt.Printf("Base branch: %s\n", baseBranch)
	if len(skipped) > 0 {
		fmt.Printf("Skipped (%d, no changes): %s\n", len(skipped), strings.Join(skipped, ", "))
	}
	fmt.Println()

	// Per-ticket analysis
	for _, info := range included {
		fmt.Printf("--- %s: %s (%d commit(s)) ---\n", info.tk.ID, info.tk.Title, info.commits)
		fmt.Println(info.diffStat)
		fmt.Println()
	}

	// Cross-ticket conflict detection
	fileOwners := make(map[string][]string) // file -> list of ticket IDs
	for _, info := range included {
		for _, f := range info.files {
			fileOwners[f] = append(fileOwners[f], info.tk.ID)
		}
	}
	var conflictFiles []string
	for f, owners := range fileOwners {
		if len(owners) > 1 {
			conflictFiles = append(conflictFiles, fmt.Sprintf("  %s — %s", f, strings.Join(owners, ", ")))
		}
	}
	if len(conflictFiles) > 0 {
		sort.Strings(conflictFiles)
		fmt.Println("--- Cross-Ticket Conflict Risk ---")
		fmt.Println("These files are modified by multiple tickets:")
		for _, line := range conflictFiles {
			fmt.Println(line)
		}
		fmt.Println()
	}

	// Create PR worktree
	ts := time.Now().UTC().Format("20060102-150405")
	prWorktreeID := "pr-" + ts
	prBranch := "pr/batch-" + ts
	prPath, err := createPRWorktree(repoRoot, prWorktreeID, prBranch, baseBranch)
	if err != nil {
		return err
	}

	fmt.Println("=== PR Worktree Created ===")
	fmt.Printf("Path: %s\n", prPath)
	fmt.Println()
	fmt.Println("⚠️  " + guidance.PRCommitRules())
	fmt.Println()
	fmt.Println("Enter the worktree and squash-merge each ticket:")
	fmt.Printf("  cd %q\n", prPath)
	fmt.Println()
	for _, info := range included {
		commitMsg := suggestCommitMessage(info.tk)
		firstLine := strings.SplitN(commitMsg, "\n", 2)[0]
		fmt.Printf("  git merge --squash %s && git commit -m %q\n", info.workBranch, firstLine)
	}
	fmt.Println()
	fmt.Println("Then push and create PR:")
	fmt.Printf("  git push -u origin %s\n", prBranch)
	fmt.Printf("  gh pr create --base %s\n", baseBranch)

	return nil
}

// mergeableTickets returns tickets that can be batch-merged:
// status is REVIEW, HUMAN-REVIEW, or DONE, and an st/<id> branch exists.
func mergeableTickets(store *ticket.Store, project, repoRoot string) ([]*ticket.Ticket, error) {
	all, err := store.ListMeta(ticket.ListFilter{Project: project})
	if err != nil {
		return nil, err
	}

	var result []*ticket.Ticket
	for _, tk := range all {
		if tk.Status != ticket.StatusReview &&
			tk.Status != ticket.StatusHumanReview &&
			tk.Status != ticket.StatusDone {
			continue
		}
		exists, err := gitBranchExists(repoRoot, spawn.BranchName(tk.ID))
		if err != nil {
			return nil, err
		}
		if exists {
			result = append(result, tk)
		}
	}
	return result, nil
}

// sortByDependencyOrder sorts tickets using topological sort (Kahn's algorithm)
// on the DependsOn graph. Tie-break: priority (P0 first), then updated (newest first).
func sortByDependencyOrder(tickets []*ticket.Ticket) []*ticket.Ticket {
	if len(tickets) <= 1 {
		return tickets
	}

	// Build ID set for tickets in this batch
	idSet := make(map[string]bool, len(tickets))
	byID := make(map[string]*ticket.Ticket, len(tickets))
	for _, tk := range tickets {
		idSet[tk.ID] = true
		byID[tk.ID] = tk
	}

	// Build in-degree map (only count deps within the batch)
	inDegree := make(map[string]int, len(tickets))
	// dependents[A] = list of tickets that depend on A
	dependents := make(map[string][]string, len(tickets))
	for _, tk := range tickets {
		if _, ok := inDegree[tk.ID]; !ok {
			inDegree[tk.ID] = 0
		}
		for _, dep := range tk.DependsOn {
			if idSet[dep] {
				inDegree[tk.ID]++
				dependents[dep] = append(dependents[dep], tk.ID)
			}
		}
	}

	// Collect nodes with no in-batch dependencies
	var queue []*ticket.Ticket
	for _, tk := range tickets {
		if inDegree[tk.ID] == 0 {
			queue = append(queue, tk)
		}
	}
	sortByPriorityThenUpdated(queue)

	var result []*ticket.Ticket
	for len(queue) > 0 {
		// Pop first
		tk := queue[0]
		queue = queue[1:]
		result = append(result, tk)

		// Reduce in-degree for dependents
		for _, depID := range dependents[tk.ID] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, byID[depID])
			}
		}
		// Re-sort queue after adding new items
		sortByPriorityThenUpdated(queue)
	}

	// If there's a cycle, append remaining tickets
	if len(result) < len(tickets) {
		for _, tk := range tickets {
			found := false
			for _, r := range result {
				if r.ID == tk.ID {
					found = true
					break
				}
			}
			if !found {
				result = append(result, tk)
			}
		}
	}

	return result
}

// sortByPriorityThenUpdated sorts tickets by priority (P0 first), then updated (newest first).
func sortByPriorityThenUpdated(tickets []*ticket.Ticket) {
	sort.SliceStable(tickets, func(i, j int) bool {
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].Updated.After(tickets[j].Updated)
	})
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

// gitCommitCount returns the number of commits on head that are not on base.
func gitCommitCount(repoRoot, base, head string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", base+".."+head)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("git rev-list --count %s..%s: %w", base, head, err)
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count); err != nil {
		return 0, fmt.Errorf("parse commit count: %w", err)
	}
	return count, nil
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
	mergeBaseCmd := exec.Command("git", "merge-base", base, work)
	mergeBaseCmd.Dir = repoRoot
	mbOut, err := mergeBaseCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git merge-base: %w", err)
	}
	mergeBase := strings.TrimSpace(string(mbOut))

	baseFiles, err := gitChangedFiles(repoRoot, mergeBase, base)
	if err != nil {
		return nil, err
	}

	workFiles, err := gitChangedFiles(repoRoot, mergeBase, work)
	if err != nil {
		return nil, err
	}

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
	cmd := exec.Command("git", "diff", "--name-only", from+"..."+to)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only %s...%s: %w", from, to, err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// createPRWorktree creates a git worktree for PR preparation off the base branch.
// Returns the worktree path. Fails if the worktree already exists.
func createPRWorktree(repoRoot, worktreeID, branchName, baseBranch string) (string, error) {
	prPath := spawn.WorktreePath(repoRoot, worktreeID)
	if _, err := os.Stat(prPath); err == nil {
		return "", fmt.Errorf("PR worktree already exists at %s — remove it first if you want to start over:\n  git worktree remove %s", prPath, prPath)
	}

	worktreesDir := spawn.WorktreePath(repoRoot, "")
	if err := os.MkdirAll(worktreesDir, 0o755); err != nil {
		return "", fmt.Errorf("create .worktrees dir: %w", err)
	}

	cmd := exec.Command("git", "worktree", "add", "-b", branchName, prPath, baseBranch)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create PR worktree: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return prPath, nil
}

// suggestCommitMessage builds a commit message from the ticket.
func suggestCommitMessage(tk *ticket.Ticket) string {
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

		if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "**actor:") || strings.HasPrefix(trimmed, "**assignee:") {
			if inContent {
				break
			}
			continue
		}

		if trimmed == "" {
			if inContent {
				break
			}
			continue
		}

		inContent = true
		descLines = append(descLines, trimmed)
	}

	return strings.Join(descLines, "\n")
}
