package identity

import (
	"os"
	"testing"
)

func TestSessionID(t *testing.T) {
	orig := os.Getenv("CLAUDE_SESSION_ID")
	defer os.Setenv("CLAUDE_SESSION_ID", orig)

	os.Setenv("CLAUDE_SESSION_ID", "test-session-123")
	if got := SessionID(); got != "test-session-123" {
		t.Errorf("SessionID() = %q, want %q", got, "test-session-123")
	}

	os.Setenv("CLAUDE_SESSION_ID", "")
	if got := SessionID(); got != "" {
		t.Errorf("SessionID() = %q, want empty", got)
	}
}

func TestActor(t *testing.T) {
	orig := os.Getenv("CLAUDE_SESSION_ID")
	defer os.Setenv("CLAUDE_SESSION_ID", orig)

	os.Setenv("CLAUDE_SESSION_ID", "sess-123")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() = %q, want %q", got, "agent")
	}

	os.Setenv("CLAUDE_SESSION_ID", "")
	if got := Actor(); got != "human" {
		t.Errorf("Actor() = %q, want %q", got, "human")
	}
}
