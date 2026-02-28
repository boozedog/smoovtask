package identity

import "testing"

func TestRunID(t *testing.T) {
	defer func() {
		runIDOverride = ""
		humanOverride = false
	}()

	if got := RunID(); got != "" {
		t.Errorf("RunID() = %q, want empty", got)
	}

	SetRunID("test-session-123")
	if got := RunID(); got != "test-session-123" {
		t.Errorf("RunID() = %q, want %q", got, "test-session-123")
	}
}

func TestSetHuman(t *testing.T) {
	defer func() {
		runIDOverride = ""
		humanOverride = false
	}()

	if got := Actor(); got != "agent" {
		t.Errorf("Actor() default = %q, want %q", got, "agent")
	}

	SetHuman(true)
	if got := Actor(); got != "human" {
		t.Errorf("Actor() with human override = %q, want %q", got, "human")
	}
}

func TestActor(t *testing.T) {
	defer func() {
		runIDOverride = ""
		humanOverride = false
	}()

	// Default is agent.
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() default = %q, want %q", got, "agent")
	}

	// Run ID still maps to agent.
	SetRunID("flag-session")
	if got := Actor(); got != "agent" {
		t.Errorf("Actor() with run-id = %q, want %q", got, "agent")
	}

	// Human override wins.
	SetHuman(true)
	if got := Actor(); got != "human" {
		t.Errorf("Actor() with human override = %q, want %q", got, "human")
	}
}
