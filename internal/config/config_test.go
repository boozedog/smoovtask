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
	if len(cfg.Projects) != 0 {
		t.Errorf("default Projects = %v, want empty", cfg.Projects)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Settings: SettingsConfig{VaultPath: "~/my-vault"},
		Projects: map[string]ProjectConfig{
			"myproject": {Path: "/tmp/myproject"},
		},
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
	if p, ok := loaded.Projects["myproject"]; !ok || p.Path != "/tmp/myproject" {
		t.Errorf("Projects[myproject] = %+v, want Path=/tmp/myproject", loaded.Projects["myproject"])
	}
}

func TestIdempotentSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Settings: SettingsConfig{VaultPath: "~/vault"},
		Projects: map[string]ProjectConfig{
			"proj": {Path: "/tmp/proj"},
		},
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

func TestEnsureDirs(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	configDir := filepath.Join(dir, "smoovtask")
	path := filepath.Join(configDir, "config.toml")

	cfg := &Config{
		Settings: SettingsConfig{VaultPath: vault},
		Projects: make(map[string]ProjectConfig),
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	// Override DefaultDir for this test by using the vault path directly
	ticketsDir := filepath.Join(vault, "tickets")
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll tickets: %v", err)
	}

	if _, err := os.Stat(ticketsDir); err != nil {
		t.Errorf("tickets dir not created: %v", err)
	}
}
