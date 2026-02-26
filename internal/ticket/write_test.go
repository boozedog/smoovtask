package ticket

import (
	"strings"
	"testing"
	"time"
)

func TestMarshal(t *testing.T) {
	created := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	tk := &Ticket{
		ID:        "sb_a7Kx2m",
		Title:     "Add rate limiting",
		Project:   "api-server",
		Status:    StatusOpen,
		Priority:  PriorityP2,
		DependsOn: []string{},
		Created:   created,
		Updated:   created,
		Tags:      []string{"api", "security"},
		Body:      "\n## Created — 2026-02-25T10:00:00Z\n**actor:** human\n\nAdd rate limiting.\n",
	}

	data, err := Marshal(tk)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		t.Error("missing opening frontmatter delimiter")
	}

	if !strings.Contains(content, "id: sb_a7Kx2m") {
		t.Error("missing ID in frontmatter")
	}
	if !strings.Contains(content, "title: Add rate limiting") {
		t.Error("missing title in frontmatter")
	}
	if !strings.Contains(content, "status: OPEN") {
		t.Error("missing status in frontmatter")
	}
	if !strings.Contains(content, "priority: P2") {
		t.Error("missing priority in frontmatter")
	}
	if !strings.Contains(content, "## Created") {
		t.Error("missing body section")
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	created := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	original := &Ticket{
		ID:        "sb_test01",
		Title:     "Test ticket",
		Project:   "test-project",
		Status:    StatusOpen,
		Priority:  PriorityP3,
		DependsOn: []string{},
		Created:   created,
		Updated:   created,
		Tags:      []string{"test"},
		Body:      "\n## Created — 2026-02-25T10:00:00Z\n**actor:** human\n\nTest description.\n",
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Title != original.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, original.Title)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status = %q, want %q", parsed.Status, original.Status)
	}
	if parsed.Priority != original.Priority {
		t.Errorf("Priority = %q, want %q", parsed.Priority, original.Priority)
	}
	if !parsed.Created.Equal(original.Created) {
		t.Errorf("Created = %v, want %v", parsed.Created, original.Created)
	}
}

func TestMarshalNilSlices(t *testing.T) {
	tk := &Ticket{
		ID:       "sb_test02",
		Title:    "Nil slices",
		Project:  "test",
		Status:   StatusOpen,
		Priority: PriorityP3,
		Created:  time.Now(),
		Updated:  time.Now(),
		// DependsOn and Tags are nil
	}

	data, err := Marshal(tk)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "depends-on: []") {
		t.Error("nil DependsOn should marshal as empty list")
	}
	if !strings.Contains(content, "tags: []") {
		t.Error("nil Tags should marshal as empty list")
	}
}

func TestAppendSection(t *testing.T) {
	tk := &Ticket{Body: ""}
	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	AppendSection(tk, "Created", "human", "", "Initial description.", nil, ts)

	if !strings.Contains(tk.Body, "## Created — 2026-02-25T10:00:00Z") {
		t.Error("missing section heading")
	}
	if !strings.Contains(tk.Body, "**actor:** human") {
		t.Error("missing actor line")
	}
	if !strings.Contains(tk.Body, "Initial description.") {
		t.Error("missing content")
	}
}

func TestAppendSectionMultiple(t *testing.T) {
	tk := &Ticket{Body: ""}
	ts1 := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC)

	AppendSection(tk, "Created", "human", "", "First.", nil, ts1)
	AppendSection(tk, "In Progress", "agent-01", "sess-123", "Starting work.", nil, ts2)

	if strings.Count(tk.Body, "## ") != 2 {
		t.Errorf("expected 2 sections, got %d", strings.Count(tk.Body, "## "))
	}
}

func TestAppendSectionNoContent(t *testing.T) {
	tk := &Ticket{Body: ""}
	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	AppendSection(tk, "Assigned", "sb", "", "", nil, ts)

	if !strings.Contains(tk.Body, "## Assigned") {
		t.Error("missing section heading")
	}
	if !strings.Contains(tk.Body, "**actor:** sb") {
		t.Error("missing actor line")
	}
}
