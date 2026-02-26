package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// testEnv sets up a temp config, tickets dir, and events dir.
// It sets SMOOVBRAIN_DIR so config.Load() and EventsDir() use the test paths.
type testEnv struct {
	ConfigDir  string
	TicketsDir string
	EventsDir  string
	Store      *ticket.Store
	EventLog   *event.EventLog
	Config     *config.Config
}

// newTestEnv creates a fully isolated test environment.
// It sets SMOOVBRAIN_DIR to redirect config.Load() and EventsDir(),
// and writes a minimal config.toml with a test project.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "config")
	ticketsDir := filepath.Join(baseDir, "vault", "tickets")
	eventsDir := filepath.Join(configDir, "events")

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatalf("create tickets dir: %v", err)
	}
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatalf("create events dir: %v", err)
	}

	// Write config.toml with a test project pointing to baseDir
	vaultPath := filepath.Join(baseDir, "vault")
	configContent := "[settings]\nvault_path = \"" + vaultPath + "\"\n\n[projects]\n[projects.testproject]\npath = \"" + baseDir + "\"\n"
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	// Set SMOOVBRAIN_DIR so config.Load() and EventsDir() use our temp paths
	t.Setenv("SMOOVBRAIN_DIR", configDir)

	// Change to baseDir so project.Detect matches "testproject"
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(baseDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	store := ticket.NewStore(ticketsDir)
	el := event.NewEventLog(eventsDir)

	return &testEnv{
		ConfigDir:  configDir,
		TicketsDir: ticketsDir,
		EventsDir:  eventsDir,
		Store:      store,
		EventLog:   el,
		Config:     cfg,
	}
}

// createTicket creates a test ticket with the given title and status.
func (e *testEnv) createTicket(t *testing.T, title string, status ticket.Status) *ticket.Ticket {
	t.Helper()

	now := time.Now().UTC()
	tk := &ticket.Ticket{
		Title:     title,
		Project:   "testproject",
		Status:    status,
		Priority:  ticket.DefaultPriority,
		Created:   now,
		Updated:   now,
		Tags:      []string{},
		DependsOn: []string{},
	}

	if err := e.Store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	return tk
}

// addNoteEvent logs a ticket.note event for the given ticket, so that
// the RequiresNote check passes when transitioning to REVIEW.
func (e *testEnv) addNoteEvent(t *testing.T, ticketID string) {
	t.Helper()
	_ = e.EventLog.Append(event.Event{
		TS:      time.Now().UTC(),
		Event:   event.TicketNote,
		Ticket:  ticketID,
		Project: "testproject",
		Actor:   "agent",
		Data:    map[string]any{"message": "test note"},
	})
}

// runCmd executes a cobra command with the given args and captures stdout.
// Commands use fmt.Printf (writes to os.Stdout), so we redirect os.Stdout
// to a pipe to capture output.
func (e *testEnv) runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Capture stdout by redirecting os.Stdout to a pipe
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w

	// Reset global flag vars to avoid state leakage between tests
	resetFlags()

	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()

	// Close writer and restore stdout before reading
	w.Close()
	os.Stdout = origStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	r.Close()

	return string(out), execErr
}

// resetFlags resets package-level flag variables to their defaults
// so tests don't leak state between runs.
func resetFlags() {
	statusTicket = ""
	noteTicket = ""
	listProject = ""
	listStatus = ""
	listAll = false
	newPriority = "P3"
	newTags = ""
	newDependsOn = ""
	newDescription = ""
}

func TestOverride_HappyPath(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "test override", ticket.StatusOpen)

	out, err := env.runCmd(t, "override", tk.ID, "done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := out; !strings.Contains(got, "Override "+tk.ID) {
		t.Errorf("output = %q, want substring %q", got, "Override "+tk.ID)
	}
	if !strings.Contains(out, "OPEN → DONE") {
		t.Errorf("output = %q, want substring %q", out, "OPEN → DONE")
	}

	// Verify ticket was updated
	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusDone {
		t.Errorf("status = %s, want DONE", updated.Status)
	}
	if updated.PriorStatus != nil {
		t.Errorf("prior_status = %v, want nil", updated.PriorStatus)
	}
}

func TestOverride_AliasResolution(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "test alias", ticket.StatusOpen)

	out, err := env.runCmd(t, "override", tk.ID, "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "OPEN → IN-PROGRESS") {
		t.Errorf("output = %q, want substring %q", out, "OPEN → IN-PROGRESS")
	}
}

func TestOverride_AnyTransition(t *testing.T) {
	env := newTestEnv(t)
	// DONE → OPEN is normally invalid, but override bypasses rules
	tk := env.createTicket(t, "test any transition", ticket.StatusDone)

	_, err := env.runCmd(t, "override", tk.ID, "open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != ticket.StatusOpen {
		t.Errorf("status = %s, want OPEN", updated.Status)
	}
}

func TestOverride_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "test bad status", ticket.StatusOpen)

	_, err := env.runCmd(t, "override", tk.ID, "bogus")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestOverride_TicketNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "override", "st_zzzzzz", "done")
	if err == nil {
		t.Fatal("expected error for missing ticket")
	}
}

func TestContext_NoSession(t *testing.T) {
	env := newTestEnv(t)
	_ = env
	t.Setenv("CLAUDE_SESSION_ID", "")

	out, err := env.runCmd(t, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"session_id": ""`) {
		t.Errorf("output = %q, want session_id empty", out)
	}
	if !strings.Contains(out, `"active_ticket": null`) {
		t.Errorf("output = %q, want active_ticket null", out)
	}
}

func TestContext_WithActiveTicket(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("CLAUDE_SESSION_ID", "test-session-123")

	tk := env.createTicket(t, "active ticket", ticket.StatusInProgress)
	tk.Assignee = "test-session-123"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"session_id": "test-session-123"`) {
		t.Errorf("output = %q, want session_id test-session-123", out)
	}
	if !strings.Contains(out, tk.ID) {
		t.Errorf("output = %q, want active_ticket %s", out, tk.ID)
	}
}
