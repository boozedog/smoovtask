package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/workflow"
	"github.com/spf13/cobra"
)

var overrideCmd = &cobra.Command{
	Use:   "override <ticket-id> <status>",
	Short: "Force-set ticket status (human override, bypasses rules)",
	Args:  cobra.ExactArgs(2),
	RunE:  runOverride,
}

func init() {
	rootCmd.AddCommand(overrideCmd)
}

func runOverride(_ *cobra.Command, args []string) error {
	id := args[0]

	targetStatus, err := workflow.StatusFromAlias(strings.ToLower(args[1]))
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticketsDir, err := cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	store := ticket.NewStore(ticketsDir)

	tk, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("get ticket: %w", err)
	}

	now := time.Now().UTC()
	oldStatus := tk.Status
	tk.Status = targetStatus
	tk.PriorStatus = nil
	tk.Updated = now

	actor := identity.Actor()
	sessionID := identity.SessionID()
	content := fmt.Sprintf("%s → %s", oldStatus, targetStatus)
	ticket.AppendSection(tk, "Override", actor, sessionID, content, nil, now)

	if err := store.Save(tk); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	// Log event
	eventsDir, err := cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}
	el := event.NewEventLog(eventsDir)
	_ = el.Append(event.Event{
		TS:      now,
		Event:   "status.override",
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		Session: sessionID,
		Data: map[string]any{
			"from":   string(oldStatus),
			"to":     string(targetStatus),
			"reason": "override",
		},
	})

	fmt.Printf("Override %s: %s → %s\n", tk.ID, oldStatus, targetStatus)

	return nil
}
