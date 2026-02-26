package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// newTestEnvResolved wraps newTestEnv and fixes the macOS /var → /private/var
// symlink mismatch that causes project.Detect to fail. On macOS, t.TempDir()
// returns /var/folders/... but os.Getwd() resolves the symlink to /private/var/...
func newTestEnvResolved(t *testing.T) *testEnv {
	t.Helper()
	env := newTestEnv(t)

	// os.Getwd() may resolve symlinks (macOS: /var → /private/var).
	// The config was written with t.TempDir() paths which may not be resolved.
	// Rewrite config with the resolved CWD so project.Detect matches.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	// Check if the config's project path already matches CWD
	for _, proj := range env.Config.Projects {
		if proj.Path == cwd {
			return env
		}
	}

	// Paths differ — rewrite config with resolved CWD
	resolvedVault := filepath.Join(cwd, "vault")
	configContent := "[settings]\nvault_path = \"" + resolvedVault + "\"\n\n[projects]\n[projects.testproject]\npath = \"" + cwd + "\"\n"
	configPath := filepath.Join(env.ConfigDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	// Reload config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	env.Config = cfg

	// Update store to point to resolved tickets dir
	resolvedTicketsDir := filepath.Join(resolvedVault, "tickets")
	if err := os.MkdirAll(resolvedTicketsDir, 0o755); err != nil {
		t.Fatalf("create resolved tickets dir: %v", err)
	}
	env.TicketsDir = resolvedTicketsDir
	env.Store = ticket.NewStore(resolvedTicketsDir)

	return env
}

func TestNew_HappyPath(t *testing.T) {
	env := newTestEnvResolved(t)

	out, err := env.runCmd(t, "new", "my first ticket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Created st_") {
		t.Errorf("output = %q, want substring %q", out, "Created st_")
	}
	if !strings.Contains(out, "my first ticket") {
		t.Errorf("output = %q, want title in output", out)
	}

	// Verify ticket exists in store
	tickets, err := env.Store.List(ticket.ListFilter{Project: "testproject"})
	if err != nil {
		t.Fatalf("list tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("got %d tickets, want 1", len(tickets))
	}

	tk := tickets[0]
	if tk.Title != "my first ticket" {
		t.Errorf("title = %q, want %q", tk.Title, "my first ticket")
	}
	if tk.Status != ticket.StatusOpen {
		t.Errorf("status = %s, want OPEN", tk.Status)
	}
	if tk.Priority != ticket.PriorityP3 {
		t.Errorf("priority = %s, want P3", tk.Priority)
	}
	if tk.Project != "testproject" {
		t.Errorf("project = %q, want %q", tk.Project, "testproject")
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	if len(events) == 0 {
		t.Error("no events logged")
	}
	if events[0].Event != event.TicketCreated {
		t.Errorf("event type = %q, want %q", events[0].Event, event.TicketCreated)
	}
}

func TestNew_PriorityFlag(t *testing.T) {
	env := newTestEnvResolved(t)

	out, err := env.runCmd(t, "new", "--priority", "P1", "urgent ticket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created st_") {
		t.Errorf("output = %q, want substring %q", out, "Created st_")
	}

	tickets, err := env.Store.List(ticket.ListFilter{Project: "testproject"})
	if err != nil {
		t.Fatalf("list tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("got %d tickets, want 1", len(tickets))
	}
	if tickets[0].Priority != ticket.PriorityP1 {
		t.Errorf("priority = %s, want P1", tickets[0].Priority)
	}
}

func TestNew_TagsFlag(t *testing.T) {
	env := newTestEnvResolved(t)

	_, err := env.runCmd(t, "new", "--tags", "foo,bar", "tagged ticket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tickets, err := env.Store.List(ticket.ListFilter{Project: "testproject"})
	if err != nil {
		t.Fatalf("list tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("got %d tickets, want 1", len(tickets))
	}

	tk := tickets[0]
	if len(tk.Tags) != 2 {
		t.Fatalf("tags = %v, want 2 tags", tk.Tags)
	}
	if tk.Tags[0] != "foo" || tk.Tags[1] != "bar" {
		t.Errorf("tags = %v, want [foo bar]", tk.Tags)
	}
}

func TestNew_DependsOnUnresolved(t *testing.T) {
	env := newTestEnvResolved(t)

	// Create a dependency ticket that is NOT done
	dep := env.createTicket(t, "dependency", ticket.StatusOpen)

	out, err := env.runCmd(t, "new", "--depends-on", dep.ID, "dependent ticket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Auto-blocked") {
		t.Errorf("output = %q, want substring %q", out, "Auto-blocked")
	}

	// Find the new ticket (not the dependency)
	tickets, err := env.Store.List(ticket.ListFilter{Project: "testproject"})
	if err != nil {
		t.Fatalf("list tickets: %v", err)
	}

	var newTk *ticket.Ticket
	for _, tk := range tickets {
		if tk.Title == "dependent ticket" {
			newTk = tk
			break
		}
	}
	if newTk == nil {
		t.Fatal("dependent ticket not found")
	}

	if newTk.Status != ticket.StatusBlocked {
		t.Errorf("status = %s, want BLOCKED", newTk.Status)
	}
	if newTk.PriorStatus == nil {
		t.Fatal("prior_status is nil, want OPEN")
	}
	if *newTk.PriorStatus != ticket.StatusOpen {
		t.Errorf("prior_status = %s, want OPEN", *newTk.PriorStatus)
	}
}

func TestNew_DependsOnAllDone(t *testing.T) {
	env := newTestEnvResolved(t)

	// Create a dependency ticket that IS done
	dep := env.createTicket(t, "done dependency", ticket.StatusDone)

	out, err := env.runCmd(t, "new", "--depends-on", dep.ID, "unblocked ticket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out, "Auto-blocked") {
		t.Errorf("output = %q, should not contain Auto-blocked", out)
	}

	// Find the new ticket
	tickets, err := env.Store.List(ticket.ListFilter{Project: "testproject"})
	if err != nil {
		t.Fatalf("list tickets: %v", err)
	}

	var newTk *ticket.Ticket
	for _, tk := range tickets {
		if tk.Title == "unblocked ticket" {
			newTk = tk
			break
		}
	}
	if newTk == nil {
		t.Fatal("unblocked ticket not found")
	}

	if newTk.Status != ticket.StatusOpen {
		t.Errorf("status = %s, want OPEN", newTk.Status)
	}
}

func TestNew_DependsOnMissingDep(t *testing.T) {
	env := newTestEnvResolved(t)

	// Depend on a ticket ID that doesn't exist — CheckDependencies treats
	// missing deps as unresolved, so the new ticket should auto-block.
	out, err := env.runCmd(t, "new", "--depends-on", "st_zzzzzz", "orphan dep ticket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The ticket is created then auto-blocked because the dep can't be found
	if !strings.Contains(out, "Auto-blocked") {
		t.Errorf("output = %q, want substring %q", out, "Auto-blocked")
	}
}

func TestNew_InvalidPriority(t *testing.T) {
	env := newTestEnvResolved(t)
	_ = env

	_, err := env.runCmd(t, "new", "--priority", "P9", "bad priority")
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}
