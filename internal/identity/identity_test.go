package identity

import (
	"os"
	"testing"
)

func TestRunID(t *testing.T) {
	orig := os.Getenv("CLAUDE_SESSION_ID")
	defer os.Setenv("CLAUDE_SESSION_ID", orig)
	defer func() { runIDOverride = "" }()

	os.Setenv("CLAUDE_SESSION_ID", "test-session-123")
	if got := RunID(); got != "test-session-123" {
		t.Errorf("RunID() = %q, want %q", got, "test-session-123")
	}

	os.Setenv("CLAUDE_SESSION_ID", "")
	if got := RunID(); got != "" {
		t.Errorf("RunID() = %q, want empty", got)
	}
}

func TestRunIDOverride(t *testing.T) {
	orig := os.Getenv("CLAUDE_SESSION_ID")
	defer os.Setenv("CLAUDE_SESSION_ID", orig)
	defer func() { runIDOverride = "" }()

	os.Setenv("CLAUDE_SESSION_ID", "env-session")
	SetRunID("override-session")
	if got := RunID(); got != "override-session" {
		t.Errorf("RunID() = %q, want %q", got, "override-session")
	}

	// Clear override, falls back to env
	SetRunID("")
	if got := RunID(); got != "env-session" {
		t.Errorf("RunID() = %q, want %q", got, "env-session")
	}
}

func TestActor(t *testing.T) {
	origSession := os.Getenv("CLAUDE_SESSION_ID")
	origClaude := os.Getenv("CLAUDECODE")
	defer os.Setenv("CLAUDE_SESSION_ID", origSession)
	defer os.Setenv("CLAUDECODE", origClaude)
	defer func() { runIDOverride = "" }()

	// Run ID set → agent
	os.Setenv("CLAUDE_SESSION_ID", "sess-123")
	os.Setenv("CLAUDECODE", "")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() with run ID = %q, want %q", got, "agent")
	}

	// CLAUDECODE=1 without run ID → agent
	os.Setenv("CLAUDE_SESSION_ID", "")
	os.Setenv("CLAUDECODE", "1")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() with CLAUDECODE=1 = %q, want %q", got, "agent")
	}

	// Neither → human
	os.Setenv("CLAUDE_SESSION_ID", "")
	os.Setenv("CLAUDECODE", "")
	if got := Actor(); got != "human" {
		t.Errorf("Actor() = %q, want %q", got, "human")
	}

	// Override → agent
	SetRunID("flag-session")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() with override = %q, want %q", got, "agent")
	}
}
