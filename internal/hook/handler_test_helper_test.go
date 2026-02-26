package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
)

// testEnv holds paths for a test environment with config and events dirs.
type testEnv struct {
	Home      string
	ConfigDir string
	EventsDir string
}

// setupTestEnv creates a temporary directory structure that mimics ~/.smoovtask
// and sets $HOME so that config.Load() and EventsDir() resolve correctly.
// The optional projectPath, if non-empty, registers a project named "test-project"
// pointing at that path.
func setupTestEnv(t *testing.T, projectPath string) testEnv {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".smoovtask")
	eventsDir := filepath.Join(configDir, "events")
	vaultDir := filepath.Join(tmpDir, "vault")
	ticketsDir := filepath.Join(vaultDir, "tickets")

	for _, d := range []string{configDir, eventsDir, ticketsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Build config.toml
	cfg := "[settings]\nvault_path = " + quote(vaultDir) + "\n"
	if projectPath != "" {
		cfg += "\n[projects.test-project]\npath = " + quote(projectPath) + "\n"
	}

	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	return testEnv{
		Home:      tmpDir,
		ConfigDir: configDir,
		EventsDir: eventsDir,
	}
}

// ticketsDir returns the tickets directory path for this test env.
func (e testEnv) ticketsDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(e.Home, "vault", "tickets")
}

// quote returns a TOML-safe quoted string.
func quote(s string) string {
	return `"` + s + `"`
}

// readTodayEvent reads the first event from today's JSONL file in eventsDir.
func readTodayEvent(t *testing.T, eventsDir string) event.Event {
	t.Helper()

	filename := time.Now().UTC().Format("2006-01-02") + ".jsonl"
	path := filepath.Join(eventsDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read events file %s: %v", path, err)
	}

	var ev event.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("unmarshal event: %v (data: %s)", err, data)
	}
	return ev
}

// assertEvent verifies common event fields.
func assertEvent(t *testing.T, ev event.Event, wantType, wantSession, wantProject string) {
	t.Helper()

	if ev.Event != wantType {
		t.Errorf("event type = %q, want %q", ev.Event, wantType)
	}
	if ev.Session != wantSession {
		t.Errorf("session = %q, want %q", ev.Session, wantSession)
	}
	if ev.Project != wantProject {
		t.Errorf("project = %q, want %q", ev.Project, wantProject)
	}
	if ev.Actor != "agent" {
		t.Errorf("actor = %q, want %q", ev.Actor, "agent")
	}
	if ev.TS.IsZero() {
		t.Error("timestamp is zero")
	}
}
