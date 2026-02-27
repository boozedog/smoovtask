package templates

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", time.Now().Add(-10 * time.Second), "just now"},
		{"1 minute ago", time.Now().Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", time.Now().Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", time.Now().Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", time.Now().Add(-3 * time.Hour), "3 hours ago"},
		{"yesterday", time.Now().Add(-30 * time.Hour), "yesterday"},
		{"5 days ago", time.Now().Add(-5 * 24 * time.Hour), "5 days ago"},
		{"old date falls back to format", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "2025-01-01 00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(tt.t)
			if got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTicketPartialURL(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"basic", "st_abc123", "/partials/ticket/st_abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ticketPartialURL(tt.id)
			if got != tt.want {
				t.Errorf("ticketPartialURL(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
