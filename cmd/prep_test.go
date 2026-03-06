package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestPrep_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}

	// The test env already ran git init + commit, so we have a repo with a commit.
	// Create a "master" branch (newTestEnv inits on whatever default branch).
	// Rename current branch to master so detectBaseBranch finds it.
	runGitCmd(t, wd, "branch", "-M", "master")

	// Create ticket and a work branch with some changes
	tk := env.createTicket(t, "test prep feature", ticket.StatusInProgress)
	tk.Assignee = "test-session-prep"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Create work branch from master
	workBranch := "st/" + tk.ID
	runGitCmd(t, wd, "checkout", "-b", workBranch)

	// Add a file on the work branch
	testFile := filepath.Join(wd, "feature.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write feature.go: %v", err)
	}
	runGitCmd(t, wd, "add", "feature.go")
	runGitCmd(t, wd, "commit", "-m", "add feature")

	// Switch back to master
	runGitCmd(t, wd, "checkout", "master")

	out, err := env.runCmd(t, "--human", "prep", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Should show ticket info
	if !strings.Contains(out, tk.ID) {
		t.Errorf("output missing ticket ID %s: %s", tk.ID, out)
	}

	// Should show work branch
	if !strings.Contains(out, workBranch) {
		t.Errorf("output missing work branch %s: %s", workBranch, out)
	}

	// Should report ready to commit (no conflicts expected)
	if !strings.Contains(out, "Ready to Commit") {
		t.Errorf("output missing 'Ready to Commit': %s", out)
	}

	// Should show suggested commit message
	if !strings.Contains(out, "test prep feature") {
		t.Errorf("output missing suggested commit message: %s", out)
	}

	// PR worktree should exist
	prPath := filepath.Join(wd, ".worktrees", "pr-"+tk.ID)
	if _, err := os.Stat(prPath); err != nil {
		t.Errorf("PR worktree should exist at %s: %v", prPath, err)
	}

	// feature.go should be staged in the PR worktree
	if _, err := os.Stat(filepath.Join(prPath, "feature.go")); err != nil {
		t.Errorf("feature.go should exist in PR worktree: %v", err)
	}
}

func TestPrep_NoWorkBranch(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tk := env.createTicket(t, "no branch yet", ticket.StatusInProgress)
	tk.Assignee = "test-session"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	_, err = env.runCmd(t, "--human", "prep", tk.ID)
	if err == nil {
		t.Fatal("expected error when work branch doesn't exist")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want substring 'not found'", err.Error())
	}
}

func TestPrep_ExistingPRWorktree(t *testing.T) {
	env := newTestEnv(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	runGitCmd(t, wd, "branch", "-M", "master")

	tk := env.createTicket(t, "existing PR wt", ticket.StatusInProgress)
	tk.Assignee = "test-session"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Create work branch
	workBranch := "st/" + tk.ID
	runGitCmd(t, wd, "checkout", "-b", workBranch)
	testFile := filepath.Join(wd, "change.txt")
	if err := os.WriteFile(testFile, []byte("change\n"), 0o644); err != nil {
		t.Fatalf("write change.txt: %v", err)
	}
	runGitCmd(t, wd, "add", "change.txt")
	runGitCmd(t, wd, "commit", "-m", "work")
	runGitCmd(t, wd, "checkout", "master")

	// Pre-create the PR worktree directory
	prPath := filepath.Join(wd, ".worktrees", "pr-"+tk.ID)
	if err := os.MkdirAll(prPath, 0o755); err != nil {
		t.Fatalf("mkdir PR worktree: %v", err)
	}

	_, err = env.runCmd(t, "--human", "prep", tk.ID)
	if err == nil {
		t.Fatal("expected error when PR worktree already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want substring 'already exists'", err.Error())
	}
}
