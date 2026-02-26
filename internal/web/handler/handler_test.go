package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/handler"
	"github.com/boozedog/smoovtask/internal/web/sse"
)

// testSetup creates temporary directories and returns a handler with test data.
func testSetup(t *testing.T) (*handler.Handler, string, string) {
	t.Helper()

	ticketsDir := t.TempDir()
	eventsDir := t.TempDir()

	// Create a test ticket.
	store := ticket.NewStore(ticketsDir)
	tk := &ticket.Ticket{
		ID:       "st_abc123",
		Title:    "Test ticket",
		Project:  "testproj",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP3,
		Created:  time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Body:     "## Description\n\nThis is a **test** ticket.",
	}
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	// Create another ticket with different status.
	tk2 := &ticket.Ticket{
		ID:       "st_def456",
		Title:    "In progress ticket",
		Project:  "testproj",
		Status:   ticket.StatusInProgress,
		Priority: ticket.PriorityP1,
		Assignee: "session-123",
		Created:  time.Date(2026, 2, 26, 11, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 2, 26, 11, 0, 0, 0, time.UTC),
		Body:     "Working on this.",
	}
	if err := store.Create(tk2); err != nil {
		t.Fatal(err)
	}

	// Create a test event.
	ev := event.Event{
		TS:      time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Event:   event.TicketCreated,
		Ticket:  "st_abc123",
		Project: "testproj",
		Actor:   "human",
	}
	evLog := event.NewEventLog(eventsDir)
	if err := evLog.Append(ev); err != nil {
		t.Fatal(err)
	}

	broker := sse.NewBroker()
	h := handler.New(ticketsDir, eventsDir, broker)
	return h, ticketsDir, eventsDir
}

func TestBoard(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.Board(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test ticket") {
		t.Error("expected board to contain 'Test ticket'")
	}
	if !strings.Contains(body, "In progress ticket") {
		t.Error("expected board to contain 'In progress ticket'")
	}
	if !strings.Contains(body, "smoovtask") {
		t.Error("expected board to contain 'smoovtask' in layout")
	}
}

func TestPartialBoard(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/board", nil)
	w := httptest.NewRecorder()

	h.PartialBoard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "st-board") {
		t.Error("expected partial to contain board CSS class")
	}
	// Partial should NOT include the full layout.
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("partial should not include full HTML layout")
	}
}

func TestList(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "st_abc123") {
		t.Error("expected list to contain ticket ID st_abc123")
	}
	if !strings.Contains(body, "st_def456") {
		t.Error("expected list to contain ticket ID st_def456")
	}
}

func TestListFilterByStatus(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/list?status=OPEN", nil)
	w := httptest.NewRecorder()

	h.PartialList(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "st_abc123") {
		t.Error("expected filtered list to contain OPEN ticket")
	}
	if strings.Contains(body, "st_def456") {
		t.Error("expected filtered list to exclude IN-PROGRESS ticket")
	}
}

func TestTicket(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/ticket/st_abc123", nil)
	req.SetPathValue("id", "st_abc123")
	w := httptest.NewRecorder()

	h.Ticket(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test ticket") {
		t.Error("expected ticket page to contain title")
	}
	// Goldmark should render **test** as <strong>test</strong>
	if !strings.Contains(body, "<strong>test</strong>") {
		t.Error("expected rendered markdown with bold text")
	}
}

func TestTicketNotFound(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/ticket/st_nonexist", nil)
	req.SetPathValue("id", "st_nonexist")
	w := httptest.NewRecorder()

	h.Ticket(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestActivity(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/activity", nil)
	w := httptest.NewRecorder()

	h.Activity(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, event.TicketCreated) {
		t.Error("expected activity page to contain ticket.created event")
	}
}

func TestActivityFilterByProject(t *testing.T) {
	h, _, eventsDir := testSetup(t)

	// Add an event for a different project.
	evLog := event.NewEventLog(eventsDir)
	ev := event.Event{
		TS:      time.Now().UTC(),
		Event:   event.StatusOpen,
		Ticket:  "st_xyz789",
		Project: "otherproj",
		Actor:   "agent",
	}
	if err := evLog.Append(ev); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/activity?project=testproj", nil)
	w := httptest.NewRecorder()

	h.Activity(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "st_abc123") {
		t.Error("expected testproj events in filtered activity")
	}
	if strings.Contains(body, "st_xyz789") {
		t.Error("expected otherproj events to be filtered out")
	}
}

func TestEvents(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()

	// Run the SSE handler in a goroutine since it blocks.
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.Events(w, req)
	}()

	// Give the handler a moment to start, then check headers.
	time.Sleep(50 * time.Millisecond)

	// The handler should have written the keepalive comment.
	body := w.Body.String()
	if !strings.Contains(body, ": keepalive") {
		t.Error("expected keepalive comment in SSE stream")
	}
}

func TestSSEBrokerBroadcast(t *testing.T) {
	broker := sse.NewBroker()

	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)

	ev := event.Event{
		TS:    time.Now().UTC(),
		Event: event.TicketCreated,
	}
	broker.Broadcast(ev)

	select {
	case received := <-ch:
		if received.Event != event.TicketCreated {
			t.Errorf("expected %s, got %s", event.TicketCreated, received.Event)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast event")
	}
}

func TestSSEBrokerMultipleClients(t *testing.T) {
	broker := sse.NewBroker()

	ch1 := broker.Subscribe()
	ch2 := broker.Subscribe()
	defer broker.Unsubscribe(ch1)
	defer broker.Unsubscribe(ch2)

	if broker.Count() != 2 {
		t.Fatalf("expected 2 clients, got %d", broker.Count())
	}

	ev := event.Event{Event: event.StatusDone}
	broker.Broadcast(ev)

	for _, ch := range []chan event.Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.Event != event.StatusDone {
				t.Errorf("expected %s, got %s", event.StatusDone, received.Event)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for broadcast")
		}
	}
}

func TestSSEBrokerUnsubscribe(t *testing.T) {
	broker := sse.NewBroker()

	ch := broker.Subscribe()
	if broker.Count() != 1 {
		t.Fatalf("expected 1 client, got %d", broker.Count())
	}

	broker.Unsubscribe(ch)
	if broker.Count() != 0 {
		t.Fatalf("expected 0 clients after unsubscribe, got %d", broker.Count())
	}
}

func TestWatcher(t *testing.T) {
	eventsDir := t.TempDir()
	broker := sse.NewBroker()

	watcher, err := sse.NewWatcher(eventsDir, broker)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)

	// Write a JSONL event to the watched directory.
	ev := event.Event{
		TS:      time.Now().UTC(),
		Event:   event.TicketCreated,
		Ticket:  "st_test01",
		Project: "testproj",
		Actor:   "human",
	}
	data, _ := json.Marshal(ev)
	jsonlPath := filepath.Join(eventsDir, time.Now().Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(jsonlPath, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	// The watcher should detect the new file and broadcast the event.
	select {
	case received := <-ch:
		if received.Ticket != "st_test01" {
			t.Errorf("expected ticket st_test01, got %s", received.Ticket)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for watcher to broadcast event")
	}
}
