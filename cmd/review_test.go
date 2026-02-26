package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestReview_CleanSession(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "clean-reviewer")

	tk := env.createTicket(t, "review me", ticket.StatusReview)

	out, err := env.runCmd(t, "review", tk.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Claimed "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Claimed "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Assignee != "clean-reviewer" {
		t.Errorf("assignee = %q, want %q", updated.Assignee, "clean-reviewer")
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == "ticket.review-claimed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no ticket.review-claimed event logged")
	}
}

func TestReview_SessionTouchedTicket(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "tainted-session")

	tk := env.createTicket(t, "tainted review", ticket.StatusReview)

	// Create an event that shows this session previously touched the ticket
	el := event.NewEventLog(env.EventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.StatusInProgress,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   "agent",
		Session: "tainted-session",
	})

	_, err := env.runCmd(t, "review", tk.ID)
	if err == nil {
		t.Fatal("expected error for session that touched ticket")
	}
	if !strings.Contains(err.Error(), "review denied") {
		t.Errorf("error = %q, want substring %q", err.Error(), "review denied")
	}
}

func TestReview_NotInReviewStatus(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "clean-reviewer")

	tk := env.createTicket(t, "not in review", ticket.StatusInProgress)

	_, err := env.runCmd(t, "review", tk.ID)
	if err == nil {
		t.Fatal("expected error for ticket not in REVIEW status")
	}
	if !strings.Contains(err.Error(), "not REVIEW") {
		t.Errorf("error = %q, want substring %q", err.Error(), "not REVIEW")
	}
}

func TestReview_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "clean-reviewer")
	_ = env

	_, err := env.runCmd(t, "review", "st_zzzzzz")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}
