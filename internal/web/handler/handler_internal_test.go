package handler

import (
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
)

func TestResolveRunSources(t *testing.T) {
	eventsDir := t.TempDir()
	log := event.NewEventLog(eventsDir)

	appendEvent := func(ev event.Event) {
		t.Helper()
		if err := log.Append(ev); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	appendEvent(event.Event{
		TS:     time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC),
		Event:  event.HookSessionStart,
		RunID:  "run-1",
		Source: "claude",
	})
	appendEvent(event.Event{
		TS:     time.Date(2026, 2, 28, 10, 1, 0, 0, time.UTC),
		Event:  event.HookSessionStart,
		RunID:  "run-2",
		Source: "opencode",
	})
	appendEvent(event.Event{
		TS:    time.Date(2026, 2, 28, 10, 2, 0, 0, time.UTC),
		Event: event.HookSessionStart,
		RunID: "run-3",
	})
	appendEvent(event.Event{
		TS:     time.Date(2026, 2, 28, 10, 3, 0, 0, time.UTC),
		Event:  event.HookSessionStart,
		RunID:  "run-1",
		Source: "opencode",
	})

	h := &Handler{eventsDir: eventsDir}

	runSources := h.resolveRunSources([]string{"run-1", "run-2", "run-3", "run-missing", ""})

	if got := runSources["run-1"]; got != "opencode" {
		t.Fatalf("run-1 source = %q, want %q", got, "opencode")
	}
	if got := runSources["run-2"]; got != "opencode" {
		t.Fatalf("run-2 source = %q, want %q", got, "opencode")
	}
	if _, ok := runSources["run-3"]; ok {
		t.Fatal("run-3 should be absent when source is empty")
	}
	if _, ok := runSources["run-missing"]; ok {
		t.Fatal("run-missing should be absent")
	}
}
