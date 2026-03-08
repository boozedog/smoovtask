package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/boozedog/smoovtask/internal/config"
)

const (
	roleLeader      = "leader"
	roleImplementer = "implementer"
	roleReviewer    = "reviewer"
	roleWorker      = "worker"
)

var launchInTmux = launchRoleInTmux

var tmuxNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func launchSession(role, ticketID, cliOverride string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cliName, err := resolveCLIName(cfg, cliOverride)
	if err != nil {
		return err
	}

	if err := launchInTmux(role, ticketID, cliName); err != nil {
		return err
	}

	return nil
}

func resolveCLIName(_ *config.Config, cliOverride string) (string, error) {
	cliName := strings.TrimSpace(cliOverride)
	if cliName == "" {
		cliName = "claude"
	}

	switch cliName {
	case "claude", "opencode", "pi":
		return cliName, nil
	default:
		return "", fmt.Errorf("unknown cli %q (supported: claude, opencode, pi)", cliName)
	}
}

func launchRoleInTmux(role, ticketID, cliName string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}

	cliPath, err := exec.LookPath(cliName)
	if err != nil {
		return fmt.Errorf("%s CLI not found in PATH: %w", cliName, err)
	}

	sessionName := tmuxSessionName(role, ticketID)
	tmuxArgs := []string{"new-session", "-A", "-s", sessionName, "env", "ST_ROLE=" + role}
	if ticketID != "" {
		tmuxArgs = append(tmuxArgs, "ST_REVIEW_TICKET="+ticketID)
	}
	tmuxArgs = append(tmuxArgs, cliPath)

	cmd := exec.Command(tmuxPath, tmuxArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launch %s session in tmux: %w", role, err)
	}

	return nil
}

func tmuxSessionName(role, ticketID string) string {
	base := "st-" + role
	if ticketID == "" {
		return base
	}
	return base + "-" + sanitizeTmuxName(ticketID)
}

func sanitizeTmuxName(s string) string {
	s = strings.TrimSpace(s)
	s = tmuxNameSanitizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "session"
	}
	return s
}
