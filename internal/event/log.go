package event

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// EventLog manages appending events to daily JSONL files.
type EventLog struct {
	dir string
}

// NewEventLog creates an EventLog that writes to the given directory.
func NewEventLog(dir string) *EventLog {
	return &EventLog{dir: dir}
}

// Append writes a single event as a JSON line to the daily file.
// The daily file is determined by the event's timestamp (YYYY-MM-DD.jsonl).
// File locking via flock ensures concurrent safety.
func (l *EventLog) Append(e Event) error {
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return fmt.Errorf("create events dir: %w", err)
	}

	filename := e.TS.UTC().Format("2006-01-02") + ".jsonl"
	path := filepath.Join(l.dir, filename)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open event file: %w", err)
	}
	defer f.Close()

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("lock event file: %w", err)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN)

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	data = append(data, '\n')

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	return nil
}
