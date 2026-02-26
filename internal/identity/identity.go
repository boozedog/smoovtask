package identity

import "os"

// sessionOverride is set via SetSessionID when --session flag is provided.
var sessionOverride string

// SetSessionID sets an explicit session ID override (from --session flag).
func SetSessionID(id string) {
	sessionOverride = id
}

// SessionID returns the current session ID.
// Priority: explicit override > CLAUDE_SESSION_ID env var.
func SessionID() string {
	if sessionOverride != "" {
		return sessionOverride
	}
	return os.Getenv("CLAUDE_SESSION_ID")
}

// Actor returns the actor identifier for the current context.
// Returns "agent" if a session ID is set or CLAUDECODE=1; otherwise "human".
func Actor() string {
	if SessionID() != "" || os.Getenv("CLAUDECODE") == "1" {
		return "agent"
	}
	return "human"
}
