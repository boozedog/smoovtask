package event

import "time"

// Event type constants.
const (
	TicketCreated  = "ticket.created"
	TicketAssigned = "ticket.assigned"
	TicketNote     = "ticket.note"

	StatusBacklog    = "status.backlog"
	StatusOpen       = "status.open"
	StatusInProgress = "status.in-progress"
	StatusReview     = "status.review"
	StatusRework     = "status.rework"
	StatusDone       = "status.done"
	StatusBlocked    = "status.blocked"

	HookPreTool       = "hook.pre-tool"
	HookPostTool      = "hook.post-tool"
	HookSessionStart  = "hook.session-start"
	HookStop          = "hook.stop"
	HookSubagentStop  = "hook.subagent-stop"
	HookTaskCompleted = "hook.task-completed"
	HookTeammateIdle  = "hook.teammate-idle"
	HookPermissionReq = "hook.permission-request"
	HookSessionEnd    = "hook.session-end"
)

// Event represents a single event in the system log.
type Event struct {
	TS      time.Time      `json:"ts"`
	Event   string         `json:"event"`
	Ticket  string         `json:"ticket"`
	Project string         `json:"project"`
	Actor   string         `json:"actor"`
	RunID   string         `json:"run_id"`
	Data    map[string]any `json:"data"`
}
