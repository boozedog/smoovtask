package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/identity"
	"github.com/boozedog/smoovtask/internal/spawn"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note [ticket-id] [message]",
	Short: "Append a note to the current ticket",
	Long: `Append a note to the current ticket.

The message can be provided as a positional argument, or via a drop file at
.st/notes/<run-id>.md. When no message argument is given, st note looks for
the drop file, reads its content, and deletes it after appending.`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runNote,
}

var noteTicket string

func init() {
	noteCmd.Flags().StringVar(&noteTicket, "ticket", "", "ticket ID (default: current ticket)")
	rootCmd.AddCommand(noteCmd)
}

// noteFilePath returns the absolute path to the drop file for file-based note input.
// The file is at .st/notes/<run-id>.md under the main repo root (not a worktree).
func noteFilePath(runID string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	root, err := spawn.WorktreeRepoRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("st note requires a git repository: %w", err)
	}
	return filepath.Join(root, ".st", "notes", runID+".md"), nil
}

// readNoteFile reads and removes the drop file, returning its content.
func readNoteFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// Remove the file after reading — best effort, don't fail the note.
	_ = os.Remove(path)
	return string(data), nil
}

func runNote(_ *cobra.Command, args []string) error {
	// Support: st note <message>, st note <ticket-id> <message>, or st note (file-based)
	var message string
	ticketFlag := noteTicket
	switch len(args) {
	case 2:
		if ticketFlag == "" {
			ticketFlag = args[0]
		}
		message = args[1]
	case 1:
		// Single arg — if --ticket is not set and arg looks like a ticket ID, error helpfully
		if ticketFlag == "" && ticket.LooksLikeTicketID(args[0]) {
			return fmt.Errorf("argument %q looks like a ticket ID — did you mean: st note %s \"<message>\"?", args[0], args[0])
		}
		message = args[0]
	case 0:
		// No args — try reading from drop file
		runID := identity.RunID()
		if runID == "" {
			return fmt.Errorf("--run-id is required when using file-based notes (no message argument provided)")
		}
		path, err := noteFilePath(runID)
		if err != nil {
			return err
		}
		content, err := readNoteFile(path)
		if err != nil {
			return fmt.Errorf("no message argument and no note file found at %s", path)
		}
		message = strings.TrimSpace(content)
		if message == "" {
			return fmt.Errorf("note file %s is empty", path)
		}
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
	runID := identity.RunID()
	actor := identity.Actor()

	tk, err := resolveCurrentTicket(store, cfg, runID, ticketFlag)
	if err != nil {
		return err
	}

	message = unescapeNote(message)

	now := time.Now().UTC()
	ticket.AppendSection(tk, "Note", actor, runID, message, nil, now)

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
		Event:   event.TicketNote,
		Ticket:  tk.ID,
		Project: tk.Project,
		Actor:   actor,
		RunID:   runID,
		Data:    map[string]any{"message": message},
	})

	fmt.Printf("Note added to %s\n", tk.ID)
	return nil
}

// unescapeNote converts literal \n sequences to real newlines in note text,
// but preserves them inside inline code (`...`) and fenced code blocks (```...```).
func unescapeNote(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		// Fenced code block: ```
		if i+2 < len(s) && s[i] == '`' && s[i+1] == '`' && s[i+2] == '`' {
			end := strings.Index(s[i+3:], "```")
			if end >= 0 {
				end += i + 3 + 3 // past closing ```
				b.WriteString(s[i:end])
				i = end
				continue
			}
			// Unclosed fenced block — write rest verbatim
			b.WriteString(s[i:])
			return b.String()
		}

		// Inline code: `
		if s[i] == '`' {
			end := strings.IndexByte(s[i+1:], '`')
			if end >= 0 {
				end += i + 1 + 1 // past closing `
				b.WriteString(s[i:end])
				i = end
				continue
			}
			// Unclosed inline code — write rest verbatim
			b.WriteString(s[i:])
			return b.String()
		}

		// Literal \n outside code
		if i+1 < len(s) && s[i] == '\\' && s[i+1] == 'n' {
			b.WriteByte('\n')
			i += 2
			continue
		}

		b.WriteByte(s[i])
		i++
	}

	return b.String()
}
