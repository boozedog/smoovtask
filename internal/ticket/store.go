package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Store provides file-based ticket storage.
type Store struct {
	ticketsDir string
}

// NewStore creates a Store that reads/writes tickets in the given directory.
func NewStore(ticketsDir string) *Store {
	return &Store{ticketsDir: ticketsDir}
}

// ListFilter defines optional filters for listing tickets.
type ListFilter struct {
	Project  string
	Status   Status
	Excludes []Status
}

// Create generates a new ticket and writes it to disk.
func (s *Store) Create(t *Ticket) error {
	if err := os.MkdirAll(s.ticketsDir, 0o755); err != nil {
		return fmt.Errorf("create tickets dir: %w", err)
	}

	if t.ID == "" {
		id, err := GenerateID(s.ticketsDir)
		if err != nil {
			return fmt.Errorf("generate ID: %w", err)
		}
		t.ID = id
	}

	data, err := Render(t)
	if err != nil {
		return fmt.Errorf("render ticket: %w", err)
	}

	path := filepath.Join(s.ticketsDir, t.Filename())
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
	// Remove old file if it exists (ID matches but filename might differ due to timestamp)
	oldPath, err := s.findFile(t.ID)
	if err == nil && oldPath != "" {
		path := filepath.Join(s.ticketsDir, t.Filename())
		if oldPath != path {
			os.Remove(oldPath)
		}
	}

	data, err := Render(t)
	if err != nil {
		return fmt.Errorf("render ticket: %w", err)
	}

	path := filepath.Join(s.ticketsDir, t.Filename())
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
	entries, err := os.ReadDir(s.ticketsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read tickets dir: %w", err)
	}

	var tickets []*Ticket
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.ticketsDir, entry.Name()))
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
	entries, err := os.ReadDir(s.ticketsDir)
	if err != nil {
		return "", fmt.Errorf("read tickets dir: %w", err)
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		extracted := extractIDFromFilename(name)
		if extracted == "" {
			continue
		}
		if extracted == id {
			// Exact match — return immediately.
			return filepath.Join(s.ticketsDir, name), nil
		}
		if strings.HasPrefix(extracted, id) {
			matches = append(matches, filepath.Join(s.ticketsDir, name))
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
