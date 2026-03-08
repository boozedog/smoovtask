package spawn

import (
	"fmt"
	"syscall"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
)

// WorkerState represents the current state of a spawned worker.
type WorkerState string

const (
	WorkerRunning   WorkerState = "running"
	WorkerCompleted WorkerState = "completed"
	WorkerFailed    WorkerState = "failed"
	WorkerTimeout   WorkerState = "timeout"
	WorkerStale     WorkerState = "stale" // PID no longer exists but no completion event
)

// WorkerInfo contains information about a spawned worker for a ticket.
type WorkerInfo struct {
	TicketID string
	PID      int
	State    WorkerState
	Branch   string
	Worktree string
	RunID    string
	Started  time.Time
	Elapsed  time.Duration
}

// GetWorkerInfo returns info about the most recent worker for a ticket.
// Returns nil if no spawn events exist for this ticket.
func GetWorkerInfo(eventsDir, ticketID string) (*WorkerInfo, error) {
	events, err := event.QueryEvents(eventsDir, event.Query{TicketID: ticketID})
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	return workerInfoFromEvents(ticketID, events), nil
}

// BatchGetWorkerInfo returns worker info for all tickets that have spawn events.
// It scans event files once instead of per-ticket, returning a map of ticket ID to WorkerInfo.
func BatchGetWorkerInfo(eventsDir string) (map[string]*WorkerInfo, error) {
	events, err := event.QueryEvents(eventsDir, event.Query{})
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	// Group spawn events by ticket ID
	byTicket := make(map[string][]event.Event)
	for _, e := range events {
		if e.Ticket == "" {
			continue
		}
		switch e.Event {
		case SpawnStarted, SpawnCompleted, SpawnFailed, SpawnTimeout:
			byTicket[e.Ticket] = append(byTicket[e.Ticket], e)
		}
	}

	result := make(map[string]*WorkerInfo, len(byTicket))
	for ticketID, ticketEvents := range byTicket {
		if info := workerInfoFromEvents(ticketID, ticketEvents); info != nil {
			result[ticketID] = info
		}
	}

	return result, nil
}

// workerInfoFromEvents derives WorkerInfo from a set of events for a single ticket.
func workerInfoFromEvents(ticketID string, events []event.Event) *WorkerInfo {
	// Find the most recent spawn.started event for this ticket
	var lastStart *event.Event
	var lastEnd *event.Event
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Event {
		case SpawnStarted:
			if lastStart == nil {
				lastStart = &events[i]
			}
		case SpawnCompleted, SpawnFailed, SpawnTimeout:
			if lastEnd == nil {
				lastEnd = &events[i]
			}
		}
	}

	if lastStart == nil {
		return nil
	}

	// If there's a terminal event after the start, use that
	if lastEnd != nil && lastEnd.TS.After(lastStart.TS) {
		state := WorkerCompleted
		switch lastEnd.Event {
		case SpawnFailed:
			state = WorkerFailed
		case SpawnTimeout:
			state = WorkerTimeout
		}
		elapsed, _ := time.ParseDuration(stringFromData(lastEnd.Data, "elapsed"))
		return &WorkerInfo{
			TicketID: ticketID,
			PID:      intFromData(lastStart.Data, "pid"),
			State:    state,
			Branch:   stringFromData(lastStart.Data, "branch"),
			Worktree: stringFromData(lastStart.Data, "worktree"),
			RunID:    lastStart.RunID,
			Started:  lastStart.TS,
			Elapsed:  elapsed,
		}
	}

	// No terminal event — check if PID is still alive
	pid := intFromData(lastStart.Data, "pid")
	info := &WorkerInfo{
		TicketID: ticketID,
		PID:      pid,
		Branch:   stringFromData(lastStart.Data, "branch"),
		Worktree: stringFromData(lastStart.Data, "worktree"),
		RunID:    lastStart.RunID,
		Started:  lastStart.TS,
		Elapsed:  time.Since(lastStart.TS),
	}

	if pid > 0 && isProcessAlive(pid) {
		info.State = WorkerRunning
	} else {
		info.State = WorkerStale
	}

	return info
}

// isProcessAlive checks if a process with the given PID exists.
// Returns false for invalid PIDs (0 or negative) to prevent signaling
// the caller's process group via kill(0, 0).
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func stringFromData(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intFromData(data map[string]any, key string) int {
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}
