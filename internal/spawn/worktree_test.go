package spawn

import (
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
