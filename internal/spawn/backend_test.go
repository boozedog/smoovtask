package spawn

import (
	"testing"
)

func TestGetBackend(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"claude", false},
		{"", false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := GetBackend(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error for unknown backend")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if b == nil {
				t.Error("expected non-nil backend")
			}
		})
	}
}

func TestClaudeBackendName(t *testing.T) {
	b := &ClaudeBackend{}
	if b.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", b.Name(), "claude")
	}
}
