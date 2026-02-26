package identity

import "os"

// SessionID returns the current Claude session ID from the environment.
// Returns empty string if not in a Claude session.
func SessionID() string {
	return os.Getenv("CLAUDE_SESSION_ID")
}

// Actor returns the actor identifier for the current context.
// If in a Claude session, returns "agent"; otherwise "human".
func Actor() string {
	if SessionID() != "" {
		return "agent"
	}
	return "human"
}
