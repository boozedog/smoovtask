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

		// Invalid transitions
		{ticket.StatusBacklog, ticket.StatusDone, false},
		{ticket.StatusOpen, ticket.StatusReview, false},
		{ticket.StatusOpen, ticket.StatusDone, false},
		{ticket.StatusRework, ticket.StatusDone, false},
		{ticket.StatusDone, ticket.StatusOpen, false},
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
		{"OPEN", ticket.StatusOpen},
		{"IN-PROGRESS", ticket.StatusInProgress},
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
