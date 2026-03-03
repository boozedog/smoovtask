package hook

import (
	"fmt"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// writingTools is the set of tools that modify files.
var writingTools = map[string]bool{
	"Edit":         true,
	"MultiEdit":    true,
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

	ticketID := lookupActiveTicket(cfg, proj, input.SessionID)

	data := map[string]any{
		"tool": input.ToolName,
	}
	if input.ToolName == "Bash" {
		if cmd, ok := input.ToolInput["command"]; ok {
			data["command"] = cmd
		}
	}

	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.HookPreTool,
		Ticket:  ticketID,
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
		Data:    data,
	})

	// Hard-block writing tools when no active ticket is assigned to the run.
	if writingTools[input.ToolName] && ticketID == "" && proj != "" {
		msg := missingTicketWriteBlockMessage(input.SessionID)
		return Output{
			AdditionalContext: msg,
			Decision: &Decision{
				Behavior: "deny",
				Reason:   msg,
			},
		}, nil
	}

	return Output{}, nil
}

func missingTicketWriteBlockMessage(runID string) string {
	if runID == "" {
		return "BLOCKED: write/edit tools require an active smoovtask ticket assigned to this run. " +
			"Run `st list` and then `st pick <ticket-id>` before retrying."
	}

	return fmt.Sprintf(
		"BLOCKED: write/edit tools require an active smoovtask ticket in IN-PROGRESS or REWORK assigned to this run. Run `st pick <ticket-id> --run-id %s` and retry.",
		runID,
	)
}

// activeTicketID returns the ticket ID assigned to sessionID, or "" if none.
func activeTicketID(store *ticket.Store, proj, sessionID string) string {
	tickets, err := store.List(ticket.ListFilter{Project: proj})
	if err != nil {
		return ""
	}
	for _, tk := range tickets {
		if tk.Assignee == sessionID &&
			(tk.Status == ticket.StatusInProgress || tk.Status == ticket.StatusRework) {
			return tk.ID
		}
	}
	return ""
}

// lookupActiveTicket resolves the active ticket for a session.
// Returns "" if config, project, or session ID are missing, or no ticket is assigned.
func lookupActiveTicket(cfg *config.Config, proj, sessionID string) string {
	if proj == "" || sessionID == "" {
		return ""
	}
	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return ""
	}
	return activeTicketID(ticket.NewStore(ticketsDir), proj, sessionID)
}
