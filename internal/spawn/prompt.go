package spawn

import (
	"fmt"
	"strings"

	"github.com/boozedog/smoovtask/internal/ticket"
)

// BuildPrompt constructs the prompt that gets fed to the AI agent worker.
// It includes ticket context and st CLI instructions.
func BuildPrompt(tk *ticket.Ticket, runID string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "You are working on ticket %s: %s\n\n", tk.ID, tk.Title)

	if tk.Body != "" {
		b.WriteString("## Ticket Context\n\n")
		b.WriteString(tk.Body)
		if !strings.HasSuffix(tk.Body, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Instructions\n\n")
	fmt.Fprintf(&b, "FIRST: Run `st pick %s --run-id %s` to claim the ticket before doing anything else.\n\n", tk.ID, runID)
	b.WriteString("- Work in the current directory (a git worktree of the main repo)\n")
	b.WriteString("- Make your changes and commit them with simple, short commit messages\n")
	b.WriteString("- Keep commits simple: `git add <files> && git commit -m \"short message\"` — no heredocs, no co-authors, no elaborate formatting\n")
	fmt.Fprintf(&b, "- Use `st note --ticket %s --run-id %s \"message\"` to log progress frequently\n", tk.ID, runID)
	fmt.Fprintf(&b, "- When done, run `st status review --ticket %s --run-id %s` to submit for review, then run /exit to end the session\n", tk.ID, runID)
	fmt.Fprintf(&b, "- If you get stuck, run `st status blocked --ticket %s --run-id %s` and add a note explaining why\n", tk.ID, runID)
	b.WriteString("- Do not push to remote. Do not create PRs. Just commit locally.\n")
	b.WriteString("- Do not amend commits or rebase. Create new commits for each logical change.\n")

	return b.String()
}
