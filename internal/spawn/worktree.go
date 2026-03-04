package spawn

import (
	"errors"
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

// EnsureWorktree creates or reuses a ticket worktree.
// If the worktree already exists, it is reused.
// If the branch already exists, it is checked out in the worktree.
func EnsureWorktree(repoRoot, ticketID, baseRef string) (worktreePath, branch string, created bool, err error) {
	if strings.TrimSpace(baseRef) == "" {
		baseRef = "HEAD"
	}

	worktreePath = WorktreePath(repoRoot, ticketID)
	branch = BranchName(ticketID)

	if _, statErr := os.Stat(worktreePath); statErr == nil {
		return worktreePath, branch, false, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", "", false, fmt.Errorf("check worktree path %s: %w", worktreePath, statErr)
	}

	worktreesDir := filepath.Join(repoRoot, ".worktrees")
	if mkErr := os.MkdirAll(worktreesDir, 0o755); mkErr != nil {
		return "", "", false, fmt.Errorf("create .worktrees dir: %w", mkErr)
	}

	exists, existsErr := branchExists(repoRoot, branch)
	if existsErr != nil {
		return "", "", false, existsErr
	}

	var cmd *exec.Cmd
	if exists {
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	} else {
		cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseRef)
	}
	cmd.Dir = repoRoot
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		return "", "", false, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), runErr)
	}

	return worktreePath, branch, true, nil
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

// CurrentCommit returns the current HEAD commit hash for the provided directory.
func CurrentCommit(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func branchExists(repoRoot, branch string) (bool, error) {
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

// WorktreeIsClean returns true if the worktree at the given path has no
// uncommitted changes (staged, unstaged, or untracked files).
func WorktreeIsClean(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status in %s: %w", worktreePath, err)
	}
	return strings.TrimSpace(string(out)) == "", nil
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
