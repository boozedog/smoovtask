package ticket

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCreate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	tk := &Ticket{
		Title:     "Test ticket",
		Project:   "test-project",
		Status:    StatusOpen,
		Priority:  PriorityP3,
		DependsOn: []string{},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
		Tags:      []string{"test"},
	}
	AppendSection(tk, "Created", "human", "", "Test description.", nil, tk.Created)

	if err := store.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if tk.ID == "" {
		t.Error("ticket ID should be set after Create")
	}

	// Verify file exists
	path := filepath.Join(dir, tk.Filename())
	if _, err := os.Stat(path); err != nil {
		t.Errorf("ticket file not found: %v", err)
	}
}

func TestStoreGet(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	tk := &Ticket{
		Title:     "Get test",
		Project:   "test",
		Status:    StatusOpen,
		Priority:  PriorityP2,
		DependsOn: []string{},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
		Tags:      []string{},
	}
	AppendSection(tk, "Created", "human", "", "Description.", nil, tk.Created)

	if err := store.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := store.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if got.ID != tk.ID {
		t.Errorf("ID = %q, want %q", got.ID, tk.ID)
	}
	if got.Title != tk.Title {
		t.Errorf("Title = %q, want %q", got.Title, tk.Title)
	}
}

func TestStoreGetByPrefix(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	tk := &Ticket{
		Title:     "Prefix test",
		Project:   "test",
		Status:    StatusOpen,
		Priority:  PriorityP3,
		DependsOn: []string{},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
		Tags:      []string{},
	}
	AppendSection(tk, "Created", "human", "", "Description.", nil, tk.Created)

	if err := store.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Get by prefix (st_ + first 2 chars)
	prefix := tk.ID[:5] // st_xx
	got, err := store.Get(prefix)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", prefix, err)
	}

	if got.ID != tk.ID {
		t.Errorf("ID = %q, want %q", got.ID, tk.ID)
	}
}

func TestStoreGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	_, err := store.Get("st_zzzzzz")
	if err == nil {
		t.Error("Get() should error for nonexistent ticket")
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	now := time.Now().UTC()
	for i, proj := range []string{"proj-a", "proj-a", "proj-b"} {
		tk := &Ticket{
			Title:     "Ticket " + string(rune('A'+i)),
			Project:   proj,
			Status:    StatusOpen,
			Priority:  PriorityP3,
			DependsOn: []string{},
			Created:   now.Add(time.Duration(i) * time.Minute),
			Updated:   now.Add(time.Duration(i) * time.Minute),
			Tags:      []string{},
		}
		AppendSection(tk, "Created", "human", "", "Desc.", nil, tk.Created)
		if err := store.Create(tk); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// List all
	all, err := store.List(ListFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List() returned %d tickets, want 3", len(all))
	}

	// Filter by project
	projA, err := store.List(ListFilter{Project: "proj-a"})
	if err != nil {
		t.Fatalf("List(proj-a) error: %v", err)
	}
	if len(projA) != 2 {
		t.Errorf("List(proj-a) returned %d tickets, want 2", len(projA))
	}

	// Filter by status (none are IN-PROGRESS)
	inProg, err := store.List(ListFilter{Status: StatusInProgress})
	if err != nil {
		t.Fatalf("List(IN-PROGRESS) error: %v", err)
	}
	if len(inProg) != 0 {
		t.Errorf("List(IN-PROGRESS) returned %d tickets, want 0", len(inProg))
	}
}

func TestStoreListMeta(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	now := time.Now().UTC()
	first := &Ticket{
		Title:     "Meta A",
		Project:   "proj-a",
		Status:    StatusOpen,
		Priority:  PriorityP3,
		DependsOn: []string{},
		Created:   now,
		Updated:   now,
		Tags:      []string{"x"},
	}
	AppendSection(first, "Created", "human", "", "Body A", nil, first.Created)
	if err := store.Create(first); err != nil {
		t.Fatalf("Create(first) error: %v", err)
	}

	second := &Ticket{
		Title:     "Meta B",
		Project:   "proj-b",
		Status:    StatusInProgress,
		Priority:  PriorityP1,
		DependsOn: []string{first.ID},
		Created:   now.Add(time.Minute),
		Updated:   now.Add(time.Minute),
		Tags:      []string{"y"},
	}
	AppendSection(second, "Created", "human", "", "Body B", nil, second.Created)
	if err := store.Create(second); err != nil {
		t.Fatalf("Create(second) error: %v", err)
	}

	meta, err := store.ListMeta(ListFilter{})
	if err != nil {
		t.Fatalf("ListMeta() error: %v", err)
	}
	if len(meta) != 2 {
		t.Fatalf("ListMeta() returned %d tickets, want 2", len(meta))
	}

	for _, tk := range meta {
		if tk.Body != "" {
			t.Errorf("ListMeta() Body = %q, want empty", tk.Body)
		}
	}

	inProg, err := store.ListMeta(ListFilter{Status: StatusInProgress})
	if err != nil {
		t.Fatalf("ListMeta(IN-PROGRESS) error: %v", err)
	}
	if len(inProg) != 1 {
		t.Fatalf("ListMeta(IN-PROGRESS) returned %d tickets, want 1", len(inProg))
	}
	if inProg[0].Project != "proj-b" {
		t.Errorf("project = %q, want %q", inProg[0].Project, "proj-b")
	}
	if len(inProg[0].DependsOn) != 1 || inProg[0].DependsOn[0] != first.ID {
		t.Errorf("depends-on = %v, want [%s]", inProg[0].DependsOn, first.ID)
	}
}

func TestStoreListEmptyDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	store := NewStore(dir)
	tickets, err := store.List(ListFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("List() returned %d tickets, want 0", len(tickets))
	}
}

func TestStoreSave(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	tk := &Ticket{
		Title:     "Save test",
		Project:   "test",
		Status:    StatusOpen,
		Priority:  PriorityP3,
		DependsOn: []string{},
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
		Tags:      []string{},
	}
	AppendSection(tk, "Created", "human", "", "Description.", nil, tk.Created)

	if err := store.Create(tk); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Modify and save
	tk.Status = StatusInProgress
	tk.Assignee = "agent-01"
	tk.Updated = time.Now().UTC()

	if err := store.Save(tk); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify
	got, err := store.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Status != StatusInProgress {
		t.Errorf("Status = %q, want %q", got.Status, StatusInProgress)
	}
	if got.Assignee != "agent-01" {
		t.Errorf("Assignee = %q, want %q", got.Assignee, "agent-01")
	}
}
