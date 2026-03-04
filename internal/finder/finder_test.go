package finder

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFindFiles_Walk(t *testing.T) {
	// Force walk mode for this test.
	origUseFd := UseFd
	UseFd = false
	t.Cleanup(func() { UseFd = origUseFd })

	dir := t.TempDir()

	// Create nested structure mimicking project layout.
	nested := filepath.Join(dir, "proj", "tickets", "2026", "03")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	ticketFile := filepath.Join(nested, "2026-03-01T10:00-st_abc123.md")
	if err := os.WriteFile(ticketFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Non-ticket file should be ignored.
	nonTicket := filepath.Join(nested, "notes.md")
	if err := os.WriteFile(nonTicket, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := FindFiles(dir)
	if err != nil {
		t.Fatalf("FindFiles() error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != ticketFile {
		t.Errorf("got %q, want %q", files[0], ticketFile)
	}
}

func TestFindFiles_NonexistentDir(t *testing.T) {
	origUseFd := UseFd
	UseFd = false
	t.Cleanup(func() { UseFd = origUseFd })

	files, err := FindFiles(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("FindFiles() error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestFindFiles_MultipleProjects(t *testing.T) {
	origUseFd := UseFd
	UseFd = false
	t.Cleanup(func() { UseFd = origUseFd })

	dir := t.TempDir()

	projects := []struct {
		path string
		file string
	}{
		{"projA/tickets/2026/02", "2026-02-15T10:00-st_aaa111.md"},
		{"projA/tickets/2026/03", "2026-03-01T10:00-st_bbb222.md"},
		{"projB/tickets/2026/02", "2026-02-20T10:00-st_ccc333.md"},
	}

	for _, p := range projects {
		full := filepath.Join(dir, p.path)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(full, p.file), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := FindFiles(dir)
	if err != nil {
		t.Fatalf("FindFiles() error: %v", err)
	}

	sort.Strings(files)
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %v", len(files), files)
	}
}
