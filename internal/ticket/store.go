package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/boozedog/smoovtask/internal/finder"
)

// Store provides file-based ticket storage.
type Store struct {
	projectsDir string
}

// NewStore creates a Store that reads/writes tickets under the given projects directory.
func NewStore(projectsDir string) *Store {
	return &Store{projectsDir: projectsDir}
}

// ListFilter defines optional filters for listing tickets.
type ListFilter struct {
	Project  string
	Status   Status
	Excludes []Status
}

// ticketDir returns the directory for a ticket based on project and creation time.
// Format: <projectsDir>/<project>/tickets/YYYY/MM
func (s *Store) ticketDir(t *Ticket) string {
	return filepath.Join(
		s.projectsDir,
		t.Project,
		"tickets",
		t.Created.UTC().Format("2006"),
		t.Created.UTC().Format("01"),
	)
}

// ticketPath returns the full file path for a ticket.
func (s *Store) ticketPath(t *Ticket) string {
	return filepath.Join(s.ticketDir(t), t.Filename())
}

// Create generates a new ticket and writes it to disk.
func (s *Store) Create(t *Ticket) error {
	if t.Project == "" {
		return fmt.Errorf("ticket project is required")
	}

	dir := s.ticketDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create ticket dir: %w", err)
	}

	if t.ID == "" {
		id, err := GenerateID(s.projectsDir)
		if err != nil {
			return fmt.Errorf("generate ID: %w", err)
		}
		t.ID = id
	}

	data, err := Render(t)
	if err != nil {
		return fmt.Errorf("render ticket: %w", err)
	}

	path := s.ticketPath(t)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write ticket: %w", err)
	}

	return nil
}

// Get retrieves a ticket by ID (exact match or prefix).
func (s *Store) Get(id string) (*Ticket, error) {
	path, err := s.findFile(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ticket: %w", err)
	}

	return Parse(data)
}

// Save writes an existing ticket back to disk.
func (s *Store) Save(t *Ticket) error {
	// Remove old file if it exists (ID matches but path might differ)
	oldPath, err := s.findFile(t.ID)
	if err == nil && oldPath != "" {
		newPath := s.ticketPath(t)
		if oldPath != newPath {
			os.Remove(oldPath)
		}
	}

	dir := s.ticketDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create ticket dir: %w", err)
	}

	data, err := Render(t)
	if err != nil {
		return fmt.Errorf("render ticket: %w", err)
	}

	path := s.ticketPath(t)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write ticket: %w", err)
	}

	return nil
}

// List returns all tickets, optionally filtered.
func (s *Store) List(filter ListFilter) ([]*Ticket, error) {
	return s.listWithParser(filter, Parse)
}

// ListMeta returns ticket frontmatter data only (Body is empty).
func (s *Store) ListMeta(filter ListFilter) ([]*Ticket, error) {
	return s.listWithParser(filter, ParseFrontmatter)
}

func (s *Store) listWithParser(filter ListFilter, parser func([]byte) (*Ticket, error)) ([]*Ticket, error) {
	searchDir := s.projectsDir
	if filter.Project != "" {
		searchDir = filepath.Join(s.projectsDir, filter.Project, "tickets")
	}

	files, err := finder.FindFiles(searchDir)
	if err != nil {
		return nil, fmt.Errorf("find ticket files: %w", err)
	}

	var tickets []*Ticket
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		t, err := parser(data)
		if err != nil {
			continue
		}

		if filter.Project != "" && t.Project != filter.Project {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if excluded(t.Status, filter.Excludes) {
			continue
		}

		tickets = append(tickets, t)
	}

	return tickets, nil
}

func excluded(s Status, excludes []Status) bool {
	return slices.Contains(excludes, s)
}

// findFile finds the file path for a ticket by ID (exact or prefix match).
func (s *Store) findFile(id string) (string, error) {
	files, err := finder.FindFiles(s.projectsDir)
	if err != nil {
		return "", fmt.Errorf("find ticket files: %w", err)
	}

	var matches []string
	for _, path := range files {
		name := filepath.Base(path)
		extracted := extractIDFromFilename(name)
		if extracted == "" {
			continue
		}
		if extracted == id {
			// Exact match — return immediately.
			return path, nil
		}
		if strings.HasPrefix(extracted, id) {
			matches = append(matches, path)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket %s not found", id)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous ticket prefix %q: %d matches", id, len(matches))
	}
}
