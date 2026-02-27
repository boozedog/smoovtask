package workflow

import (
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestRequiresAssignee(t *testing.T) {
	if !RequiresAssignee(ticket.StatusInProgress) {
		t.Error("IN-PROGRESS should require assignee")
	}
	if RequiresAssignee(ticket.StatusOpen) {
		t.Error("OPEN should not require assignee")
	}
	if RequiresAssignee(ticket.StatusReview) {
		t.Error("REVIEW should not require assignee")
	}
}

func TestRequiresNote(t *testing.T) {
	if !RequiresNote(ticket.StatusInProgress, ticket.StatusReview) {
		t.Error("IN-PROGRESS → REVIEW should require note")
	}
	if !RequiresNote(ticket.StatusReview, ticket.StatusDone) {
		t.Error("REVIEW → DONE should require note")
	}
	if !RequiresNote(ticket.StatusReview, ticket.StatusRework) {
		t.Error("REVIEW → REWORK should require note")
	}
	if RequiresNote(ticket.StatusOpen, ticket.StatusInProgress) {
		t.Error("OPEN → IN-PROGRESS should not require note")
	}
	if RequiresNote(ticket.StatusReview, ticket.StatusOpen) {
		t.Error("REVIEW → OPEN should not require note")
	}
}

func TestHasNoteSince(t *testing.T) {
	dir := t.TempDir()
	el := event.NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	// Add a status event and a note after it
	_ = el.Append(event.Event{
		TS:     ts,
		Event:  event.StatusInProgress,
		Ticket: "st_test01",
	})
	_ = el.Append(event.Event{
		TS:     ts.Add(time.Hour),
		Event:  event.TicketNote,
		Ticket: "st_test01",
		Data:   map[string]any{"message": "progress update"},
	})

	// Should find the note after the status change
	ok, err := HasNoteSince(dir, "st_test01", ts)
	if err != nil {
		t.Fatalf("HasNoteSince: %v", err)
	}
	if !ok {
		t.Error("should find note after status change time")
	}

	// Should not find a note after the note was written
	ok, err = HasNoteSince(dir, "st_test01", ts.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("HasNoteSince: %v", err)
	}
	if ok {
		t.Error("should not find note before the since time")
	}

	// Should not find a note for a different ticket
	ok, err = HasNoteSince(dir, "st_other1", ts)
	if err != nil {
		t.Fatalf("HasNoteSince: %v", err)
	}
	if ok {
		t.Error("should not find note for different ticket")
	}
}

func TestCanReview(t *testing.T) {
	dir := t.TempDir()
	el := event.NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	// sess-1 worked on the ticket
	_ = el.Append(event.Event{
		TS:     ts,
		Event:  event.StatusInProgress,
		Ticket: "st_test01",
		RunID:  "sess-1",
	})
	_ = el.Append(event.Event{
		TS:     ts.Add(time.Hour),
		Event:  event.StatusReview,
		Ticket: "st_test01",
		RunID:  "sess-1",
	})

	// sess-1 should NOT be able to review (touched it)
	ok, err := CanReview(dir, "st_test01", "sess-1")
	if err != nil {
		t.Fatalf("CanReview: %v", err)
	}
	if ok {
		t.Error("sess-1 should not be eligible to review (touched ticket)")
	}

	// sess-2 should be able to review (never touched it)
	ok, err = CanReview(dir, "st_test01", "sess-2")
	if err != nil {
		t.Fatalf("CanReview: %v", err)
	}
	if !ok {
		t.Error("sess-2 should be eligible to review")
	}
}
