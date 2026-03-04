// Package guidance provides shared workflow instruction text
// used by CLI commands and hooks.
package guidance

import (
	"fmt"
	"path/filepath"
)

// NotesDir returns the absolute path to the notes directory under a repo root.
func NotesDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".st", "notes")
}

// NoteHowTo returns the standard instruction for how to add notes using the file-based drop approach.
// notesDir is the absolute path to the notes directory (e.g., /path/to/repo/.st/notes).
func NoteHowTo(notesDir string) string {
	return fmt.Sprintf("To add a note: use the Write tool to write your note content to `%s/<run-id>.md`, "+
		"then run `st note --run-id <run-id>` (no message argument needed — it reads from the file).", notesDir)
}

// LoggingImplementation returns the full logging guidance for implementation work
// (st pick, before-you-start context).
func LoggingImplementation(notesDir string) string {
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
		NoteHowTo(notesDir) + "\n"
}

// LoggingReview returns the full logging guidance for review work
// (st review, review context).
func LoggingReview(notesDir string) string {
	return "\n--- Logging ---\n" +
		"Log your work frequently. Use markdown formatting — headers,\n" +
		"bullet lists, **bold**, and `code` make notes easier to read. Good things to log:\n" +
		"- Key decisions and why you made them\n" +
		"- Discussions with the user — especially clarifications, scope changes, or approvals\n" +
		"- Anything surprising or noteworthy discovered during review\n" +
		"- Brief code snippets where they help explain a finding or concern\n" +
		"Notes become the ticket's audit trail — another agent should be able to understand what happened.\n" +
		NoteHowTo(notesDir) + "\n"
}

// CompactImplementation returns the compact logging guidance for hooks
// injecting context into implementation work sessions.
func CompactImplementation(notesDir string) string {
	return "LOG FREQUENTLY: Use `st note` throughout your work — not just at the end. " +
		"Use markdown formatting (headers, bullet lists, code blocks). " +
		"Log key decisions, discussions with the user (clarifications, scope changes, approvals), " +
		"anything surprising, and improvements you notice that are outside the ticket's scope. " +
		"Include brief code snippets where they help explain a change. Notes are the ticket's audit trail. " +
		NoteHowTo(notesDir)
}

// CompactReview returns the compact logging guidance for hooks
// injecting context into review work sessions.
func CompactReview(notesDir string) string {
	return "LOG FREQUENTLY: Use `st note` throughout your work — not just at the end. " +
		"Use markdown formatting (headers, bullet lists, code blocks). " +
		"Log key decisions, discussions with the user (clarifications, scope changes, approvals), " +
		"and anything surprising. Include brief code snippets where they help explain a change. " +
		"Notes are the ticket's audit trail. " +
		NoteHowTo(notesDir)
}
