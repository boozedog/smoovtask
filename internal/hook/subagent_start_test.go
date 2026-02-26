package hook

import (
	"testing"
)

func TestTicketIDPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Work on st_a7Kx2m: add rate limiting", "st_a7Kx2m"},
		{"st_Qr9fZw is the target", "st_Qr9fZw"},
		{"no ticket here", ""},
		{"st_ too short", ""},
		{"st_!@#$%^ invalid chars", ""},
	}

	for _, tt := range tests {
		got := ticketIDPattern.FindString(tt.input)
		if got != tt.want {
			t.Errorf("FindString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
