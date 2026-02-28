package workflow

import (
	"testing"

	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from ticket.Status
		to   ticket.Status
		want bool
	}{
		{ticket.StatusBacklog, ticket.StatusOpen, true},
		{ticket.StatusOpen, ticket.StatusInProgress, true},
		{ticket.StatusInProgress, ticket.StatusReview, true},
		{ticket.StatusReview, ticket.StatusDone, true},
		{ticket.StatusReview, ticket.StatusRework, true},
		{ticket.StatusRework, ticket.StatusInProgress, true},

		// Blocked transitions
		{ticket.StatusOpen, ticket.StatusBlocked, true},
		{ticket.StatusInProgress, ticket.StatusBlocked, true},
		{ticket.StatusBlocked, ticket.StatusOpen, true}, // snap back

		// Handoff transitions — IN-PROGRESS and REWORK can move to OPEN
		{ticket.StatusInProgress, ticket.StatusOpen, true},
		{ticket.StatusRework, ticket.StatusOpen, true},

		// Backlog transitions — any status can move to backlog
		{ticket.StatusOpen, ticket.StatusBacklog, true},
		{ticket.StatusInProgress, ticket.StatusBacklog, true},
		{ticket.StatusReview, ticket.StatusBacklog, true},
		{ticket.StatusRework, ticket.StatusBacklog, true},
		{ticket.StatusDone, ticket.StatusBacklog, true},

		// Cancel transitions — most active states can be cancelled
		{ticket.StatusBacklog, ticket.StatusCancelled, true},
		{ticket.StatusOpen, ticket.StatusCancelled, true},
		{ticket.StatusInProgress, ticket.StatusCancelled, true},
		{ticket.StatusReview, ticket.StatusCancelled, true},
		{ticket.StatusRework, ticket.StatusCancelled, true},
		{ticket.StatusCancelled, ticket.StatusBacklog, true},

		// Invalid transitions
		{ticket.StatusBacklog, ticket.StatusDone, false},
		{ticket.StatusOpen, ticket.StatusReview, false},
		{ticket.StatusOpen, ticket.StatusDone, false},
		{ticket.StatusRework, ticket.StatusDone, false},
		{ticket.StatusDone, ticket.StatusOpen, false},
		{ticket.StatusDone, ticket.StatusCancelled, false},
		{ticket.StatusCancelled, ticket.StatusOpen, false},
	}

	for _, tt := range tests {
		got := CanTransition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	// Valid
	if err := ValidateTransition(ticket.StatusOpen, ticket.StatusInProgress); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Same status
	if err := ValidateTransition(ticket.StatusOpen, ticket.StatusOpen); err == nil {
		t.Error("expected error for same status")
	}

	// Invalid
	if err := ValidateTransition(ticket.StatusBacklog, ticket.StatusDone); err == nil {
		t.Error("expected error for invalid transition")
	}
}

func TestStatusFromAlias(t *testing.T) {
	tests := []struct {
		input string
		want  ticket.Status
	}{
		{"review", ticket.StatusReview},
		{"submit", ticket.StatusReview},
		{"start", ticket.StatusInProgress},
		{"begin", ticket.StatusInProgress},
		{"done", ticket.StatusDone},
		{"complete", ticket.StatusDone},
		{"reject", ticket.StatusRework},
		{"cancel", ticket.StatusCancelled},
		{"cancelled", ticket.StatusCancelled},
		{"OPEN", ticket.StatusOpen},
		{"IN-PROGRESS", ticket.StatusInProgress},
		{"CANCELLED", ticket.StatusCancelled},
	}

	for _, tt := range tests {
		got, err := StatusFromAlias(tt.input)
		if err != nil {
			t.Errorf("StatusFromAlias(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("StatusFromAlias(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Invalid
	if _, err := StatusFromAlias("invalid"); err == nil {
		t.Error("expected error for invalid alias")
	}
}
