package identity

import "os"

// runIDOverride is set via SetRunID when --run-id flag is provided.
var runIDOverride string

// SetRunID sets an explicit run ID override (from --run-id flag).
func SetRunID(id string) {
	runIDOverride = id
}

// RunID returns the current run ID.
// Priority: explicit override > CLAUDE_SESSION_ID env var.
func RunID() string {
	if runIDOverride != "" {
		return runIDOverride
	}
	return os.Getenv("CLAUDE_SESSION_ID")
}

// Actor returns the actor identifier for the current context.
// Returns "agent" if a run ID is set or CLAUDECODE=1; otherwise "human".
func Actor() string {
	if RunID() != "" || os.Getenv("CLAUDECODE") == "1" {
		return "agent"
	}
	return "human"
}
