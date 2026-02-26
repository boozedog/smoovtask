package project

import (
	"testing"

	"github.com/boozedog/smoovbrain/internal/config"
)

func TestDetect(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{
			"api-server":  {Path: "/home/user/projects/api-server"},
			"smoovbrain":  {Path: "/home/user/projects/smoovbrain"},
			"nested-proj": {Path: "/home/user/projects/api-server/services/auth"},
		},
	}

	tests := []struct {
		dir  string
		want string
	}{
		{"/home/user/projects/api-server", "api-server"},
		{"/home/user/projects/api-server/internal", "api-server"},
		{"/home/user/projects/smoovbrain", "smoovbrain"},
		{"/home/user/projects/smoovbrain/cmd", "smoovbrain"},
		{"/home/user/projects/unknown", ""},
		{"/other/path", ""},
		// Longest prefix match: nested project wins
		{"/home/user/projects/api-server/services/auth", "nested-proj"},
		{"/home/user/projects/api-server/services/auth/handler", "nested-proj"},
	}

	for _, tt := range tests {
		got := Detect(cfg, tt.dir)
		if got != tt.want {
			t.Errorf("Detect(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestDetectEmpty(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]config.ProjectConfig{},
	}

	got := Detect(cfg, "/some/path")
	if got != "" {
		t.Errorf("Detect with no projects = %q, want empty", got)
	}
}
