package ticket

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	input := `---
id: sb_a7Kx2m
title: Add rate limiting to API
project: api-server
status: OPEN
prior-status: null
assignee: ""
priority: P2
depends-on: []
created: 2026-02-25T10:00:00Z
updated: 2026-02-25T10:00:00Z
tags: [api, security]
---

## Created — 2026-02-25T10:00:00Z
**actor:** human

Add rate limiting middleware to all public endpoints.
`

	tk, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if tk.ID != "sb_a7Kx2m" {
		t.Errorf("ID = %q, want %q", tk.ID, "sb_a7Kx2m")
	}
	if tk.Title != "Add rate limiting to API" {
		t.Errorf("Title = %q, want %q", tk.Title, "Add rate limiting to API")
	}
	if tk.Project != "api-server" {
		t.Errorf("Project = %q, want %q", tk.Project, "api-server")
	}
	if tk.Status != StatusOpen {
		t.Errorf("Status = %q, want %q", tk.Status, StatusOpen)
	}
	if tk.Priority != PriorityP2 {
		t.Errorf("Priority = %q, want %q", tk.Priority, PriorityP2)
	}
	if tk.Assignee != "" {
		t.Errorf("Assignee = %q, want empty", tk.Assignee)
	}

	wantCreated := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	if !tk.Created.Equal(wantCreated) {
		t.Errorf("Created = %v, want %v", tk.Created, wantCreated)
	}

	if len(tk.Tags) != 2 || tk.Tags[0] != "api" || tk.Tags[1] != "security" {
		t.Errorf("Tags = %v, want [api security]", tk.Tags)
	}

	if tk.Body == "" {
		t.Error("Body is empty, expected content")
	}
}

func TestParseNoPriorStatus(t *testing.T) {
	input := `---
id: sb_test01
title: Test
project: test
status: OPEN
priority: P3
created: 2026-02-25T10:00:00Z
updated: 2026-02-25T10:00:00Z
tags: []
---

Body here.
`

	tk, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if tk.PriorStatus != nil {
		t.Errorf("PriorStatus = %v, want nil", tk.PriorStatus)
	}
}

func TestParseMissingFrontmatter(t *testing.T) {
	_, err := Parse([]byte("no frontmatter here"))
	if err == nil {
		t.Error("Parse() should error on missing frontmatter")
	}
}

func TestParseMissingClosingDelimiter(t *testing.T) {
	_, err := Parse([]byte("---\nid: test\nno closing"))
	if err == nil {
		t.Error("Parse() should error on missing closing delimiter")
	}
}

func TestSplitFrontmatter(t *testing.T) {
	input := "---\nkey: value\n---\n\nBody content."
	fm, body, err := splitFrontmatter(input)
	if err != nil {
		t.Fatalf("splitFrontmatter() error: %v", err)
	}
	if fm != "key: value" {
		t.Errorf("frontmatter = %q, want %q", fm, "key: value")
	}
	if body != "\nBody content." {
		t.Errorf("body = %q, want %q", body, "\nBody content.")
	}
}
