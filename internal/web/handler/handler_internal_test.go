package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/web/sse"
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

func TestResolveRunLastHookTimes(t *testing.T) {
	eventsDir := t.TempDir()
	log := event.NewEventLog(eventsDir)

	appendEvent := func(ev event.Event) {
		t.Helper()
		if err := log.Append(ev); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	latestRun1 := time.Date(2026, 2, 28, 10, 3, 0, 0, time.UTC)
	latestRun2 := time.Date(2026, 2, 28, 10, 4, 0, 0, time.UTC)

	appendEvent(event.Event{TS: time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC), Event: event.HookSessionStart, RunID: "run-1"})
	appendEvent(event.Event{TS: time.Date(2026, 2, 28, 10, 1, 0, 0, time.UTC), Event: event.TicketCreated, RunID: "run-1"})
	appendEvent(event.Event{TS: latestRun1, Event: event.HookPostTool, RunID: "run-1"})
	appendEvent(event.Event{TS: latestRun2, Event: event.HookPreTool, RunID: "run-2"})

	h := &Handler{eventsDir: eventsDir}
	lastHooks := h.resolveRunLastHookTimes([]string{"run-1", "run-2", "run-3", ""})

	if got := lastHooks["run-1"]; !got.Equal(latestRun1) {
		t.Fatalf("run-1 last hook = %v, want %v", got, latestRun1)
	}
	if got := lastHooks["run-2"]; !got.Equal(latestRun2) {
		t.Fatalf("run-2 last hook = %v, want %v", got, latestRun2)
	}
	if _, ok := lastHooks["run-3"]; ok {
		t.Fatal("run-3 should be absent when there are no hook events")
	}
}

func TestEventsStreamsAgentPingAsPing(t *testing.T) {
	h := &Handler{broker: sse.NewBroker()}

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.Events(w, req)
	}()

	time.Sleep(50 * time.Millisecond)
	h.broker.Broadcast(event.Event{
		Event: "agent-ping",
		RunID: "ses_1",
		Data: map[string]any{
			"hook":   event.HookPreTool,
			"ticket": "st_test01",
		},
	})
	h.broker.Broadcast(event.Event{Event: "refresh-activity"})
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	if strings.Contains(body, "event: agent-ping") {
		t.Fatal("raw agent-ping event name should not appear in stream")
	}
	if !strings.Contains(body, "event: ping") {
		t.Fatal("expected agent-ping to be forwarded as 'event: ping'")
	}
	if !strings.Contains(body, `"run_id":"ses_1"`) {
		t.Fatal("expected ping payload to include run_id")
	}
	if !strings.Contains(body, `"ticket":"st_test01"`) {
		t.Fatal("expected ping payload to include ticket")
	}
	if !strings.Contains(body, "event: refresh-activity") {
		t.Fatal("expected refresh-activity event in stream")
	}
}
