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

	context := fmt.Sprintf(
		"smoovtask ticket assigned: %s — %s (project: %s, priority: %s)\n\n"+
			"REQUIRED workflow — you MUST follow these steps:\n"+
			"1. `st pick %s` — claim the ticket before starting ANY work\n"+
			"2. `st note \"message\"` — document progress as you work\n"+
			"3. `st status review` — submit when done\n\n"+
			"Do NOT start editing code without running `st pick` first.",
		tk.ID, tk.Title, tk.Project, tk.Priority, tk.ID,
	)

	return Output{AdditionalContext: context}, nil
}
