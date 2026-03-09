package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall(t *testing.T) {
	// Override HOME so we write to a temp dir.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := Install(); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	// Verify skill files were written.
	for _, name := range []string{"team-lead", "review-team-lead"} {
		path := filepath.Join(tmp, ".claude", "skills", name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", path, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("expected %s to have content", path)
		}
	}
}

func TestInstallOverwrites(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Write an old version of team-lead.
	dir := filepath.Join(tmp, ".claude", "skills", "team-lead")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("old content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "old content" {
		t.Error("expected Install to overwrite old content")
	}
}

func TestUninstall(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Install first.
	if err := Install(); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	// Uninstall.
	if err := Uninstall(); err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}

	// Verify skill directories were removed.
	for _, name := range []string{"team-lead", "review-team-lead"} {
		path := filepath.Join(tmp, ".claude", "skills", name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", path)
		}
	}
}

func TestUninstallPreservesOtherSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Install st skills.
	if err := Install(); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	// Create a non-st skill.
	otherDir := filepath.Join(tmp, ".claude", "skills", "my-custom-skill")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "SKILL.md"), []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Uninstall st skills.
	if err := Uninstall(); err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}

	// Custom skill should still exist.
	if _, err := os.Stat(filepath.Join(otherDir, "SKILL.md")); err != nil {
		t.Errorf("expected custom skill to be preserved: %v", err)
	}
}
