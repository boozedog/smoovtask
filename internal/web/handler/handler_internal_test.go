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

func TestEventsSkipsAgentPing(t *testing.T) {
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
	h.broker.Broadcast(event.Event{Event: "agent-ping", RunID: "ses_1"})
	h.broker.Broadcast(event.Event{Event: "refresh-activity"})
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	if strings.Contains(body, "event: agent-ping") {
		t.Fatal("agent-ping should not be forwarded to global /events stream")
	}
	if !strings.Contains(body, "event: refresh-activity") {
		t.Fatal("expected refresh-activity event in /events stream")
	}
}

func TestAgentEventsStreamsAllRunIDsWithoutPathFilter(t *testing.T) {
	h := &Handler{broker: sse.NewBroker()}

	req := httptest.NewRequest(http.MethodGet, "/events/agent", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.AgentEvents(w, req)
	}()

	time.Sleep(50 * time.Millisecond)
	h.broker.Broadcast(event.Event{Event: "agent-ping", RunID: "ses_any"})
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		body := w.Body.String()
		if strings.Contains(body, "event: ping") && strings.Contains(body, `"run_id":"ses_any"`) {
			cancel()
			<-done
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done
	t.Fatal("expected ping payload with run_id on unfiltered agent stream")
}

func TestAgentEventsIncludesTicketWhenPresent(t *testing.T) {
	h := &Handler{broker: sse.NewBroker()}

	req := httptest.NewRequest(http.MethodGet, "/events/agent", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.AgentEvents(w, req)
	}()

	time.Sleep(50 * time.Millisecond)
	h.broker.Broadcast(event.Event{
		Event: "agent-ping",
		RunID: "ses_any",
		Data: map[string]any{
			"hook":   event.HookPreTool,
			"ticket": "st_ping001",
		},
	})

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		body := w.Body.String()
		if strings.Contains(body, "event: ping") &&
			strings.Contains(body, `"run_id":"ses_any"`) &&
			strings.Contains(body, `"ticket":"st_ping001"`) {
			cancel()
			<-done
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done
	t.Fatal("expected ping payload with ticket when broker event includes ticket")
}

func TestAgentEventsStreamsOnlyMatchingRunID(t *testing.T) {
	h := &Handler{broker: sse.NewBroker()}

	req := httptest.NewRequest(http.MethodGet, "/events/agent/ses_match", nil)
	req.SetPathValue("runID", "ses_match")
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.AgentEvents(w, req)
	}()

	time.Sleep(50 * time.Millisecond)
	h.broker.Broadcast(event.Event{Event: "agent-ping", RunID: "ses_other"})
	time.Sleep(50 * time.Millisecond)
	if strings.Contains(w.Body.String(), "event: ping") {
		cancel()
		<-done
		t.Fatal("unexpected ping for non-matching run ID")
	}

	h.broker.Broadcast(event.Event{Event: "agent-ping", RunID: "ses_match"})
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		body := w.Body.String()
		if strings.Contains(body, "event: ping") && strings.Contains(body, `"run_id":"ses_match"`) {
			cancel()
			<-done
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done
	t.Fatal("expected ping for matching run ID")
}
