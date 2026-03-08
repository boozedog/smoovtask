package hook

import (
	"fmt"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/rules"
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

	proj := detectProject(cfg, input.CWD)

	// Auto-handoff when exiting plan mode — release the ticket so the new
	// build session can pick it up.
	if input.ToolName == "ExitPlanMode" && proj != "" && input.SessionID != "" {
		if out, handled := handleExitPlanModeHandoff(cfg, proj, input, eventsDir); handled {
			return out, nil
		}
	}

	ticketID := lookupActiveTicket(cfg, proj, input.SessionID)

	data := map[string]any{
		"tool": input.ToolName,
	}
	switch input.ToolName {
	case "Bash":
		if cmd, ok := input.ToolInput["command"]; ok {
			data["command"] = cmd
		}
	case "Read", "Edit", "Write", "NotebookEdit", "MultiEdit":
		if fp, ok := input.ToolInput["file_path"]; ok {
			data["file_path"] = fp
		}
	case "Glob":
		if pat, ok := input.ToolInput["pattern"]; ok {
			data["pattern"] = pat
		}
	case "Grep":
		if pat, ok := input.ToolInput["pattern"]; ok {
			data["pattern"] = pat
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
				HookEventName: "PreToolUse",
				Behavior:      "deny",
				Reason:        msg,
			},
		}, nil
	}

	// Hard-block git commit commands that contain attribution trailers.
	if msg := rejectCommitAttribution(input); msg != "" {
		return Output{
			Decision: &Decision{
				HookEventName: "PreToolUse",
				Behavior:      "deny",
				Reason:        msg,
			},
		}, nil
	}

	// Evaluate auto-allow/deny rules.
	rulesDir, err := cfg.RulesDir()
	if err != nil {
		return Output{}, nil
	}

	result := rules.Evaluate(rulesDir, "PreToolUse", input.ToolName, input.ToolInput)
	if result == nil {
		return Output{}, nil
	}

	{
		ruleData := map[string]any{
			"tool":     input.ToolName,
			"decision": string(result.Decision),
			"ruleset":  result.Ruleset,
			"rule":     result.Rule,
			"reason":   result.Reason,
		}
		// Include the command context for display in the Rules UI.
		switch input.ToolName {
		case "Bash":
			if cmd, ok := input.ToolInput["command"].(string); ok {
				ruleData["command"] = cmd
			}
		case "Read", "Edit", "Write", "NotebookEdit", "MultiEdit":
			if fp, ok := input.ToolInput["file_path"].(string); ok {
				ruleData["file_path"] = fp
			}
		case "Glob":
			if pat, ok := input.ToolInput["pattern"].(string); ok {
				ruleData["pattern"] = pat
			}
		case "Grep":
			if pat, ok := input.ToolInput["pattern"].(string); ok {
				ruleData["pattern"] = pat
			}
		}
		_ = el.Append(event.Event{
			TS:      time.Now().UTC(),
			Event:   event.HookRuleDecision,
			Ticket:  ticketID,
			Project: proj,
			Actor:   "agent",
			RunID:   input.SessionID,
			Source:  input.Source,
			Data:    ruleData,
		})
	}

	switch result.Decision {
	case rules.ActionAllow:
		return Output{
			Decision: &Decision{HookEventName: "PreToolUse", Behavior: "allow", Reason: result.Reason},
		}, nil
	case rules.ActionDeny:
		return Output{
			Decision: &Decision{HookEventName: "PreToolUse", Behavior: "deny", Reason: result.Reason},
		}, nil
	default:
		return Output{}, nil
	}
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

// rejectCommitAttribution checks if a Bash tool call is a git commit whose
// message contains Co-Authored-By, Signed-off-by, or similar attribution
// trailers. Returns a denial reason, or "" if the command is fine.
func rejectCommitAttribution(input *Input) string {
	if input.ToolName != "Bash" {
		return ""
	}
	cmd, ok := input.ToolInput["command"].(string)
	if !ok {
		return ""
	}
	lower := strings.ToLower(cmd)
	if !strings.Contains(lower, "git commit") {
		return ""
	}
	for _, trailer := range []string{"co-authored-by", "signed-off-by"} {
		if strings.Contains(lower, trailer) {
			return "BLOCKED: commit messages must not contain Co-Authored-By, Signed-off-by, or any attribution trailers. " +
				"Remove the trailer and retry with a simple commit message."
		}
	}
	return ""
}

// handleExitPlanModeHandoff hands off the active ticket when ExitPlanMode is
// called. Returns (output, true) if a handoff was performed, or (_, false) if
// there was no active ticket to hand off (caller should continue normally).
func handleExitPlanModeHandoff(cfg *config.Config, proj string, input *Input, eventsDir string) (Output, bool) {
	projectsDir, err := cfg.ProjectsDir()
	if err != nil {
		return Output{}, false
	}
	store := ticket.NewStore(projectsDir)
	ticketID := activeTicketID(store, proj, input.SessionID)
	if ticketID == "" {
		return Output{}, false
	}

	tk, err := store.Get(ticketID)
	if err != nil {
		return Output{}, false
	}

	now := time.Now().UTC()
	previousAssignee := tk.Assignee
	oldStatus := tk.Status

	tk.Status = ticket.StatusOpen
	tk.Assignee = ""
	tk.Updated = now

	ticket.AppendSection(tk, "Handed Off", "agent", input.SessionID, "", map[string]string{
		"reason":            "plan-mode-exit",
		"previous-assignee": previousAssignee,
	}, now)

	if err := store.Save(tk); err != nil {
		return Output{}, false
	}

	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   event.TicketHandoff,
		Ticket:  tk.ID,
		Project: proj,
		Actor:   "agent",
		RunID:   input.SessionID,
		Source:  input.Source,
		Data: map[string]any{
			"from":              string(oldStatus),
			"previous_assignee": previousAssignee,
			"reason":            "plan-mode-exit",
		},
	})

	return Output{}, true
}

// detectProject resolves the vault path from config and detects the project.
func detectProject(cfg *config.Config, dir string) string {
	vaultPath, err := cfg.VaultPath()
	if err != nil {
		return ""
	}
	return project.Detect(vaultPath, dir)
}

// lookupActiveTicket resolves the active ticket for a session.
// Returns "" if config, project, or session ID are missing, or no ticket is assigned.
func lookupActiveTicket(cfg *config.Config, proj, sessionID string) string {
	if proj == "" || sessionID == "" {
		return ""
	}
	projectsDir, err := cfg.ProjectsDir()
	if err != nil {
		return ""
	}
	return activeTicketID(ticket.NewStore(projectsDir), proj, sessionID)
}
