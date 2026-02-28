package project

import (
	"os/exec"
	"testing"
)

func TestRepoName(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		// SSH SCP-style
		{"git@github.com:boozedog/smoovtask.git", "boozedog/smoovtask"},
		{"git@github.com:boozedog/smoovtask", "boozedog/smoovtask"},
		// HTTPS
		{"https://github.com/boozedog/smoovtask.git", "boozedog/smoovtask"},
		{"https://github.com/boozedog/smoovtask", "boozedog/smoovtask"},
		// ssh:// URL
		{"ssh://git@github.com/boozedog/smoovtask.git", "boozedog/smoovtask"},
		{"ssh://git@github.com/boozedog/smoovtask", "boozedog/smoovtask"},
		// Edge cases
		{"", ""},
	}

	for _, tt := range tests {
		got := RepoName(tt.url)
		if got != tt.want {
			t.Errorf("RepoName(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestGitRemoteURL_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	got := GitRemoteURL(dir)
	if got != "" {
		t.Errorf("GitRemoteURL(non-git dir) = %q, want empty", got)
	}
}

func TestGitRemoteURL_WithRemote(t *testing.T) {
	dir := t.TempDir()

	// Set up a git repo with an origin remote.
	for _, args := range [][]string{
		{"init"},
		{"remote", "add", "origin", "https://github.com/example/repo.git"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	got := GitRemoteURL(dir)
	want := "https://github.com/example/repo.git"
	if got != want {
		t.Errorf("GitRemoteURL() = %q, want %q", got, want)
	}
}
