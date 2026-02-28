package project

import (
	"os/exec"
	"testing"

	"github.com/boozedog/smoovtask/internal/config"
)

func TestDetect(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{
			"api-server":  {Path: "/home/user/projects/api-server"},
			"smoovtask":   {Path: "/home/user/projects/smoovtask"},
			"nested-proj": {Path: "/home/user/projects/api-server/services/auth"},
		},
	}

	tests := []struct {
		dir  string
		want string
	}{
		{"/home/user/projects/api-server", "api-server"},
		{"/home/user/projects/api-server/internal", "api-server"},
		{"/home/user/projects/smoovtask", "smoovtask"},
		{"/home/user/projects/smoovtask/cmd", "smoovtask"},
		{"/home/user/projects/unknown", ""},
		{"/other/path", ""},
		// Longest prefix match: nested project wins
		{"/home/user/projects/api-server/services/auth", "nested-proj"},
		{"/home/user/projects/api-server/services/auth/handler", "nested-proj"},
	}

	for _, tt := range tests {
		got := Detect(cfg, tt.dir)
		if got != tt.want {
			t.Errorf("Detect(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestDetectEmpty(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{},
	}

	got := Detect(cfg, "/some/path")
	if got != "" {
		t.Errorf("Detect with no projects = %q, want empty", got)
	}
}

func TestDetect_GitFallback(t *testing.T) {
	// Create a temp git repo with an origin remote.
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"remote", "add", "origin", "https://github.com/example/myproject.git"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{
			"myproject": {
				Path: "/some/other/path",
				Repo: "https://github.com/example/myproject.git",
			},
		},
	}

	got := Detect(cfg, dir)
	if got != "myproject" {
		t.Errorf("Detect(git fallback) = %q, want %q", got, "myproject")
	}
}

func TestDetect_PathPreferredOverGit(t *testing.T) {
	// When path matches, git should not be needed.
	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{
			"myproject": {
				Path: "/home/user/myproject",
				Repo: "https://github.com/example/myproject.git",
			},
		},
	}

	got := Detect(cfg, "/home/user/myproject")
	if got != "myproject" {
		t.Errorf("Detect(path match) = %q, want %q", got, "myproject")
	}
}

func TestDetect_NonGitNoPathMatch(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{
			"myproject": {
				Path: "/some/other/path",
				Repo: "https://github.com/example/myproject.git",
			},
		},
	}

	got := Detect(cfg, dir)
	if got != "" {
		t.Errorf("Detect(non-git, no path match) = %q, want empty", got)
	}
}
