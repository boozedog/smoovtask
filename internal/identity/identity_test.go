package identity

import (
	"os"
	"testing"
)

func TestSessionID(t *testing.T) {
	orig := os.Getenv("CLAUDE_SESSION_ID")
	defer os.Setenv("CLAUDE_SESSION_ID", orig)
	defer func() { sessionOverride = "" }()

	os.Setenv("CLAUDE_SESSION_ID", "test-session-123")
	if got := SessionID(); got != "test-session-123" {
		t.Errorf("SessionID() = %q, want %q", got, "test-session-123")
	}

	os.Setenv("CLAUDE_SESSION_ID", "")
	if got := SessionID(); got != "" {
		t.Errorf("SessionID() = %q, want empty", got)
	}
}

func TestSessionIDOverride(t *testing.T) {
	orig := os.Getenv("CLAUDE_SESSION_ID")
	defer os.Setenv("CLAUDE_SESSION_ID", orig)
	defer func() { sessionOverride = "" }()

	os.Setenv("CLAUDE_SESSION_ID", "env-session")
	SetSessionID("override-session")
	if got := SessionID(); got != "override-session" {
		t.Errorf("SessionID() = %q, want %q", got, "override-session")
	}

	// Clear override, falls back to env
	SetSessionID("")
	if got := SessionID(); got != "env-session" {
		t.Errorf("SessionID() = %q, want %q", got, "env-session")
	}
}

func TestActor(t *testing.T) {
	origSession := os.Getenv("CLAUDE_SESSION_ID")
	origClaude := os.Getenv("CLAUDECODE")
	defer os.Setenv("CLAUDE_SESSION_ID", origSession)
	defer os.Setenv("CLAUDECODE", origClaude)
	defer func() { sessionOverride = "" }()

	// Session ID set → agent
	os.Setenv("CLAUDE_SESSION_ID", "sess-123")
	os.Setenv("CLAUDECODE", "")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() with session = %q, want %q", got, "agent")
	}

	// CLAUDECODE=1 without session → agent
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
	SetSessionID("flag-session")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() with override = %q, want %q", got, "agent")
	}
}
