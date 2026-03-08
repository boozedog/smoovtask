package cmd

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// testEnv sets up a temp config, projects dir, and events dir.
// It sets SMOOVBRAIN_DIR so config.Load() and EventsDir() use the test paths.
type testEnv struct {
	ConfigDir   string
	ProjectsDir string
	EventsDir   string
	Store       *ticket.Store
	EventLog    *event.EventLog
	Config      *config.Config
}

// newTestEnv creates a fully isolated test environment.
// It sets SMOOVBRAIN_DIR to redirect config.Load() and EventsDir(),
// and writes a minimal config.toml with a test project.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "config")
	projectsDir := filepath.Join(baseDir, "vault", "projects")
	eventsDir := filepath.Join(configDir, "events")

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("create projects dir: %v", err)
	}
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatalf("create events dir: %v", err)
	}

	// Write config.toml with vault path only.
	vaultPath := filepath.Join(baseDir, "vault")
	configContent := "[settings]\nvault_path = \"" + vaultPath + "\"\n"
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	// Register "testproject" via vault project.md.
	if err := project.SaveMeta(vaultPath, "testproject", &project.ProjectMeta{Path: baseDir}); err != nil {
		t.Fatalf("save project meta: %v", err)
	}

	// Set SMOOVBRAIN_DIR so config.Load() and EventsDir() use our temp paths
	t.Setenv("SMOOVBRAIN_DIR", configDir)

	// Initialize a git repo so worktree checks work in tests.
	for _, gitArgs := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", gitArgs...)
		cmd.Dir = baseDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", gitArgs, out, err)
		}
	}

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

	// Initialize a git repo so commands that resolve the repo root work.
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = baseDir
		if out, gitErr := cmd.CombinedOutput(); gitErr != nil {
			t.Fatalf("git %v: %s: %v", args, out, gitErr)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	store := ticket.NewStore(projectsDir)
	el := event.NewEventLog(eventsDir)

	return &testEnv{
		ConfigDir:   configDir,
		ProjectsDir: projectsDir,
		EventsDir:   eventsDir,
		Store:       store,
		EventLog:    el,
		Config:      cfg,
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
// the RequiresNote check passes when transitioning to review states.
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

// ensureCleanWorktree creates a git worktree at .worktrees/<ticketID> with a
// clean working tree, so requireCleanWorktree passes during tests.
func (e *testEnv) ensureCleanWorktree(t *testing.T, ticketID string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	wtDir := filepath.Join(cwd, ".worktrees")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		t.Fatalf("create .worktrees dir: %v", err)
	}

	branch := "st/" + ticketID
	wtPath := filepath.Join(wtDir, ticketID)
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %s: %v", out, err)
	}
}

// runCmd executes a cobra command with the given args and captures stdout.
// Commands use fmt.Printf (writes to os.Stdout), so we redirect os.Stdout
// to a pipe to capture output.
func (e *testEnv) runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	effectiveArgs := args
	if shouldDefaultToHuman(args) {
		effectiveArgs = append([]string{"--human"}, args...)
	}

	return e.runCmdRaw(t, effectiveArgs...)
}

func (e *testEnv) runCmdRaw(t *testing.T, args ...string) (string, error) {
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

func shouldDefaultToHuman(args []string) bool {
	for _, arg := range args {
		if arg == "--human" || arg == "--run-id" {
			return false
		}
		if strings.HasPrefix(arg, "--run-id=") {
			return false
		}
	}

	if len(args) > 0 && args[0] == "hook" {
		return false
	}

	return true
}

// resetFlags resets package-level flag variables to their defaults
// so tests don't leak state between runs.
func resetFlags() {
	runIDFlag = ""
	humanFlag = false
	statusTicket = ""
	noteTicket = ""
	listProject = ""
	listStatus = ""
	listAll = false
	newPriority = "P3"
	newTags = ""
	newDependsOn = ""
	newDescription = ""
	newProject = ""
	newTitle = ""
	pickTicket = ""
	reviewTicket = ""
	reviewCLI = ""
	leaderCLI = ""
	workCLI = ""
	handoffTicket = ""
	prepTicket = ""
	prepBase = ""
	spawnTimeout = 45 * time.Minute
	spawnBackend = "claude"
	spawnDryRun = false
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

	out, err := env.runCmd(t, "--human", "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"run_id": ""`) {
		t.Errorf("output = %q, want run_id empty", out)
	}
	if !strings.Contains(out, `"active_ticket": null`) {
		t.Errorf("output = %q, want active_ticket null", out)
	}
}

func TestContext_WithActiveTicket(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "active ticket", ticket.StatusInProgress)
	tk.Assignee = "test-session-123"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "--run-id", "test-session-123", "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"run_id": "test-session-123"`) {
		t.Errorf("output = %q, want run_id test-session-123", out)
	}
	if !strings.Contains(out, tk.ID) {
		t.Errorf("output = %q, want active_ticket %s", out, tk.ID)
	}
}
