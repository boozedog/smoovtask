// Package guidance provides shared workflow instruction text
// used by CLI commands and hooks.
package guidance

// NoteHowTo returns the standard instruction for how to add notes.
func NoteHowTo() string {
	return "To add a note: use the Write tool to write your note content to a file, " +
		"then run `st note --file <path> --ticket <ticket-id> --run-id <run-id>` (the file is deleted after reading)."
}

// LoggingImplementation returns the full logging guidance for implementation work
// (st pick, before-you-start context).
func LoggingImplementation() string {
	return "\n--- Logging ---\n" +
		"Log your work frequently. Use markdown formatting — headers,\n" +
		"bullet lists, **bold**, and `code` make notes easier to read. Good things to log:\n" +
		"- Key decisions and why you made them\n" +
		"- Discussions with the user — especially clarifications, scope changes, or approvals\n" +
		"- Design trade-offs considered and the chosen approach\n" +
		"- Significant progress milestones or blockers encountered\n" +
		"- Brief code snippets where they help explain a change or decision\n" +
		"- Improvements you notice that are outside the ticket's scope\n" +
		"Notes become the ticket's audit trail — another agent should be able to understand what happened.\n" +
		NoteHowTo() + "\n"
}

// LoggingReview returns the full logging guidance for review work
// (st review, review context).
func LoggingReview() string {
	return "\n--- Logging ---\n" +
		"Log your work frequently. Use markdown formatting — headers,\n" +
		"bullet lists, **bold**, and `code` make notes easier to read. Good things to log:\n" +
		"- Key decisions and why you made them\n" +
		"- Discussions with the user — especially clarifications, scope changes, or approvals\n" +
		"- Anything surprising or noteworthy discovered during review\n" +
		"- Brief code snippets where they help explain a finding or concern\n" +
		"Notes become the ticket's audit trail — another agent should be able to understand what happened.\n" +
		NoteHowTo() + "\n"
}

// CompactImplementation returns the compact logging guidance for hooks
// injecting context into implementation work sessions.
func CompactImplementation() string {
	return "LOG FREQUENTLY: Use `st note` throughout your work — not just at the end. " +
		"Use markdown formatting (headers, bullet lists, code blocks). " +
		"Log key decisions, discussions with the user (clarifications, scope changes, approvals), " +
		"anything surprising, and improvements you notice that are outside the ticket's scope. " +
		"Include brief code snippets where they help explain a change. Notes are the ticket's audit trail. " +
		NoteHowTo()
}

// CompactReview returns the compact logging guidance for hooks
// injecting context into review work sessions.
func CompactReview() string {
	return "LOG FREQUENTLY: Use `st note` throughout your work — not just at the end. " +
		"Use markdown formatting (headers, bullet lists, code blocks). " +
		"Log key decisions, discussions with the user (clarifications, scope changes, approvals), " +
		"and anything surprising. Include brief code snippets where they help explain a change. " +
		"Notes are the ticket's audit trail. " +
		NoteHowTo()
}
