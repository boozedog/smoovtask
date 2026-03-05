package spawn

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestBuildPrompt(t *testing.T) {
	tk := &ticket.Ticket{
		ID:    "st_abc123",
		Title: "Fix login bug",
		Body:  "## Description\nThe login form doesn't validate emails.\n",
	}
	runID := "spawn-test123"

	repoRoot := "/fake/repo"
	prompt := BuildPrompt(tk, runID, repoRoot)

	// Should contain ticket info
	if !strings.Contains(prompt, "st_abc123") {
		t.Error("prompt should contain ticket ID")
	}
	if !strings.Contains(prompt, "Fix login bug") {
		t.Error("prompt should contain ticket title")
	}

	// Should contain ticket body
	if !strings.Contains(prompt, "login form doesn't validate") {
		t.Error("prompt should contain ticket body")
	}

	// Should contain --file and --ticket in note instruction
	if !strings.Contains(prompt, "--file <path>") {
		t.Error("prompt should contain --file flag in note instruction")
	}
	if !strings.Contains(prompt, "--ticket st_abc123 --run-id spawn-test123") {
		t.Error("prompt should contain st note command with ticket ID and run ID")
	}
	if !strings.Contains(prompt, "st status review --ticket st_abc123 --run-id spawn-test123") {
		t.Error("prompt should contain st status review command")
	}

	// Should not contain push instructions
	if !strings.Contains(prompt, "Do not push to remote") {
		t.Error("prompt should tell worker not to push")
	}
}

func TestBuildPromptEmptyBody(t *testing.T) {
	tk := &ticket.Ticket{
		ID:    "st_xyz789",
		Title: "Add tests",
	}

	prompt := BuildPrompt(tk, "run-123", "/fake/repo")

	if strings.Contains(prompt, "## Ticket Context") {
		t.Error("prompt should not contain ticket context section when body is empty")
	}
	if !strings.Contains(prompt, "## Instructions") {
		t.Error("prompt should always contain instructions section")
	}
}
