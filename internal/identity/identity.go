package identity

// runIDOverride is set via SetRunID when --run-id flag is provided.
var runIDOverride string

// humanOverride is set via SetHuman when --human flag is provided.
var humanOverride bool

// SetRunID sets an explicit run ID override (from --run-id flag).
func SetRunID(id string) {
	runIDOverride = id
}

// SetHuman marks the current invocation as human-driven.
func SetHuman(human bool) {
	humanOverride = human
}

// RunID returns the current run ID.
func RunID() string {
	return runIDOverride
}

// Actor returns the actor identifier for the current context.
// Default is "agent"; --human forces "human".
func Actor() string {
	if humanOverride {
		return "human"
	}
	return "agent"
}
