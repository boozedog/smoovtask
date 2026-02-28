package hook

import (
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// writingTools is the set of tools that modify files.
var writingTools = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"NotebookEdit": true,
}

// HandlePreTool logs a pre-tool event and warns if a writing tool is used
// without an active ticket.
func HandlePreTool(input *Input) (Output, error) {
	cfg, err := config.Load()
	if err != nil {
		return Output{}, nil // Don't fail on config errors for async hooks
	}

	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return Output{}, nil
	}

	proj := project.Detect(cfg, input.CWD)

	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPreTool,
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
		Data: map[string]any{
			"tool": input.ToolName,
		},
	})

	// Warn if a writing tool is used with no active ticket
	if writingTools[input.ToolName] && proj != "" && input.SessionID != "" {
		ticketsDir, err := cfg.TicketsDir()
		if err != nil {
			return Output{}, nil
		}

		store := ticket.NewStore(ticketsDir)
		if !hasActiveTicket(store, proj, input.SessionID) {
			return Output{
				AdditionalContext: "WARNING: You are editing code without an active smoovtask ticket. " +
					"Run `st pick st_xxxxxx` to claim a ticket first. " +
					"Unattributed work creates audit gaps.",
			}, nil
		}
	}

	return Output{}, nil
}

// hasActiveTicket checks if the session has an IN-PROGRESS or REWORK ticket.
func hasActiveTicket(store *ticket.Store, proj, sessionID string) bool {
	tickets, err := store.List(ticket.ListFilter{Project: proj})
	if err != nil {
		return false
	}
	for _, tk := range tickets {
		if tk.Assignee == sessionID &&
			(tk.Status == ticket.StatusInProgress || tk.Status == ticket.StatusRework) {
			return true
		}
	}
	return false
}
