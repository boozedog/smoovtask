package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Query defines optional filters for scanning events.
type Query struct {
	TicketID string
	Project  string
	RunID    string
	After    time.Time
	Before   time.Time
}

// QueryEvents scans relevant JSONL files and returns events matching the query.
func QueryEvents(dir string, q Query) ([]Event, error) {
	files, err := relevantFiles(dir, q)
	if err != nil {
		return nil, err
	}

	var results []Event
	for _, path := range files {
		events, err := scanFile(path, q)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", filepath.Base(path), err)
		}
		results = append(results, events...)
	}

	return results, nil
}

// RunIDsForTicket returns unique run IDs that appear in events for a ticket.
func RunIDsForTicket(dir, ticketID string) ([]string, error) {
	events, err := QueryEvents(dir, Query{TicketID: ticketID})
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var sessions []string
	for _, e := range events {
		if e.RunID == "" {
			continue
		}
		if _, ok := seen[e.RunID]; !ok {
			seen[e.RunID] = struct{}{}
			sessions = append(sessions, e.RunID)
		}
	}

	return sessions, nil
}

// relevantFiles returns sorted JSONL file paths that could contain matching events.
func relevantFiles(dir string, q Query) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read events dir: %w", err)
	}

	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		// Filter by date range if After/Before are set.
		if !q.After.IsZero() || !q.Before.IsZero() {
			dateStr := strings.TrimSuffix(name, ".jsonl")
			fileDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue // skip files with unexpected names
			}

			// File contains events for this entire day (00:00 to 23:59:59).
			fileEnd := fileDate.Add(24*time.Hour - time.Nanosecond)

			if !q.After.IsZero() && fileEnd.Before(q.After) {
				continue
			}
			if !q.Before.IsZero() && fileDate.After(q.Before) {
				continue
			}
		}

		paths = append(paths, filepath.Join(dir, name))
	}

	sort.Strings(paths)
	return paths, nil
}

// scanFile reads a single JSONL file and returns events matching the query.
func scanFile(path string, q Query) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	var results []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed lines
		}

		if matches(e, q) {
			results = append(results, e)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	return results, nil
}

// matches returns true if the event satisfies all non-zero query filters.
func matches(e Event, q Query) bool {
	if q.TicketID != "" && e.Ticket != q.TicketID {
		return false
	}
	if q.Project != "" && e.Project != q.Project {
		return false
	}
	if q.RunID != "" && e.RunID != q.RunID {
		return false
	}
	if !q.After.IsZero() && e.TS.Before(q.After) {
		return false
	}
	if !q.Before.IsZero() && e.TS.After(q.Before) {
		return false
	}
	return true
}
