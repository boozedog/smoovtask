package spawn

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

// Options configures a spawn operation.
type Options struct {
	TicketID string
	Backend  string
	Timeout  time.Duration
	RunID    string // run ID for the spawned worker (generated if empty)
}

// Result contains the outcome of a spawn operation.
type Result struct {
	PID          int
	WorktreePath string
	Branch       string
	RunID        string
	LogPath      string
	TmuxWindow   string // non-empty when launched in a tmux window

	// Wait blocks until the worker process exits and logs the outcome event.
	// The caller must call Wait to ensure monitoring completes.
	Wait func() error
}

// InTmux returns true if the current process is running inside tmux.
func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// Run executes the full spawn workflow:
// 1. Load ticket and create worktree
// 2. Build prompt and launch worker (tmux window if inside tmux, headless otherwise)
// 3. Log spawn.started event
// 4. Return Wait function for monitoring
func Run(opts Options) (*Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return nil, fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)
	tk, err := store.Get(opts.TicketID)
	if err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}

	// Find repo root — handle being in a worktree already
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	repoRoot, err := WorktreeRepoRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("find repo root: %w", err)
	}

	// Create worktree
	worktreePath, branch, err := CreateWorktree(repoRoot, tk.ID)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Generate run ID for the worker
	workerRunID := opts.RunID
	if workerRunID == "" {
		workerRunID = generateRunID()
	}

	// Build prompt
	prompt := BuildPrompt(tk, workerRunID)

	// Set up timeout context
	ctx := context.Background()
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	logPath := filepath.Join(worktreePath, "worker.log")

	// Set up event logger
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)

	// Dispatch: tmux window or headless
	if InTmux() {
		return launchTmux(ctx, cancel, el, tk, opts, workerRunID, worktreePath, branch, logPath, prompt)
	}
	return launchHeadless(ctx, cancel, el, tk, opts, workerRunID, worktreePath, branch, logPath, prompt)
}

func launchHeadless(ctx context.Context, cancel context.CancelFunc, el *event.EventLog, tk *ticket.Ticket, opts Options, workerRunID, worktreePath, branch, logPath, prompt string) (*Result, error) {
	backend, err := GetBackend(opts.Backend)
	if err != nil {
		cancel()
		return nil, err
	}

	cmd, err := backend.Start(ctx, worktreePath, prompt, logPath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("start worker: %w", err)
	}

	pid := cmd.Process.Pid
	now := time.Now().UTC()

	_ = el.Append(event.Event{
		TS:      now,
		Event:   SpawnStarted,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   "agent",
		RunID:   workerRunID,
		Data: map[string]any{
			"pid":      pid,
			"worktree": worktreePath,
			"branch":   branch,
			"backend":  backend.Name(),
			"timeout":  opts.Timeout.String(),
			"mode":     "headless",
		},
	})

	waitFn := func() error {
		defer cancel()
		waitErr := cmd.Wait()
		return logOutcome(el, tk, workerRunID, now, pid, cmd.ProcessState.ExitCode(), waitErr, ctx.Err())
	}

	return &Result{
		PID:          pid,
		WorktreePath: worktreePath,
		Branch:       branch,
		RunID:        workerRunID,
		LogPath:      logPath,
		Wait:         waitFn,
	}, nil
}

