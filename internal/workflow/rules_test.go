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
	if RequiresNote(ticket.StatusOpen, ticket.StatusInProgress) {
		t.Error("OPEN → IN-PROGRESS should not require note")
	}
}

func TestCanReview(t *testing.T) {
	dir := t.TempDir()
	el := event.NewEventLog(dir)

	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	// sess-1 worked on the ticket
	_ = el.Append(event.Event{
		TS:      ts,
		Event:   event.StatusInProgress,
		Ticket:  "st_test01",
		Session: "sess-1",
	})
	_ = el.Append(event.Event{
		TS:      ts.Add(time.Hour),
		Event:   event.StatusReview,
		Ticket:  "st_test01",
		Session: "sess-1",
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
