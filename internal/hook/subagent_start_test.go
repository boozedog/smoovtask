package hook

import (
	"testing"
)

func TestTicketIDPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Work on sb_a7Kx2m: add rate limiting", "sb_a7Kx2m"},
		{"sb_Qr9fZw is the target", "sb_Qr9fZw"},
		{"no ticket here", ""},
		{"sb_ too short", ""},
		{"sb_!@#$%^ invalid chars", ""},
	}

	for _, tt := range tests {
		got := ticketIDPattern.FindString(tt.input)
		if got != tt.want {
			t.Errorf("FindString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
