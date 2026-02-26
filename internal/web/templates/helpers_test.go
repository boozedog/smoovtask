package templates

import (
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/ticket"
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

func TestBackPartialURL(t *testing.T) {
	tests := []struct {
		name    string
		backURL string
		want    string
	}{
		{"list returns partials/list", "/list", "/partials/list"},
		{"board returns partials/board", "/", "/partials/board"},
		{"other returns partials/board", "/something", "/partials/board"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backPartialURL(tt.backURL)
			if got != tt.want {
				t.Errorf("backPartialURL(%q) = %q, want %q", tt.backURL, got, tt.want)
			}
		})
	}
}

func TestTicketPartialURL(t *testing.T) {
	tests := []struct {
		name string
		data TicketData
		want string
	}{
		{
			"from board",
			TicketData{
				Ticket:  &ticket.Ticket{ID: "st_abc123"},
				BackURL: "/",
			},
			"/partials/ticket/st_abc123",
		},
		{
			"from list adds query param",
			TicketData{
				Ticket:  &ticket.Ticket{ID: "st_abc123"},
				BackURL: "/list",
			},
			"/partials/ticket/st_abc123?from=list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ticketPartialURL(tt.data)
			if got != tt.want {
				t.Errorf("ticketPartialURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
