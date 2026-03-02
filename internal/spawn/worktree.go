package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreePath returns the worktree path for a ticket: <repo>/.worktrees/<ticket-id>
func WorktreePath(repoRoot, ticketID string) string {
	return filepath.Join(repoRoot, ".worktrees", ticketID)
}

// BranchName returns the branch name for a spawned worker: st/<ticket-id>
func BranchName(ticketID string) string {
	return "st/" + ticketID
}

// CreateWorktree creates a git worktree for a ticket.
// It creates the worktree at .worktrees/<ticket-id> with branch st/<ticket-id>.
func CreateWorktree(repoRoot, ticketID string) (worktreePath, branch string, err error) {
	worktreePath = WorktreePath(repoRoot, ticketID)
	branch = BranchName(ticketID)

	// Check if worktree already exists
	if _, statErr := os.Stat(worktreePath); statErr == nil {
		return "", "", fmt.Errorf("worktree already exists at %s — remove it first or use a different ticket", worktreePath)
	}

	// Ensure .worktrees directory exists
	worktreesDir := filepath.Join(repoRoot, ".worktrees")
	if err := os.MkdirAll(worktreesDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create .worktrees dir: %w", err)
	}

	// Create worktree with new branch
	cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return worktreePath, branch, nil
}

// RepoRoot finds the git repository root from any path within it.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeRepoRoot returns the root of the main worktree (not a linked worktree).
// If we're already in a worktree, this traverses up to find the main repo.
func WorktreeRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-common-dir: %w", err)
	}
	commonDir := strings.TrimSpace(string(out))

	// The common dir is the .git directory of the main worktree.
	// Its parent is the repo root.
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(dir, commonDir)
	}
	return filepath.Dir(commonDir), nil
}
