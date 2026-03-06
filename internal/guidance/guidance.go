// Package guidance provides shared workflow instruction text
// used by CLI commands and hooks.
package guidance

// CommitRules returns the standard instruction forbidding attribution lines in commit messages.
func CommitRules() string {
	return "NEVER include Co-Authored-By, Signed-off-by, or any other attribution/trailer lines in commit messages. " +
		"ALWAYS disable GPG signing on commits — the human will sign when preparing the PR. " +
		"Keep commits simple: `git add <files> && git -c commit.gpgsign=false commit -m \"short message\"` — no heredocs, no trailers, no elaborate formatting."
}

// NoteHowTo returns the standard instruction for how to add notes.
func NoteHowTo() string {
	return "To add a note: use the Write tool to write your note content to `<ticket-id>-note.md` in the current directory, " +
		"then run `st note --file <ticket-id>-note.md --ticket <ticket-id> --run-id <run-id>` (the file is deleted after reading)."
}

// LoggingImplementation returns the full note guidance for implementation work
// (st pick, before-you-start context).
func LoggingImplementation() string {
	return "\n--- Notes Are Required ---\n" +
		"You MUST add notes to this ticket throughout your work — not just at the end.\n" +
		"`st status review` will REJECT your transition if you haven't added any notes.\n\n" +
		"Add a note IMMEDIATELY after each of these:\n" +
		"- The user answers a question or clarifies requirements\n" +
		"- You make a key design decision or choose between approaches\n" +
		"- You finish a meaningful chunk of implementation\n" +
		"- You hit a blocker or discover something surprising\n" +
		"- You notice improvements outside the ticket's scope\n\n" +
		"Use markdown formatting (headers, bullet lists, **bold**, `code`).\n" +
		"Notes are the ticket's audit trail — another agent must be able to\n" +
		"understand what happened and why.\n" +
		NoteHowTo() + "\n"
}

// LoggingReview returns the full note guidance for review work
// (st review, review context).
func LoggingReview() string {
	return "\n--- Notes Are Required ---\n" +
		"You MUST add notes to this ticket throughout your review — not just at the end.\n" +
		"`st status done/human-review/rework` will REJECT your transition without notes.\n\n" +
		"Add a note IMMEDIATELY after each of these:\n" +
		"- You finish reviewing a file or component\n" +
		"- You find an issue, concern, or something noteworthy\n" +
		"- The user answers a question or clarifies intent\n" +
		"- You make a judgment call about severity or scope\n\n" +
		"Use markdown formatting (headers, bullet lists, **bold**, `code`).\n" +
		"Notes are the ticket's audit trail — another agent must be able to\n" +
		"understand what was reviewed and what was found.\n" +
		NoteHowTo() + "\n"
}

// PromptReminder returns a short note reminder for injection after user messages.
func PromptReminder() string {
	return "Reminder: if the user just answered a question, clarified requirements, or made a decision, " +
		"add a note to the ticket capturing that decision BEFORE continuing with implementation. " +
		NoteHowTo()
}
