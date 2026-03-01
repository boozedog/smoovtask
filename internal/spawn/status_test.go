package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
)

func writeEvent(t *testing.T, dir string, e event.Event) {
	t.Helper()
	filename := e.TS.UTC().Format("2006-01-02") + ".jsonl"
	path := filepath.Join(dir, filename)
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.Write(data)
	f.Write([]byte("\n"))
}

func TestGetWorkerInfoNoEvents(t *testing.T) {
	dir := t.TempDir()
	info, err := GetWorkerInfo(dir, "st_abc123")
	if err != nil {
		t.Fatal(err)
	}
	if info != nil {
		t.Error("expected nil info when no events exist")
	}
}

func TestGetWorkerInfoCompleted(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()

	writeEvent(t, dir, event.Event{
		TS:     now.Add(-10 * time.Minute),
		Event:  SpawnStarted,
		Ticket: "st_abc123",
		RunID:  "spawn-test",
		Data: map[string]any{
			"pid":      float64(12345),
			"worktree": "/tmp/wt",
			"branch":   "st/st_abc123",
			"backend":  "claude",
		},
	})
	writeEvent(t, dir, event.Event{
		TS:     now,
		Event:  SpawnCompleted,
		Ticket: "st_abc123",
		RunID:  "spawn-test",
		Data: map[string]any{
			"pid":       float64(12345),
			"exit_code": float64(0),
			"elapsed":   "10m0s",
		},
	})

	info, err := GetWorkerInfo(dir, "st_abc123")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.State != WorkerCompleted {
		t.Errorf("state = %q, want %q", info.State, WorkerCompleted)
	}
	if info.Branch != "st/st_abc123" {
		t.Errorf("branch = %q, want %q", info.Branch, "st/st_abc123")
	}
}

func TestGetWorkerInfoFailed(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()

	writeEvent(t, dir, event.Event{
		TS:     now.Add(-5 * time.Minute),
		Event:  SpawnStarted,
		Ticket: "st_fail",
		RunID:  "spawn-fail",
		Data: map[string]any{
			"pid":      float64(99999),
			"worktree": "/tmp/wt-fail",
			"branch":   "st/st_fail",
		},
	})
	writeEvent(t, dir, event.Event{
		TS:     now,
		Event:  SpawnFailed,
		Ticket: "st_fail",
		RunID:  "spawn-fail",
		Data: map[string]any{
			"pid":       float64(99999),
			"exit_code": float64(1),
			"elapsed":   "5m0s",
			"error":     "exit status 1",
		},
	})

	info, err := GetWorkerInfo(dir, "st_fail")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.State != WorkerFailed {
		t.Errorf("state = %q, want %q", info.State, WorkerFailed)
	}
}

func TestGetWorkerInfoStale(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()

	// Spawn event with a PID that doesn't exist
	writeEvent(t, dir, event.Event{
		TS:     now.Add(-30 * time.Minute),
		Event:  SpawnStarted,
		Ticket: "st_stale",
		RunID:  "spawn-stale",
		Data: map[string]any{
			"pid":      float64(999999999), // very unlikely to be a real PID
			"worktree": "/tmp/wt-stale",
			"branch":   "st/st_stale",
		},
	})

	info, err := GetWorkerInfo(dir, "st_stale")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.State != WorkerStale {
		t.Errorf("state = %q, want %q", info.State, WorkerStale)
	}
}

func TestGetWorkerInfoTimeout(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()

	writeEvent(t, dir, event.Event{
		TS:     now.Add(-45 * time.Minute),
		Event:  SpawnStarted,
		Ticket: "st_timeout",
		RunID:  "spawn-timeout",
		Data: map[string]any{
			"pid":      float64(11111),
			"worktree": "/tmp/wt-timeout",
			"branch":   "st/st_timeout",
		},
	})
	writeEvent(t, dir, event.Event{
		TS:     now,
		Event:  SpawnTimeout,
		Ticket: "st_timeout",
		RunID:  "spawn-timeout",
		Data: map[string]any{
			"pid":     float64(11111),
			"elapsed": "45m0s",
		},
	})

	info, err := GetWorkerInfo(dir, "st_timeout")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.State != WorkerTimeout {
		t.Errorf("state = %q, want %q", info.State, WorkerTimeout)
	}
}
