package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds the global smoovtask configuration.
type Config struct {
	Settings SettingsConfig           `toml:"settings"`
	Projects map[string]ProjectConfig `toml:"projects"`
}

// SettingsConfig holds global settings.
type SettingsConfig struct {
	VaultPath string `toml:"vault_path"`
}

// ProjectConfig holds per-project settings.
type ProjectConfig struct {
	Path string `toml:"path"`
	Repo string `toml:"repo,omitempty"`
}

// DefaultDir returns the default config directory (~/.smoovtask).
// If SMOOVBRAIN_DIR is set, uses that path instead.
func DefaultDir() (string, error) {
	if d := os.Getenv("SMOOVBRAIN_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".smoovtask"), nil
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads config from the default path, applying defaults.
// If the file doesn't exist, returns a config with defaults.
func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads config from the given path, applying defaults.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{
		Projects: make(map[string]ProjectConfig),
	}
	cfg.applyDefaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]ProjectConfig)
	}

	return cfg, nil
}

// Save writes config to the default path.
func (c *Config) Save() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes config to the given path, creating directories as needed.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return nil
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}

// VaultPath returns the expanded vault path.
func (c *Config) VaultPath() (string, error) {
	return ExpandPath(c.Settings.VaultPath)
}

// TicketsDir returns the expanded tickets directory path.
func (c *Config) TicketsDir() (string, error) {
	vault, err := c.VaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(vault, "tickets"), nil
}

// EventsDir returns the expanded events directory path.
func (c *Config) EventsDir() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "events"), nil
}

// EnsureDirs creates the vault tickets dir and events dir if they don't exist.
func (c *Config) EnsureDirs() error {
	tickets, err := c.TicketsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(tickets, 0o755); err != nil {
		return fmt.Errorf("create tickets dir: %w", err)
	}

	events, err := c.EventsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(events, 0o755); err != nil {
		return fmt.Errorf("create events dir: %w", err)
	}

	return nil
}

func (c *Config) applyDefaults() {
	if c.Settings.VaultPath == "" {
		c.Settings.VaultPath = "~/obsidian/smoovtask"
	}
}
