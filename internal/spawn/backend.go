// Package spawn manages launching AI agent workers in isolated git worktrees.
package spawn

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Backend launches a non-interactive AI agent process.
type Backend interface {
	// Name returns the backend identifier (e.g. "claude").
	Name() string

	// Start launches the agent with the given prompt in the given working directory.
	// It returns the started command (for PID tracking) without waiting for it to finish.
	// If logPath is non-empty, stdout/stderr are written to that file.
	Start(ctx context.Context, workdir, prompt, logPath string) (*exec.Cmd, error)
}

// ClaudeBackend runs claude -p in non-interactive mode.
type ClaudeBackend struct{}

func (b *ClaudeBackend) Name() string { return "claude" }

func (b *ClaudeBackend) Start(ctx context.Context, workdir, prompt, logPath string) (*exec.Cmd, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	cmd := exec.CommandContext(ctx, claudePath, "-p", prompt)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "ST_ROLE=worker")

	if logPath != "" {
		f, err := os.Create(logPath)
		if err != nil {
			return nil, fmt.Errorf("create worker log %s: %w", logPath, err)
		}
		cmd.Stdout = f
		cmd.Stderr = f
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude process: %w", err)
	}

	return cmd, nil
}

// GetBackend returns a backend by name.
func GetBackend(name string) (Backend, error) {
	switch name {
	case "claude", "":
		return &ClaudeBackend{}, nil
	default:
		return nil, fmt.Errorf("unknown backend %q (available: claude)", name)
	}
}
