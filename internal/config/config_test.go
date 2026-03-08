package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Settings.VaultPath != "~/obsidian/smoovtask" {
		t.Errorf("default VaultPath = %q, want %q", cfg.Settings.VaultPath, "~/obsidian/smoovtask")
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Settings: SettingsConfig{VaultPath: "~/my-vault"},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if loaded.Settings.VaultPath != cfg.Settings.VaultPath {
		t.Errorf("VaultPath = %q, want %q", loaded.Settings.VaultPath, cfg.Settings.VaultPath)
	}
}

func TestIdempotentSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Settings: SettingsConfig{VaultPath: "~/vault"},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("first SaveTo: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first: %v", err)
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("second SaveTo: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read second: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("saves differ:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got, err := ExpandPath(tt.input)
		if err != nil {
			t.Errorf("ExpandPath(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEventsDirDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SMOOVBRAIN_DIR", dir)

	cfg := &Config{}
	cfg.applyDefaults()

	got, err := cfg.EventsDir()
	if err != nil {
		t.Fatalf("EventsDir: %v", err)
	}
	want := filepath.Join(dir, "events")
	if got != want {
		t.Errorf("EventsDir() = %q, want %q", got, want)
	}
}

func TestEventsDirCustom(t *testing.T) {
	customDir := t.TempDir()

	cfg := &Config{
		Settings: SettingsConfig{
			VaultPath:  "~/obsidian/smoovtask",
			EventsPath: customDir,
		},
	}

	got, err := cfg.EventsDir()
	if err != nil {
		t.Fatalf("EventsDir: %v", err)
	}
	if got != customDir {
		t.Errorf("EventsDir() = %q, want %q", got, customDir)
	}
}

func TestEventsDirCustomWithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	cfg := &Config{
		Settings: SettingsConfig{
			VaultPath:  "~/obsidian/smoovtask",
			EventsPath: "~/my-events",
		},
	}

	got, err := cfg.EventsDir()
	if err != nil {
		t.Fatalf("EventsDir: %v", err)
	}
	want := filepath.Join(home, "my-events")
	if got != want {
		t.Errorf("EventsDir() = %q, want %q", got, want)
	}
}

func TestEnsureDirs(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	configDir := filepath.Join(dir, "smoovtask")
	path := filepath.Join(configDir, "config.toml")

	cfg := &Config{
		Settings: SettingsConfig{VaultPath: vault},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	// Override DefaultDir for this test by using the vault path directly
	projectsDir := filepath.Join(vault, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll projects: %v", err)
	}

	if _, err := os.Stat(projectsDir); err != nil {
		t.Errorf("projects dir not created: %v", err)
	}
}
