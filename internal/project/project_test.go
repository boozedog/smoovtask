package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupVault creates a temp vault with project.md files for the given projects.
func setupVault(t *testing.T, projects map[string]*ProjectMeta) string {
	t.Helper()
	vaultDir := t.TempDir()
	for name, meta := range projects {
		if err := SaveMeta(vaultDir, name, meta); err != nil {
			t.Fatalf("SaveMeta(%s): %v", name, err)
		}
	}
	return vaultDir
}

func TestDetect(t *testing.T) {
	vault := setupVault(t, map[string]*ProjectMeta{
		"api-server":  {Path: "/home/user/projects/api-server"},
		"smoovtask":   {Path: "/home/user/projects/smoovtask"},
		"nested-proj": {Path: "/home/user/projects/api-server/services/auth"},
	})

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
		got := Detect(vault, tt.dir)
		if got != tt.want {
			t.Errorf("Detect(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestDetectEmpty(t *testing.T) {
	vault := setupVault(t, map[string]*ProjectMeta{})

	got := Detect(vault, "/some/path")
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

	vault := setupVault(t, map[string]*ProjectMeta{
		"myproject": {
			Path: "/some/other/path",
			Repo: "https://github.com/example/myproject.git",
		},
	})

	got := Detect(vault, dir)
	if got != "myproject" {
		t.Errorf("Detect(git fallback) = %q, want %q", got, "myproject")
	}
}

func TestDetect_PathPreferredOverGit(t *testing.T) {
	vault := setupVault(t, map[string]*ProjectMeta{
		"myproject": {
			Path: "/home/user/myproject",
			Repo: "https://github.com/example/myproject.git",
		},
	})

	got := Detect(vault, "/home/user/myproject")
	if got != "myproject" {
		t.Errorf("Detect(path match) = %q, want %q", got, "myproject")
	}
}

func TestDetect_NonGitNoPathMatch(t *testing.T) {
	dir := t.TempDir()

	vault := setupVault(t, map[string]*ProjectMeta{
		"myproject": {
			Path: "/some/other/path",
			Repo: "https://github.com/example/myproject.git",
		},
	})

	got := Detect(vault, dir)
	if got != "" {
		t.Errorf("Detect(non-git, no path match) = %q, want empty", got)
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	vaultDir := t.TempDir()

	meta := &ProjectMeta{
		Path: "/home/user/myproject",
		Repo: "https://github.com/example/myproject.git",
	}

	if err := SaveMeta(vaultDir, "myproject", meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	loaded, err := LoadMeta(vaultDir, "myproject")
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadMeta returned nil")
	}
	if loaded.Path != meta.Path {
		t.Errorf("Path = %q, want %q", loaded.Path, meta.Path)
	}
	if loaded.Repo != meta.Repo {
		t.Errorf("Repo = %q, want %q", loaded.Repo, meta.Repo)
	}
}

func TestLoadMetaNotFound(t *testing.T) {
	vaultDir := t.TempDir()

	meta, err := LoadMeta(vaultDir, "nonexistent")
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil for nonexistent project, got %+v", meta)
	}
}

func TestListProjects(t *testing.T) {
	vaultDir := t.TempDir()
	projectsDir := filepath.Join(vaultDir, "projects")
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := os.MkdirAll(filepath.Join(projectsDir, name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	names, err := ListProjects(vaultDir)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("ListProjects returned %d names, want 3", len(names))
	}
}

func TestListProjectsMeta(t *testing.T) {
	vaultDir := t.TempDir()

	if err := SaveMeta(vaultDir, "proj-a", &ProjectMeta{Path: "/a"}); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}
	if err := SaveMeta(vaultDir, "proj-b", &ProjectMeta{Path: "/b", Repo: "https://example.com/b.git"}); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	result, err := ListProjectsMeta(vaultDir)
	if err != nil {
		t.Fatalf("ListProjectsMeta: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ListProjectsMeta returned %d entries, want 2", len(result))
	}
	if result["proj-a"].Path != "/a" {
		t.Errorf("proj-a Path = %q, want /a", result["proj-a"].Path)
	}
	if result["proj-b"].Repo != "https://example.com/b.git" {
		t.Errorf("proj-b Repo = %q, want https://example.com/b.git", result["proj-b"].Repo)
	}
}