func launchTmux(ctx context.Context, cancel context.CancelFunc, el *event.EventLog, tk *ticket.Ticket, opts Options, workerRunID, worktreePath, branch, logPath, prompt string) (*Result, error) {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("tmux not found: %w", err)
	}

	// Write prompt to file — avoids shell escaping issues with long prompts
	promptFile := filepath.Join(worktreePath, ".worker-prompt")
	if err := os.WriteFile(promptFile, []byte(prompt), 0o600); err != nil {
		cancel()
		return nil, fmt.Errorf("write prompt file: %w", err)
	}

	// Channel for tmux wait-for synchronization
	channel := "st-worker-" + workerRunID

	// Shell command to run inside the tmux window.
	// Runs claude interactively (not -p) so the TUI is visible in the tmux window.
	// Uses --append-system-prompt for the full instructions (avoids shell escaping issues)
	// and a short positional prompt to kick off the work.
	shellCmd := fmt.Sprintf(
		`ST_ROLE=worker claude --permission-mode acceptEdits --allowedTools "Bash(git commit:*) Bash(git add:*) Bash(st pick:*) Bash(st note:*) Bash(st status:*)" --append-system-prompt "$(cat .worker-prompt)" "Pick up ticket %s and complete the task described in your system prompt."; echo $? > .worker-exitcode; %s wait-for -S %s`,
		tk.ID, tmuxPath, channel,
	)

	windowName := tk.ID
	createCmd := exec.Command(tmuxPath, "new-window", "-n", windowName, "-c", worktreePath, "sh", "-c", shellCmd)
	if err := createCmd.Run(); err != nil {
		cancel()
		return nil, fmt.Errorf("create tmux window: %w", err)
	}

	// Get the PID of the process running in the new pane
	pid := tmuxPanePID(tmuxPath, windowName)

	now := time.Now().UTC()
	_ = el.Append(event.Event{
		TS:      now,
		Event:   SpawnStarted,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   "agent",
		RunID:   workerRunID,
		Data: map[string]any{
			"pid":         pid,
			"worktree":    worktreePath,
			"branch":      branch,
			"backend":     opts.Backend,
			"timeout":     opts.Timeout.String(),
			"mode":        "tmux",
			"tmux_window": windowName,
		},
	})

	// Start the wait-for command — blocks until the tmux window signals completion
	waitCmd := exec.CommandContext(ctx, tmuxPath, "wait-for", channel)
	if err := waitCmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start tmux wait-for: %w", err)
	}

	waitFn := func() error {
		defer cancel()
		waitErr := waitCmd.Wait()
		exitCode := readExitCodeFile(filepath.Join(worktreePath, ".worker-exitcode"))
		return logOutcome(el, tk, workerRunID, now, pid, exitCode, waitErr, ctx.Err())
	}

	return &Result{
		PID:          pid,
		WorktreePath: worktreePath,
		Branch:       branch,
		RunID:        workerRunID,
		LogPath:      logPath,
		TmuxWindow:   windowName,
		Wait:         waitFn,
	}, nil
}

// logOutcome logs the spawn result event and returns an error if the worker failed.
func logOutcome(el *event.EventLog, tk *ticket.Ticket, runID string, started time.Time, pid, exitCode int, waitErr error, ctxErr error) error {
	now := time.Now().UTC()
	elapsed := now.Sub(started)

	if ctxErr == context.DeadlineExceeded {
		_ = el.Append(event.Event{
			TS:      now,
			Event:   SpawnTimeout,
			Ticket:  tk.ID,
			Project: tk.Project,
			Actor:   "st",
			RunID:   runID,
			Data: map[string]any{
				"pid":     pid,
				"elapsed": elapsed.String(),
			},
		})
		return fmt.Errorf("worker timed out after %s", elapsed)
	}

	if waitErr != nil || exitCode != 0 {
		_ = el.Append(event.Event{
			TS:      now,
			Event:   SpawnFailed,
			Ticket:  tk.ID,
			Project: tk.Project,
			Actor:   "st",
			RunID:   runID,
			Data: map[string]any{
				"pid":       pid,
				"exit_code": exitCode,
				"elapsed":   elapsed.String(),
			},
		})
		if waitErr != nil {
			return fmt.Errorf("worker failed: %w", waitErr)
		}
		return fmt.Errorf("worker exited with code %d", exitCode)
	}

	_ = el.Append(event.Event{
		TS:      now,
		Event:   SpawnCompleted,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   "st",
		RunID:   runID,
		Data: map[string]any{
			"pid":       pid,
			"exit_code": 0,
			"elapsed":   elapsed.String(),
		},
	})
	return nil
}

// tmuxPanePID returns the PID of the process in a tmux window's active pane.
func tmuxPanePID(tmuxPath, windowName string) int {
	out, err := exec.Command(tmuxPath, "list-panes", "-t", windowName, "-F", "#{pane_pid}").Output()
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return pid
}

// readExitCodeFile reads an exit code written to a file by the worker shell command.
func readExitCodeFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1
	}
	return code
}

// Event type constants for spawn events.
const (
	SpawnStarted   = "spawn.started"
	SpawnCompleted = "spawn.completed"
	SpawnFailed    = "spawn.failed"
	SpawnTimeout   = "spawn.timeout"
)
