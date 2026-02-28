package cmd

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestNote_OnCurrentTicket(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "note target", ticket.StatusInProgress)
	tk.Assignee = "test-session-note"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", "this is a note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Note added to "+tk.ID)
	}

	// Verify note appears in ticket body
	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "this is a note") {
		t.Errorf("body = %q, want note content in body", updated.Body)
	}

	// Verify event logged
	events, err := event.QueryEvents(env.EventsDir, event.Query{TicketID: tk.ID})
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Event == event.TicketNote {
			found = true
			break
		}
	}
	if !found {
		t.Error("no ticket.note event logged")
	}
}

func TestNote_WithTicketFlag(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "flagged note target", ticket.StatusOpen)

	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", "--ticket", tk.ID, "flagged note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Note added to "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "flagged note") {
		t.Errorf("body = %q, want note content in body", updated.Body)
	}
}

func TestNote_NoActiveTicketAndNoFlag(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	// No tickets assigned to this session
	_, err := env.runCmd(t, "--run-id", "test-session-note", "note", "orphan note")
	if err == nil {
		t.Fatal("expected error when no active ticket and no --ticket flag")
	}
}

func TestNote_WithPositionalTicketID(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "positional note target", ticket.StatusOpen)

	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", tk.ID, "positional note message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tk.ID) {
		t.Errorf("output = %q, want substring %q", out, "Note added to "+tk.ID)
	}

	updated, err := env.Store.Get(tk.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "positional note message") {
		t.Errorf("body = %q, want note content in body", updated.Body)
	}
}

func TestNote_PositionalTicketID_FlagTakesPrecedence(t *testing.T) {
	env := newTestEnv(t)

	tkFlag := env.createTicket(t, "flag target", ticket.StatusOpen)
	tkPos := env.createTicket(t, "positional target", ticket.StatusOpen)

	// --ticket flag should take precedence over positional arg
	out, err := env.runCmd(t, "--run-id", "test-session-note", "note", "--ticket", tkFlag.ID, tkPos.ID, "flag wins message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note added to "+tkFlag.ID) {
		t.Errorf("output = %q, want note added to flag ticket %s", out, tkFlag.ID)
	}

	updated, err := env.Store.Get(tkFlag.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !strings.Contains(updated.Body, "flag wins message") {
		t.Errorf("flag ticket body = %q, want note content", updated.Body)
	}
}

func TestNote_SingleArgTicketIDErrors(t *testing.T) {
	env := newTestEnv(t)

	tk := env.createTicket(t, "target", ticket.StatusInProgress)
	tk.Assignee = "test-session-note"
	if err := env.Store.Save(tk); err != nil {
		t.Fatalf("save ticket: %v", err)
	}

	// Single arg that looks like a ticket ID should error helpfully
	_, err := env.runCmd(t, "--run-id", "test-session-note", "note", tk.ID)
	if err == nil {
		t.Fatal("expected error when single arg looks like a ticket ID")
	}
	if !strings.Contains(err.Error(), "looks like a ticket ID") {
		t.Errorf("error = %q, want helpful message about ticket ID", err.Error())
	}
}

func TestNote_NoRunID(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	// No --run-id and no --human
	_, err := env.runCmd(t, "note", "no session note")
	if err == nil {
		t.Fatal("expected error when no --run-id is provided")
	}
}

func TestUnescapeNote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text unchanged",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "basic newline",
			in:   `line1\nline2`,
			want: "line1\nline2",
		},
		{
			name: "multiple newlines",
			in:   `a\nb\nc`,
			want: "a\nb\nc",
		},
		{
			name: "inline code preserved",
			in:   `before\n` + "`" + `code\nhere` + "`" + `\nafter`,
			want: "before\n`code\\nhere`\nafter",
		},
		{
			name: "fenced code block preserved",
			in:   "before\\n```\\nfenced\\n```\\nafter",
			want: "before\n```\\nfenced\\n```\nafter",
		},
		{
			name: "no false positive on backslash other",
			in:   `hello\tworld`,
			want: `hello\tworld`,
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "trailing backslash n",
			in:   `end\n`,
			want: "end\n",
		},
		{
			name: "unclosed inline code",
			in:   "before\\n`code\\nstuff",
			want: "before\n`code\\nstuff",
		},
		{
			name: "unclosed fenced block",
			in:   "before\\n```code\\nstuff",
			want: "before\n```code\\nstuff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeNote(tt.in)
			if got != tt.want {
				t.Errorf("unescapeNote(%q)\n got %q\nwant %q", tt.in, got, tt.want)
			}
		})
	}
}
