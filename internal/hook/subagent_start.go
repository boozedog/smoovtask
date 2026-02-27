package hook

import (
	"fmt"
	"regexp"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/ticket"
)

var ticketIDPattern = regexp.MustCompile(`st_[a-zA-Z0-9]{6}`)

// HandleSubagentStart processes the SubagentStart hook.
// It parses the task prompt for a ticket ID and injects ticket context.
func HandleSubagentStart(input *Input) (Output, error) {
	// Try to extract a ticket ID from the task prompt
	ticketID := ticketIDPattern.FindString(input.TaskPrompt)
	if ticketID == "" {
		return Output{}, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return Output{}, fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return Output{}, fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	tk, err := store.Get(ticketID)
	if err != nil {
		// Ticket not found — don't fail, just skip context injection
		return Output{}, nil
	}

	var workflow string
	switch tk.Status {
	case ticket.StatusReview:
		workflow = fmt.Sprintf(
			"smoovtask ticket assigned for REVIEW: %s — %s (project: %s, priority: %s)\n\n"+
				"REQUIRED workflow — you MUST follow these steps:\n"+
				"1. `st review --ticket %s --run-id <your-run-id>` — claim the ticket for review\n"+
				"2. `st note --ticket %s --run-id <your-run-id> \"<findings>\"` — document your review findings\n"+
				"3. `st status --ticket %s --run-id <your-run-id> done` (approve) or `st status --ticket %s --run-id <your-run-id> rework` (reject)\n\n"+
				"ALWAYS pass --ticket and --run-id to st commands. Do NOT approve or reject without documenting findings via `st note` first.\n\n"+
				"LOG FREQUENTLY: Use `st note` throughout your work — not just at the end. Log key decisions, discussions with the user (clarifications, scope changes, approvals), and anything surprising. Include brief code snippets where they help explain a change. Notes are the ticket's audit trail.",
			tk.ID, tk.Title, tk.Project, tk.Priority, tk.ID, tk.ID, tk.ID, tk.ID,
		)
	default:
		workflow = fmt.Sprintf(
			"smoovtask ticket assigned: %s — %s (project: %s, priority: %s)\n\n"+
				"REQUIRED workflow — you MUST follow these steps:\n"+
				"1. `st pick %s --run-id <your-run-id>` — claim the ticket before starting ANY work\n"+
				"2. `st note --ticket %s --run-id <your-run-id> \"message\"` — document progress as you work\n"+
				"3. `st status --ticket %s --run-id <your-run-id> review` — submit when done\n\n"+
				"ALWAYS pass --ticket and --run-id to st commands. Do NOT start editing code without running `st pick` first.\n\n"+
				"LOG FREQUENTLY: Use `st note` throughout your work — not just at the end. Log key decisions, discussions with the user (clarifications, scope changes, approvals), and anything surprising. Include brief code snippets where they help explain a change. Notes are the ticket's audit trail.",
			tk.ID, tk.Title, tk.Project, tk.Priority, tk.ID, tk.ID, tk.ID,
		)
	}

	context := workflow

	return Output{AdditionalContext: context}, nil
}
