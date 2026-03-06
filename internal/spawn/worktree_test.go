package spawn

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorktreePath(t *testing.T) {
	got := WorktreePath("/home/user/project", "st_abc123")
	want := "/home/user/project/.worktrees/st_abc123"
	if got != want {
		t.Errorf("WorktreePath() = %q, want %q", got, want)
	}
}

func TestBranchName(t *testing.T) {
	got := BranchName("st_abc123")
	want := "st/st_abc123"
	if got != want {
		t.Errorf("BranchName() = %q, want %q", got, want)
	}
}

func TestEnsureWorktree_CreateThenReuse(t *testing.T) {
	repo := initGitRepoWithCommit(t)

	worktreePath, branch, created, err := EnsureWorktree(repo, "st_ticket", "HEAD")
	if err != nil {
		t.Fatalf("EnsureWorktree() create error: %v", err)
	}
	if !created {
		t.Fatal("EnsureWorktree() created = false, want true")
	}
	if branch != "st/st_ticket" {
		t.Fatalf("branch = %q, want %q", branch, "st/st_ticket")
	}
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree path %q should exist: %v", worktreePath, err)
	}

	worktreePath2, branch2, created2, err := EnsureWorktree(repo, "st_ticket", "HEAD")
	if err != nil {
		t.Fatalf("EnsureWorktree() reuse error: %v", err)
	}
	if created2 {
		t.Fatal("EnsureWorktree() created = true on reuse, want false")
	}
	if worktreePath2 != worktreePath {
		t.Fatalf("worktree path on reuse = %q, want %q", worktreePath2, worktreePath)
	}
	if branch2 != branch {
		t.Fatalf("branch on reuse = %q, want %q", branch2, branch)
	}
}

func TestEnsureWorktree_UsesCurrentCommitBase(t *testing.T) {
	repo := initGitRepoWithCommit(t)

	commit, err := CurrentCommit(repo)
	if err != nil {
		t.Fatalf("CurrentCommit() error: %v", err)
	}
	if commit == "" {
		t.Fatal("CurrentCommit() returned empty commit hash")
	}

	_, _, created, err := EnsureWorktree(repo, "st_from_commit", commit)
	if err != nil {
		t.Fatalf("EnsureWorktree() error: %v", err)
	}
	if !created {
		t.Fatal("EnsureWorktree() created = false, want true")
	}
}

func initGitRepoWithCommit(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "commit.gpgsign", "false")

	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "init")

	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
