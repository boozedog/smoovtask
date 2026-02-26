package event

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAppendBasic(t *testing.T) {
	dir := t.TempDir()
	log := NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	e := Event{
		TS:      ts,
		Event:   TicketCreated,
		Ticket:  "st_a7Kx2m",
		Project: "api-server",
		Actor:   "human",
		Data:    map[string]any{"title": "Add rate limiting"},
	}

	if err := log.Append(e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Verify file exists with correct name.
	path := filepath.Join(dir, "2026-02-25.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	// Parse the line back.
	var got Event
	if err := json.Unmarshal(data[:len(data)-1], &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Ticket != "st_a7Kx2m" {
		t.Errorf("ticket = %q, want %q", got.Ticket, "st_a7Kx2m")
	}
	if got.Event != TicketCreated {
		t.Errorf("event = %q, want %q", got.Event, TicketCreated)
	}
	if got.Project != "api-server" {
		t.Errorf("project = %q, want %q", got.Project, "api-server")
	}

	// Verify trailing newline.
	if data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestAppendMultipleLines(t *testing.T) {
	dir := t.TempDir()
	log := NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	for i := range 3 {
		e := Event{
			TS:     ts.Add(time.Duration(i) * time.Minute),
			Event:  StatusOpen,
			Ticket: "st_test01",
		}
		if err := log.Append(e); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	path := filepath.Join(dir, "2026-02-25.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	if count != 3 {
		t.Errorf("line count = %d, want 3", count)
	}
}

func TestDailyRotation(t *testing.T) {
	dir := t.TempDir()
	log := NewEventLog(dir)

	day1 := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 26, 14, 30, 0, 0, time.UTC)
	day3 := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)

	for _, ts := range []time.Time{day1, day2, day3} {
		e := Event{
			TS:     ts,
			Event:  StatusOpen,
			Ticket: "st_rotate",
		}
		if err := log.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Verify three separate files.
	for _, name := range []string{"2026-02-25.jsonl", "2026-02-26.jsonl", "2026-03-01.jsonl"} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %s is empty", name)
		}
	}
}

func TestConcurrentAppends(t *testing.T) {
	dir := t.TempDir()
	log := NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			e := Event{
				TS:      ts,
				Event:   HookPostTool,
				Ticket:  "st_conc01",
				Session: "session-test",
				Data:    map[string]any{"index": i},
			}
			if err := log.Append(e); err != nil {
				t.Errorf("concurrent Append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all lines were written and are valid JSON.
	path := filepath.Join(dir, "2026-02-25.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Errorf("line %d: invalid JSON: %v", count, err)
		}
		count++
	}
	if count != n {
		t.Errorf("line count = %d, want %d", count, n)
	}
}
