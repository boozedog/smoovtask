package ticket

import "time"

// Status represents the workflow status of a ticket.
type Status string

const (
	StatusBacklog    Status = "BACKLOG"
	StatusOpen       Status = "OPEN"
	StatusInProgress Status = "IN-PROGRESS"
	StatusReview     Status = "REVIEW"
	StatusRework     Status = "REWORK"
	StatusDone       Status = "DONE"
	StatusBlocked    Status = "BLOCKED"
)

// ValidStatuses is the set of all valid status values.
var ValidStatuses = map[Status]bool{
	StatusBacklog:    true,
	StatusOpen:       true,
	StatusInProgress: true,
	StatusReview:     true,
	StatusRework:     true,
	StatusDone:       true,
	StatusBlocked:    true,
}

// Priority represents a ticket priority (P0 = critical, P5 = backlog).
type Priority string

const (
	PriorityP0 Priority = "P0"
	PriorityP1 Priority = "P1"
	PriorityP2 Priority = "P2"
	PriorityP3 Priority = "P3"
	PriorityP4 Priority = "P4"
	PriorityP5 Priority = "P5"
)

// DefaultPriority is P3 (Normal).
const DefaultPriority = PriorityP3

// ValidPriorities is the set of all valid priority values.
var ValidPriorities = map[Priority]bool{
	PriorityP0: true,
	PriorityP1: true,
	PriorityP2: true,
	PriorityP3: true,
	PriorityP4: true,
	PriorityP5: true,
}

// Ticket represents a smoovtask ticket with frontmatter and body.
type Ticket struct {
	ID          string    `yaml:"id"`
	Title       string    `yaml:"title"`
	Project     string    `yaml:"project"`
	Status      Status    `yaml:"status"`
	PriorStatus *Status   `yaml:"prior-status"`
	Assignee    string    `yaml:"assignee"`
	Priority    Priority  `yaml:"priority"`
	DependsOn   []string  `yaml:"depends-on"`
	Created     time.Time `yaml:"created"`
	Updated     time.Time `yaml:"updated"`
	Tags        []string  `yaml:"tags"`

	// Body is the markdown body below the frontmatter.
	Body string `yaml:"-"`
}

// Filename returns the expected filename for this ticket.
// Format: 2026-02-25T10:00-st_xxxxxx.md
func (t *Ticket) Filename() string {
	return t.Created.UTC().Format("2006-01-02T15:04") + "-" + t.ID + ".md"
}
