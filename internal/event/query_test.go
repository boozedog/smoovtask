package event

import (
	"testing"
	"time"
)

// setupTestEvents writes a set of test events across multiple days.
func setupTestEvents(t *testing.T, dir string) {
	t.Helper()
	log := NewEventLog(dir)

	events := []Event{
		{
			TS:      time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC),
			Event:   TicketCreated,
			Ticket:  "st_aaa001",
			Project: "api-server",
			Actor:   "human",
			RunID:   "",
			Data:    map[string]any{"title": "First ticket"},
		},
		{
			TS:      time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
			Event:   StatusInProgress,
			Ticket:  "st_aaa001",
			Project: "api-server",
			Actor:   "agent-01",
			RunID:   "sess-abc",
		},
		{
			TS:      time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC),
			Event:   HookPostTool,
			Ticket:  "st_aaa001",
			Project: "api-server",
			Actor:   "agent-01",
			RunID:   "sess-abc",
			Data:    map[string]any{"tool": "Edit"},
		},
		{
			TS:      time.Date(2026, 2, 25, 14, 0, 0, 0, time.UTC),
			Event:   TicketCreated,
			Ticket:  "st_bbb002",
			Project: "smoovtask",
			Actor:   "human",
			RunID:   "",
			Data:    map[string]any{"title": "Second ticket"},
		},
		{
			TS:      time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC),
			Event:   StatusReview,
			Ticket:  "st_aaa001",
			Project: "api-server",
			Actor:   "agent-01",
			RunID:   "sess-abc",
		},
		{
			TS:      time.Date(2026, 2, 26, 9, 0, 0, 0, time.UTC),
			Event:   StatusInProgress,
			Ticket:  "st_bbb002",
			Project: "smoovtask",
			Actor:   "agent-02",
			RunID:   "sess-def",
		},
	}

	for _, e := range events {
		if err := log.Append(e); err != nil {
			t.Fatalf("setup append: %v", err)
		}
	}
}

func TestQueryByTicket(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	events, err := QueryEvents(dir, Query{TicketID: "st_aaa001"})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("got %d events, want 4", len(events))
	}

	for _, e := range events {
		if e.Ticket != "st_aaa001" {
			t.Errorf("unexpected ticket %q", e.Ticket)
		}
	}
}

func TestQueryByProject(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	events, err := QueryEvents(dir, Query{Project: "smoovtask"})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	for _, e := range events {
		if e.Project != "smoovtask" {
			t.Errorf("unexpected project %q", e.Project)
		}
	}
}

func TestQueryBySession(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	events, err := QueryEvents(dir, Query{RunID:   "sess-def"})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	if events[0].RunID != "sess-def" {
		t.Errorf("session = %q, want %q", events[0].RunID, "sess-def")
	}
}

func TestQueryByTimeRange(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	// Only events on Feb 25.
	after := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 2, 25, 23, 59, 59, 0, time.UTC)

	events, err := QueryEvents(dir, Query{After: after, Before: before})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	for _, e := range events {
		if e.TS.Before(after) || e.TS.After(before) {
			t.Errorf("event at %v is outside range", e.TS)
		}
	}
}

func TestQueryCombinedFilters(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	events, err := QueryEvents(dir, Query{
		TicketID: "st_aaa001",
		RunID:    "sess-abc",
	})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	// Should get the 3 events for st_aaa001 that have sess-abc (excludes the created event with no session).
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
}

func TestQueryEmptyDir(t *testing.T) {
	dir := t.TempDir()

	events, err := QueryEvents(dir, Query{})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestQueryNonexistentDir(t *testing.T) {
	events, err := QueryEvents("/nonexistent/path", Query{})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestRunIDsForTicket(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	sessions, err := RunIDsForTicket(dir, "st_aaa001")
	if err != nil {
		t.Fatalf("RunIDsForTicket: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}

	if sessions[0] != "sess-abc" {
		t.Errorf("session = %q, want %q", sessions[0], "sess-abc")
	}
}

func TestRunIDsForTicketMultiple(t *testing.T) {
	dir := t.TempDir()
	log := NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	// Same ticket, multiple sessions.
	for i, sess := range []string{"sess-1", "sess-2", "sess-1", "sess-3"} {
		e := Event{
			TS:      ts.Add(time.Duration(i) * time.Minute),
			Event:   HookPostTool,
			Ticket:  "st_multi1",
			RunID:   sess,
		}
		if err := log.Append(e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	sessions, err := RunIDsForTicket(dir, "st_multi1")
	if err != nil {
		t.Fatalf("RunIDsForTicket: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	// Verify uniqueness and order.
	want := []string{"sess-1", "sess-2", "sess-3"}
	for i, s := range sessions {
		if s != want[i] {
			t.Errorf("session[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestRunIDsForTicketNone(t *testing.T) {
	dir := t.TempDir()
	setupTestEvents(t, dir)

	sessions, err := RunIDsForTicket(dir, "st_nonexistent")
	if err != nil {
		t.Fatalf("RunIDsForTicket: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}
