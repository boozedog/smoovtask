package hook

import (
	"fmt"
	"regexp"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/ticket"
)

var ticketIDPattern = regexp.MustCompile(`st_[a-zA-Z0-9]{6}`)

// HandleSubagentStart processes the SubagentStart hook.
// It parses the task prompt for a ticket ID and injects ticket metadata.
// Workflow directives are intentionally omitted — the parent agent's task
// prompt is the subagent's primary directive.
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

	projectsDir, err := cfg.ProjectsDir()
	if err != nil {
		return Output{}, fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(projectsDir)
	tk, err := store.Get(ticketID)
	if err != nil {
		// Ticket not found — don't fail, just skip context injection
		return Output{}, nil
	}

	ctx := fmt.Sprintf(
		"smoovtask ticket context: %s — %s (project: %s, priority: %s, status: %s)\n"+
			"Log progress: write your note to `%s-note.md` in the current directory using the Write tool, then run `st note --file %s-note.md --ticket %s --run-id <your-run-id>` (the file is deleted after reading)",
		tk.ID, tk.Title, tk.Project, tk.Priority, tk.Status, tk.ID, tk.ID, tk.ID,
	)

	return Output{AdditionalContext: ctx}, nil
}
