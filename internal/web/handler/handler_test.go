package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	h := handler.New(ticketsDir, eventsDir, broker, "testproj")
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
	if !strings.Contains(body, "sse:refresh-work") {
		t.Error("expected partial to contain SSE work-refresh trigger")
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

	req := httptest.NewRequest(http.MethodGet, "/partials/list-content?status=OPEN", nil)
	w := httptest.NewRecorder()

	h.PartialListContent(w, req)

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

func TestPartialActivity(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/activity", nil)
	w := httptest.NewRecorder()

	h.PartialActivity(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, event.TicketCreated) {
		t.Error("expected partial activity to contain ticket.created event")
	}
	if !strings.Contains(body, "sse:refresh-activity") {
		t.Error("expected partial to contain SSE activity-refresh trigger")
	}
	// Partial should NOT include the full layout.
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("partial should not include full HTML layout")
	}
}

func TestPartialActivityContentPushesURL(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/activity-content?project=testproj&event_type=ticket", nil)
	w := httptest.NewRecorder()

	h.PartialActivityContent(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	pushURL := w.Header().Get("HX-Push-Url")
	if pushURL == "" {
		t.Fatal("expected HX-Push-Url header to be set")
	}
	if !strings.Contains(pushURL, "/activity?") {
		t.Errorf("expected push URL to start with /activity?, got %q", pushURL)
	}
	if !strings.Contains(pushURL, "project=testproj") {
		t.Errorf("expected push URL to contain project=testproj, got %q", pushURL)
	}
	if !strings.Contains(pushURL, "event_type=ticket") {
		t.Errorf("expected push URL to contain event_type=ticket, got %q", pushURL)
	}
}

func TestPartialActivityContentPushesURLNoFilters(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/activity-content", nil)
	w := httptest.NewRecorder()

	h.PartialActivityContent(w, req)

	pushURL := w.Header().Get("HX-Push-Url")
	if pushURL != "/activity" {
		t.Errorf("expected push URL /activity with no filters, got %q", pushURL)
	}
}

func TestPartialActivityContentStripsEmptyParams(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/activity-content?event_type=&project=testproj", nil)
	w := httptest.NewRecorder()

	h.PartialActivityContent(w, req)

	pushURL := w.Header().Get("HX-Push-Url")
	if strings.Contains(pushURL, "event_type=") {
		t.Errorf("expected empty event_type param to be stripped, got %q", pushURL)
	}
	if !strings.Contains(pushURL, "project=testproj") {
		t.Errorf("expected project=testproj to remain, got %q", pushURL)
	}
}

func TestActivityFilterByEventType(t *testing.T) {
	h, _, eventsDir := testSetup(t)

	// Add a hook event.
	evLog := event.NewEventLog(eventsDir)
	ev := event.Event{
		TS:      time.Now().UTC(),
		Event:   "hook.session_start",
		Ticket:  "",
		Project: "testproj",
		Actor:   "agent",
	}
	if err := evLog.Append(ev); err != nil {
		t.Fatal(err)
	}

	// Filter for ticket events only — should include ticket.created but not hook.session_start.
	req := httptest.NewRequest(http.MethodGet, "/partials/activity-content?event_type=ticket", nil)
	w := httptest.NewRecorder()

	h.PartialActivityContent(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "ticket.created") {
		t.Error("expected ticket events in filtered activity")
	}
	if strings.Contains(body, "hook.session_start") {
		t.Error("expected hook events to be filtered out")
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

func TestTicketTimestampFormatting(t *testing.T) {
	h, ticketsDir, _ := testSetup(t)

	// Create a ticket with ISO timestamps in the body.
	store := ticket.NewStore(ticketsDir)
	tk := &ticket.Ticket{
		ID:       "st_ts0001",
		Title:    "Timestamp test",
		Project:  "testproj",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP3,
		Created:  time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Body:     "Created at 2026-02-26T02:46:49Z and updated at 2026-02-26T10:30:00+00:00.",
	}
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ticket/st_ts0001", nil)
	req.SetPathValue("id", "st_ts0001")
	w := httptest.NewRecorder()

	h.Ticket(w, req)

	body := w.Body.String()
	// ISO timestamps should be reformatted.
	if strings.Contains(body, "2026-02-26T02:46:49Z") {
		t.Error("expected ISO timestamp to be reformatted")
	}
	if !strings.Contains(body, "2026-02-26 02:46") {
		t.Error("expected reformatted timestamp 2026-02-26 02:46")
	}
}

func TestListSortOrder(t *testing.T) {
	h, ticketsDir, _ := testSetup(t)

	// Add a DONE ticket.
	store := ticket.NewStore(ticketsDir)
	tk := &ticket.Ticket{
		ID:       "st_done01",
		Title:    "Done ticket",
		Project:  "testproj",
		Status:   ticket.StatusDone,
		Priority: ticket.PriorityP3,
		Created:  time.Date(2026, 2, 26, 9, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC),
	}
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/partials/list-content", nil)
	w := httptest.NewRecorder()
	h.PartialListContent(w, req)

	body := w.Body.String()
	// Active tickets (OPEN, IN-PROGRESS) should appear before DONE.
	openIdx := strings.Index(body, "st_abc123")
	doneIdx := strings.Index(body, "st_done01")
	if openIdx == -1 || doneIdx == -1 {
		t.Fatal("expected both tickets in output")
	}
	if openIdx > doneIdx {
		t.Error("expected active tickets before DONE tickets")
	}
}

func TestListSortByPriority(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/list-content?sort=priority&dir=asc", nil)
	w := httptest.NewRecorder()
	h.PartialListContent(w, req)

	body := w.Body.String()
	// P1 (st_def456) should appear before P3 (st_abc123) when sorted by priority asc.
	p1Idx := strings.Index(body, "st_def456")
	p3Idx := strings.Index(body, "st_abc123")
	if p1Idx == -1 || p3Idx == -1 {
		t.Fatal("expected both tickets in output")
	}
	if p1Idx > p3Idx {
		t.Error("expected P1 ticket before P3 ticket when sorted by priority asc")
	}
}

func TestPartialListContentPushesURL(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/list-content?project=testproj&status=OPEN", nil)
	w := httptest.NewRecorder()
	h.PartialListContent(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	pushURL := w.Header().Get("HX-Push-Url")
	if pushURL == "" {
		t.Fatal("expected HX-Push-Url header to be set")
	}
	if !strings.Contains(pushURL, "/list?") {
		t.Errorf("expected push URL to start with /list?, got %q", pushURL)
	}
	if !strings.Contains(pushURL, "project=testproj") {
		t.Errorf("expected push URL to contain project=testproj, got %q", pushURL)
	}
}

func TestPartialListContentPushesURLNoFilters(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/list-content", nil)
	w := httptest.NewRecorder()
	h.PartialListContent(w, req)

	pushURL := w.Header().Get("HX-Push-Url")
	if pushURL != "/list" {
		t.Errorf("expected push URL /list with no filters, got %q", pushURL)
	}
}

func TestPartialListContentStripsEmptyParams(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/partials/list-content?status=&project=testproj", nil)
	w := httptest.NewRecorder()
	h.PartialListContent(w, req)

	pushURL := w.Header().Get("HX-Push-Url")
	if strings.Contains(pushURL, "status=") {
		t.Errorf("expected empty status param to be stripped, got %q", pushURL)
	}
	if !strings.Contains(pushURL, "project=testproj") {
		t.Errorf("expected project=testproj to remain, got %q", pushURL)
	}
}

func TestActivityDefaultExcludesHooks(t *testing.T) {
	h, _, eventsDir := testSetup(t)

	// Add a hook event.
	evLog := event.NewEventLog(eventsDir)
	hookEv := event.Event{
		TS:      time.Now().UTC(),
		Event:   "hook.session_start",
		Project: "testproj",
		Actor:   "agent",
	}
	if err := evLog.Append(hookEv); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/partials/activity-content", nil)
	w := httptest.NewRecorder()
	h.PartialActivityContent(w, req)

	body := w.Body.String()
	if strings.Contains(body, "hook.session_start") {
		t.Error("expected hook events to be hidden by default")
	}
	if !strings.Contains(body, "ticket.created") {
		t.Error("expected ticket events to still be visible")
	}
}

func TestActivityExplicitHookFilter(t *testing.T) {
	h, _, eventsDir := testSetup(t)

	// Add a hook event.
	evLog := event.NewEventLog(eventsDir)
	hookEv := event.Event{
		TS:      time.Now().UTC(),
		Event:   "hook.session_start",
		Project: "testproj",
		Actor:   "agent",
	}
	if err := evLog.Append(hookEv); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/partials/activity-content?event_type=hook", nil)
	w := httptest.NewRecorder()
	h.PartialActivityContent(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "hook.session_start") {
		t.Error("expected hook events to be visible when explicitly filtered")
	}
	if strings.Contains(body, "ticket.created") {
		t.Error("expected ticket events to be hidden when filtering for hooks")
	}
}

func TestNewTicketPage(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	w := httptest.NewRecorder()

	h.NewTicket(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "New Ticket") {
		t.Fatal("expected new ticket page content")
	}
}

func TestCreateTicket(t *testing.T) {
	h, _, _ := testSetup(t)

	form := url.Values{}
	form.Set("title", "Created in web")
	form.Set("project", "testproj")
	form.Set("status", "OPEN")
	form.Set("priority", "P2")
	form.Set("description", "web form ticket")

	req := httptest.NewRequest(http.MethodPost, "/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.CreateTicket(w, req)

	if w.Result().StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Result().StatusCode)
	}
	if !strings.HasPrefix(w.Header().Get("Location"), "/ticket/st_") {
		t.Fatalf("expected redirect to /ticket/st_*, got %q", w.Header().Get("Location"))
	}
}

func TestCriticalPathPage(t *testing.T) {
	h, ticketsDir, _ := testSetup(t)
	store := ticket.NewStore(ticketsDir)

	dep := &ticket.Ticket{
		ID:       "st_cpdep",
		Title:    "Dep",
		Project:  "testproj",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP3,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(dep); err != nil {
		t.Fatal(err)
	}

	root := &ticket.Ticket{
		ID:        "st_cproot",
		Title:     "Root",
		Project:   "testproj",
		Status:    ticket.StatusOpen,
		Priority:  ticket.PriorityP3,
		DependsOn: []string{"st_cpdep"},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
	}
	if err := store.Create(root); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/critical-path", nil)
	w := httptest.NewRecorder()

	h.CriticalPath(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Critical Path") {
		t.Fatal("expected critical path page content")
	}
	if !strings.Contains(body, "st_cproot") {
		t.Fatal("expected path to include root ticket")
	}
}

func TestCriticalPathHorizontalView(t *testing.T) {
	h, ticketsDir, _ := testSetup(t)
	store := ticket.NewStore(ticketsDir)

	dep := &ticket.Ticket{
		ID:       "st_cphdep",
		Title:    "Dep",
		Project:  "testproj",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP3,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(dep); err != nil {
		t.Fatal(err)
	}

	root := &ticket.Ticket{
		ID:        "st_cphroot",
		Title:     "Root",
		Project:   "testproj",
		Status:    ticket.StatusOpen,
		Priority:  ticket.PriorityP3,
		DependsOn: []string{"st_cphdep"},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
	}
	if err := store.Create(root); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/critical-path?view=horizontal", nil)
	w := httptest.NewRecorder()

	h.CriticalPath(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Horizontal") {
		t.Fatal("expected horizontal toggle")
	}
	if !strings.Contains(body, "→") {
		t.Fatal("expected horizontal arrow")
	}
}

func TestCriticalPathScopeDefaultsToAllAndSupportsCurrent(t *testing.T) {
	h, ticketsDir, _ := testSetup(t)
	store := ticket.NewStore(ticketsDir)

	allDep := &ticket.Ticket{
		ID:       "st_alldep",
		Title:    "All dep",
		Project:  "otherproj",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP3,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(allDep); err != nil {
		t.Fatal(err)
	}
	allRoot := &ticket.Ticket{
		ID:        "st_allroot",
		Title:     "All root",
		Project:   "otherproj",
		Status:    ticket.StatusOpen,
		Priority:  ticket.PriorityP3,
		DependsOn: []string{"st_alldep"},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
	}
	if err := store.Create(allRoot); err != nil {
		t.Fatal(err)
	}

	curDep := &ticket.Ticket{
		ID:       "st_curdep",
		Title:    "Current dep",
		Project:  "testproj",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP3,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(curDep); err != nil {
		t.Fatal(err)
	}
	curRoot := &ticket.Ticket{
		ID:        "st_curroot",
		Title:     "Current root",
		Project:   "testproj",
		Status:    ticket.StatusOpen,
		Priority:  ticket.PriorityP3,
		DependsOn: []string{"st_curdep"},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
	}
	if err := store.Create(curRoot); err != nil {
		t.Fatal(err)
	}

	allReq := httptest.NewRequest(http.MethodGet, "/critical-path", nil)
	allW := httptest.NewRecorder()
	h.CriticalPath(allW, allReq)

	if allW.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", allW.Result().StatusCode)
	}
	allBody := allW.Body.String()
	if !strings.Contains(allBody, "st_allroot") {
		t.Fatal("expected default scope to include other project path")
	}

	curReq := httptest.NewRequest(http.MethodGet, "/critical-path?scope=current", nil)
	curW := httptest.NewRecorder()
	h.CriticalPath(curW, curReq)

	if curW.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", curW.Result().StatusCode)
	}
	curBody := curW.Body.String()
	if strings.Contains(curBody, "st_allroot") {
		t.Fatal("expected current scope to exclude other project path")
	}
	if !strings.Contains(curBody, "st_curroot") {
		t.Fatal("expected current scope to include current project path")
	}
}

func TestWatcher(t *testing.T) {
	eventsDir := t.TempDir()
	broker := sse.NewBroker()

	watcher, err := sse.NewWatcher(eventsDir, broker)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Close() }()

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

	// Ticket/status events trigger work refresh (and activity refresh).
	select {
	case received := <-ch:
		if received.Event != "refresh-work" {
			t.Errorf("expected refresh-work event, got %s", received.Event)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watcher to broadcast event")
	}
}

func TestWatcherHookEventBroadcastsActivityOnly(t *testing.T) {
	eventsDir := t.TempDir()
	broker := sse.NewBroker()

	watcher, err := sse.NewWatcher(eventsDir, broker)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Close() }()

	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)

	// Write a hook event. This should refresh activity only.
	ev := event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPreTool,
		Ticket:  "st_test02",
		Project: "testproj",
		Actor:   "agent",
	}
	data, _ := json.Marshal(ev)
	jsonlPath := filepath.Join(eventsDir, time.Now().Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(jsonlPath, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case received := <-ch:
		if received.Event != "refresh-activity" {
			t.Fatalf("expected refresh-activity event, got %s", received.Event)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watcher to broadcast activity event")
	}

	select {
	case received := <-ch:
		if received.Event == "refresh-work" {
			t.Fatalf("did not expect refresh-work event for hook-only write")
		}
	case <-time.After(200 * time.Millisecond):
		// no extra work refresh is expected
	}
}
