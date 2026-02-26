package ticket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRandomID(t *testing.T) {
	id, err := randomID()
	if err != nil {
		t.Fatalf("randomID() error: %v", err)
	}

	if !strings.HasPrefix(id, IDPrefix) {
		t.Errorf("ID %q does not start with %q", id, IDPrefix)
	}

	suffix := id[len(IDPrefix):]
	if len(suffix) != IDLength {
		t.Errorf("ID suffix %q has length %d, want %d", suffix, len(suffix), IDLength)
	}

	for _, c := range suffix {
		if !strings.ContainsRune(base62Chars, c) {
			t.Errorf("ID suffix contains invalid char %q", c)
		}
	}
}

func TestRandomIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		id, err := randomID()
		if err != nil {
			t.Fatalf("randomID() error: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateID(t *testing.T) {
	dir := t.TempDir()

	id, err := GenerateID(dir)
	if err != nil {
		t.Fatalf("GenerateID() error: %v", err)
	}

	if !strings.HasPrefix(id, IDPrefix) {
		t.Errorf("ID %q does not start with %q", id, IDPrefix)
	}
}

func TestGenerateIDNonexistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	id, err := GenerateID(dir)
	if err != nil {
		t.Fatalf("GenerateID() error: %v", err)
	}

	if !strings.HasPrefix(id, IDPrefix) {
		t.Errorf("ID %q does not start with %q", id, IDPrefix)
	}
}

func TestGenerateIDCollisionCheck(t *testing.T) {
	dir := t.TempDir()

	// Create a file with a known ID
	fname := "2026-02-25T10:00-st_abc123.md"
	if err := os.WriteFile(filepath.Join(dir, fname), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Generate should not return st_abc123
	for range 50 {
		id, err := GenerateID(dir)
		if err != nil {
			t.Fatalf("GenerateID() error: %v", err)
		}
		if id == "st_abc123" {
			t.Fatal("GenerateID returned a colliding ID")
		}
	}
}

func TestExtractIDFromFilename(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"2026-02-25T10:00-st_a7Kx2m.md", "st_a7Kx2m"},
		{"2026-02-25T10:30-st_b3Yz9q.md", "st_b3Yz9q"},
		{"random-file.md", ""},
		{"st_.md", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractIDFromFilename(tt.name)
		if got != tt.want {
			t.Errorf("extractIDFromFilename(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
