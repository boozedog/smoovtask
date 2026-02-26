package handler

import (
	"net/http/httptest"
	"testing"
)

func TestBackNav(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		referer  string
		wantURL  string
		wantName string
	}{
		{
			"from query param",
			"/ticket/st_abc123?from=list",
			"",
			"/list", "List",
		},
		{
			"from referer /list",
			"/ticket/st_abc123",
			"http://localhost:9090/list",
			"/list", "List",
		},
		{
			"from referer /partials/list",
			"/ticket/st_abc123",
			"http://localhost:9090/partials/list",
			"/list", "List",
		},
		{
			"query param takes priority over referer",
			"/ticket/st_abc123?from=list",
			"http://localhost:9090/",
			"/list", "List",
		},
		{
			"defaults to board with no referer",
			"/ticket/st_abc123",
			"",
			"/", "Board",
		},
		{
			"defaults to board with non-list referer",
			"/ticket/st_abc123",
			"http://localhost:9090/activity",
			"/", "Board",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}

			gotURL, gotName := backNav(req)
			if gotURL != tt.wantURL {
				t.Errorf("backNav() URL = %q, want %q", gotURL, tt.wantURL)
			}
			if gotName != tt.wantName {
				t.Errorf("backNav() name = %q, want %q", gotName, tt.wantName)
			}
		})
	}
}
