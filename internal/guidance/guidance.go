// Package guidance provides shared workflow instruction text
// used by CLI commands and hooks.
package guidance

// CommitStyle returns the universal commit message style rules (no trailers, no elaborate formatting).
// Safe to inject in any context — does not mention GPG signing.
func CommitStyle() string {
	return "NEVER include Co-Authored-By, Signed-off-by, or any other attribution/trailer lines in commit messages. " +
		"Keep commits simple: `git add <files> && git commit -m \"short message\"` — no heredocs, no trailers, no elaborate formatting."
}

// CommitRules returns the full commit instruction for ticket worktree work.
// Includes CommitStyle() plus the GPG signing skip — only inject when agents are
// committing in ticket worktrees, NOT PR worktrees.
func CommitRules() string {
	return CommitStyle() + " " +
		"ALWAYS disable GPG signing on commits — the human will sign when preparing the PR. " +
		"Use `git -c commit.gpgsign=false commit` for every commit."
}

// PRCommitRules returns commit guidance for PR worktree merges.
// PR worktree commits are signed by the human, so agents must NEVER skip GPG signing
// and must prompt the user before each commit so they can authorize the signature.
func PRCommitRules() string {
	return "NEVER use `-c commit.gpgsign=false` or `--no-gpg-sign` in the PR worktree — " +
		"the human signs these commits. Before EACH `git commit`, prompt the user with " +
		"\"Ready to commit — please prepare to sign\" so they can authorize the GPG signature. " +
		"If signing times out, ask the user to retry rather than disabling signing."
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
