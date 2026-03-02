package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestResolveCLIName_DefaultsToConfig(t *testing.T) {
	name, err := resolveCLIName(&config.Config{Agent: config.AgentConfig{CLI: "claude"}}, "")
	if err != nil {
		t.Fatalf("resolveCLIName() error = %v", err)
	}
	if name != "claude" {
		t.Fatalf("resolveCLIName() = %q, want %q", name, "claude")
	}
}

func TestResolveCLIName_OverrideWins(t *testing.T) {
	name, err := resolveCLIName(&config.Config{Agent: config.AgentConfig{CLI: "claude"}}, "opencode")
	if err != nil {
		t.Fatalf("resolveCLIName() error = %v", err)
	}
	if name != "opencode" {
		t.Fatalf("resolveCLIName() = %q, want %q", name, "opencode")
	}
}

func TestResolveCLIName_Unknown(t *testing.T) {
	_, err := resolveCLIName(&config.Config{Agent: config.AgentConfig{CLI: "claude"}}, "bogus")
	if err == nil {
		t.Fatal("expected error for unknown cli")
	}
}

func TestTmuxSessionName(t *testing.T) {
	got := tmuxSessionName(roleReviewer, "st_ABC-123")
	if got != "st-reviewer-st_ABC-123" {
		t.Fatalf("tmuxSessionName() = %q", got)
	}

	got = tmuxSessionName(roleReviewer, "ticket / with spaces")
	if got != "st-reviewer-ticket-with-spaces" {
		t.Fatalf("tmuxSessionName() with spaces = %q", got)
	}
}

func TestLeader_LaunchesRoleSession(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	orig := launchInTmux
	t.Cleanup(func() { launchInTmux = orig })

	called := false
	launchInTmux = func(role, ticketID, cliName string) error {
		called = true
		if role != roleLeader {
			t.Fatalf("role = %q, want %q", role, roleLeader)
		}
		if ticketID != "" {
			t.Fatalf("ticketID = %q, want empty", ticketID)
		}
		if cliName != "claude" {
			t.Fatalf("cliName = %q, want %q", cliName, "claude")
		}
		return nil
	}

	if _, err := env.runCmd(t, "leader"); err != nil {
		t.Fatalf("run leader: %v", err)
	}
	if !called {
		t.Fatal("launchInTmux not called")
	}
}

func TestWork_CLIOverride(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	orig := launchInTmux
	t.Cleanup(func() { launchInTmux = orig })

	launchInTmux = func(role, ticketID, cliName string) error {
		if role != roleImplementer {
			t.Fatalf("role = %q, want %q", role, roleImplementer)
		}
		if cliName != "opencode" {
			t.Fatalf("cliName = %q, want %q", cliName, "opencode")
		}
		return nil
	}

	if _, err := env.runCmd(t, "work", "--cli", "opencode"); err != nil {
		t.Fatalf("run work: %v", err)
	}
}

func TestReview_LauncherModeRequiresTicket(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := env.runCmd(t, "review")
	if err == nil {
		t.Fatal("expected error when launching review without ticket")
	}
	if !strings.Contains(err.Error(), "requires a ticket ID") {
		t.Fatalf("error = %q, want ticket ID error", err.Error())
	}
}

func TestReview_LauncherModeUsesReviewerRole(t *testing.T) {
	env := newTestEnv(t)
	tk := env.createTicket(t, "review launcher target", ticket.StatusOpen)

	orig := launchInTmux
	t.Cleanup(func() { launchInTmux = orig })

	called := false
	launchInTmux = func(role, ticketID, cliName string) error {
		called = true
		if role != roleReviewer {
			t.Fatalf("role = %q, want %q", role, roleReviewer)
		}
		if ticketID != tk.ID {
			t.Fatalf("ticketID = %q, want %q", ticketID, tk.ID)
		}
		if cliName != "claude" {
			t.Fatalf("cliName = %q, want %q", cliName, "claude")
		}
		return nil
	}

	if _, err := env.runCmd(t, "review", tk.ID); err != nil {
		t.Fatalf("run review launcher: %v", err)
	}
	if !called {
		t.Fatal("launchInTmux not called")
	}
}
