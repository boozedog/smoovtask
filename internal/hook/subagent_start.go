package hook

import (
	"fmt"
	"regexp"

	"github.com/boozedog/smoovbrain/internal/config"
	"github.com/boozedog/smoovbrain/internal/ticket"
)

var ticketIDPattern = regexp.MustCompile(`sb_[a-zA-Z0-9]{6}`)

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
		"smoovbrain ticket assigned: %s — %s (project: %s, priority: %s)\n\n"+
			"Use `sb pick %s` to claim the ticket before starting work.\n"+
			"Use `sb note \"message\"` to document progress.\n"+
			"Use `sb status review` when done.",
		tk.ID, tk.Title, tk.Project, tk.Priority, tk.ID,
	)

	return Output{AdditionalContext: context}, nil
}
